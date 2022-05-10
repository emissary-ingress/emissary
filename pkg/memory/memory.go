package memory

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/datawire/ambassador/v2/pkg/debug"
	"github.com/datawire/dlib/dlog"
)

// The MemoryUsage struct to holds memory usage and memory limit information about a cgroup.
type MemoryUsage struct {
	usage memory
	limit memory

	debug bool

	perProcess map[int]*ProcessUsage
	previous   memory
	lastAction time.Time

	// these allow mocking for tests
	readUsage      func(context.Context) (memory, memory)
	readPerProcess func(context.Context) map[int]*ProcessUsage

	// Protects the whole structure
	mutex sync.Mutex
}

// The ProcessUsage struct holds per process memory usage information.
type ProcessUsage struct {
	Pid     int
	Cmdline []string
	Usage   memory

	// This is zero if the process is still running. If the process has exited, this counts how many
	// refreshes have happened. We GC after 10 refreshes.
	RefreshesSinceExit int
}

type memory int64

const GiB = 1024 * 1024 * 1024

// The Watch method will check memory usage every 10 seconds and log it if it jumps more than 10Gi
// up or down. Additionally if memory usage exceeds 50% of the cgroup limit, it will log usage every
// minute. Usage is also unconditionally logged before returning. This function only returns if the
// context is canceled.
func (usage *MemoryUsage) Watch(ctx context.Context) {
	dbg := debug.FromContext(ctx)
	memory := dbg.Value("memory")
	memory.Store(usage.ShortString())

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case now := <-ticker.C:
			usage.Refresh(ctx)
			memory.Store(usage.ShortString())
			usage.maybeDo(now, func() {
				if usage.debug {
					dlog.Infoln(ctx, usage.Summary())
				} else {
					dlog.Infoln(ctx, usage.String())
				}
			})
		case <-ctx.Done():
			usage.Refresh(ctx)
			dlog.Infoln(ctx, usage.String())
			return
		}
	}
}

func (m *MemoryUsage) ShortString() string {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return fmt.Sprintf("%s of %s (%d%%)", m.usage.String(), m.limit.String(), m.percentUsed())
}

// Return true if conditions for action are satisifed. We take action if memory has changed more
// than 10Gi since our previous action. We also take action once per minute if usage is greather
// than 50% of our limit.
func (m *MemoryUsage) shouldDo(now time.Time) bool {
	if m.debug && (now.Sub(m.lastAction) >= 30*time.Second) {
		return true
	}

	const jump = 10 * 1024 * 1024
	delta := m.previous - m.usage
	if delta >= jump || delta <= -jump {
		return true
	}

	if m.percentUsed() > 50 && now.Sub(m.lastAction) >= 60*time.Second {
		return true
	}

	return false
}

// Do something if warranted.
func (m *MemoryUsage) maybeDo(now time.Time, f func()) {
	m.mutex.Lock()
	if m.shouldDo(now) {
		m.previous = m.usage
		m.lastAction = now
		m.mutex.Unlock()
		f()
	} else {
		m.mutex.Unlock()
	}
}

// The GetMemoryUsage function returns MemoryUsage info for the entire cgroup.
func GetMemoryUsage(ctx context.Context) *MemoryUsage {
	usage, limit := readUsage(ctx)

	debug, _ := strconv.ParseBool(os.Getenv("AMBASSADOR_MEMORY_DEBUG"))

	return &MemoryUsage{
		usage: usage,
		limit: limit,

		debug: debug,

		perProcess: readPerProcess(ctx),

		readUsage:      readUsage,
		readPerProcess: readPerProcess,
	}
}

// Convert memory to (float64) GiB.
func (m memory) GiB() float64 {
	if m == unlimited {
		return math.Inf(1)
	}

	return float64(m) / GiB
}

// Pretty print memory in gigabytes.
func (m memory) String() string {
	if m == unlimited {
		return "Unlimited"
	}

	return fmt.Sprintf("%.2fGi", m.GiB())
}

// The MemoryUsage.Refresh method updates memory usage information.
func (m *MemoryUsage) Refresh(ctx context.Context) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	usage, limit := m.readUsage(ctx)
	m.usage = usage
	m.limit = limit

	// GC process memory info that has been around for more than 10 refreshes.
	for pid, usage := range m.perProcess {
		if usage.RefreshesSinceExit > 10 {
			// It's old, let's delete it.
			delete(m.perProcess, pid)
		} else {

			// Increment the count in case the process has exited. If the process is still running,
			// this whole entry will get overwritted with a new one in the loop that follows this
			// one.
			usage.RefreshesSinceExit += 1
		}
	}

	for pid, usage := range m.readPerProcess(ctx) {
		// Overwrite any old process info with new/updated process info.
		m.perProcess[pid] = usage
	}
}

// If there is no cgroups memory limit then the value in
// /sys/fs/cgroup/memory/memory.limit_in_bytes will be math.MaxInt64 rounded down to
// the nearest pagesize. We calculate this number so we can detect if there is no memory limit.
var unlimited memory = (memory(math.MaxInt64) / memory(os.Getpagesize())) * memory(os.Getpagesize())

// Pretty print a summary of memory usage suitable for logging.
func (m *MemoryUsage) String() string {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var msg strings.Builder
	if m.limit == unlimited {
		msg.WriteString(fmt.Sprintf("Memory Usage %s", m.usage.String()))
	} else {
		msg.WriteString(fmt.Sprintf("Memory Usage %s (%d%%)", m.usage.String(), m.percentUsed()))
	}

	pids := make([]int, 0, len(m.perProcess))
	for pid := range m.perProcess {
		pids = append(pids, pid)
	}

	sort.Ints(pids)

	for _, pid := range pids {
		usage := m.perProcess[pid]
		msg.WriteString("\n  ")
		msg.WriteString(usage.String())
	}

	return msg.String()
}

func (m *MemoryUsage) Summary() string {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	lstr := ""

	if m.limit != unlimited {
		lstr = fmt.Sprintf(" (%d%%)", m.percentUsed())
	}

	cmds := make(map[string]float64, len(m.perProcess))
	sum := 0.0

	for pid := range m.perProcess {
		usage := m.perProcess[pid]

		cmdLine := usage.Cmdline
		cmd := strings.Join(cmdLine, " ")

		cmdName := "other"

		switch {
		case usage.RefreshesSinceExit > 0:
			cmdName = "exited"

		case (cmdLine[0] == "busyambassador") && (cmdLine[1] == "entrypoint"):
			cmdName = "entry"

		case (cmdLine[0] == "/usr/bin/python") && (cmdLine[1] == "/usr/bin/diagd"):
			cmdName = "diagd"

		case cmdLine[0] == "/ambassador/sidecars/amb-sidecar":
			cmdName = "sidecr"

		case cmdLine[0] == "envoy":
			if strings.Contains(cmd, "--mode validate") {
				cmdName = "vldatr"
			} else {
				cmdName = "envoy"
			}

		case strings.Contains(cmd, "watt --listen"):
			cmdName = "watt"

		case strings.Contains(cmd, "post_update"):
			cmdName = "post_update"

		case strings.Contains(cmd, "kubestatus"):
			cmdName = "kubestatus"
		}

		cmds[cmdName] += usage.Usage.GiB()
		sum += usage.Usage.GiB()
	}

	var msg strings.Builder

	msg.WriteString(fmt.Sprintf("Memory Summary %.2fGi calc %s rept%s", sum, m.usage.String(), lstr))

	cmdNames := make([]string, 0, len(cmds))

	for cmd := range cmds {
		if cmd == "exited" {
			continue
		}

		cmdNames = append(cmdNames, cmd)
	}

	sort.Strings(cmdNames)

	for _, cmd := range cmdNames {
		msg.WriteString(fmt.Sprintf(" %s %.2fGi", cmd, cmds[cmd]))
	}

	usage := cmds["exited"]

	if usage > 0 {
		msg.WriteString(fmt.Sprintf(" exited %.2fGi", usage))
	}

	return msg.String()
}

// Pretty print a summary of process memory usage.
func (pu ProcessUsage) String() string {
	status := ""
	if pu.RefreshesSinceExit > 0 {
		status = " (exited)"
	}
	return fmt.Sprintf("  PID %d, %s%s: %s", pu.Pid, pu.Usage.String(), status, strings.Join(pu.Cmdline, " "))
}

// The MemoryUsage.PercentUsed method returns memory usage as a percentage of memory limit.
func (m *MemoryUsage) PercentUsed() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.percentUsed()
}

// This the same as PercentUsed() but not protected by a lock so we can use it form places where we
// already have the lock.
func (m *MemoryUsage) percentUsed() int {
	return int(float64(m.usage) / float64(m.limit) * 100)
}

// The GetCmdline helper returns the command line for a pid. If the pid does not exist or we don't
// have access to read /proc/<pid>/cmdline, then it returns the empty string.
func GetCmdline(ctx context.Context, pid int) []string {
	bytes, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		if errors.Is(err, os.ErrPermission) || errors.Is(err, os.ErrNotExist) {
			// Don't complain if we don't have permission or the info doesn't exist.
			return nil
		}
		dlog.Errorf(ctx, "couldn't access cmdline for %d: %v", pid, err)
		return nil
	}
	return strings.Split(strings.TrimSuffix(string(bytes), "\n"), "\x00")
}

// Helper to read the usage and limit for the cgroup.
func readUsage(ctx context.Context) (memory, memory) {
	limit, err := readMemory("/sys/fs/cgroup/memory/memory.limit_in_bytes")
	if err != nil {
		if errors.Is(err, os.ErrPermission) || errors.Is(err, os.ErrNotExist) {
			// Don't complain if we don't have permission or the info doesn't exist.
			return 0, unlimited
		}
		dlog.Errorf(ctx, "couldn't access memory limit: %v", err)
		return 0, unlimited
	}

	stats, err := readMemoryStat("/sys/fs/cgroup/memory/memory.stat")
	if err != nil {
		if errors.Is(err, os.ErrPermission) || errors.Is(err, os.ErrNotExist) {
			// Don't complain if we don't have permission or the info doesn't exist.
			return 0, limit
		}
		dlog.Errorf(ctx, "couldn't access memory usage: %v", err)
		return 0, limit
	}

	// We calculate memory usage according to the OOMKiller as (rss + cache + swap) - inactive_file.
	// This is substantiated by this article[1] which claims we need to track container_memory_working_set_bytes.
	// According to this stack overflow[2], container_memory_working_set_bytes is "total usage" - "inactive file".
	// Best as I can tell from the cgroup docs[3], "total usage" is computed from memory.stat by
	// adding (rss + cache + swap), and "inactive file" is just the inactive_file field.
	//
	// [1]: https://faun.pub/how-much-is-too-much-the-linux-oomkiller-and-used-memory-d32186f29c9d
	// [2]: https://stackoverflow.com/questions/65428558/what-is-the-difference-between-container-memory-working-set-bytes-and-contain
	// [3]: https://www.kernel.org/doc/Documentation/cgroup-v1/memory.txt

	totalUsage := stats.Rss + stats.Cache + stats.Swap
	OOMUsage := totalUsage - stats.InactiveFile
	return memory(OOMUsage), limit
}

// Read an int64 from a file and convert it to memory.
func readMemory(fpath string) (memory, error) {
	contentAsB, err := ioutil.ReadFile(fpath)
	if err != nil {
		return 0, err
	}
	contentAsStr := strings.TrimSuffix(string(contentAsB), "\n")
	m, err := strconv.ParseInt(contentAsStr, 10, 64)
	return memory(m), err
}

// memoryStat represents the contents of /sys/fs/cgroup/memory/memory.stat.
// It's the data structure returned by readMemoryStat.
type memoryStat struct {
	Rss          uint64 // rss field
	Cache        uint64 // cache field
	Swap         uint64 // swap field
	InactiveFile uint64 // inactive_file field
}

func readMemoryStat(fpath string) (memoryStat, error) {
	bytes, err := ioutil.ReadFile(fpath)
	if err != nil {
		return memoryStat{}, err
	}

	return parseMemoryStat(string(bytes))
}

func parseMemoryStat(content string) (memoryStat, error) {
	result := memoryStat{}
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSuffix(line, "\n")
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}

		n, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return result, err
		}

		switch parts[0] {
		case "rss":
			result.Rss = n
		case "swap":
			result.Swap = n
		case "cache":
			result.Cache = n
		case "inactive_file":
			result.InactiveFile = n
		}
	}
	return result, nil
}

// The readPerProcess helper returns a map containing memory usage used for each process in the cgroup.
func readPerProcess(ctx context.Context) map[int]*ProcessUsage {
	result := map[int]*ProcessUsage{}

	files, err := ioutil.ReadDir("/proc")
	if err != nil {
		dlog.Errorf(ctx, "could not access memory info: %v", err)
		return nil
	}

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(file.Name())
		if err != nil {
			continue
		}

		bytes, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/smaps_rollup", pid))
		if err != nil {
			if errors.Is(err, os.ErrPermission) || errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ESRCH) {
				// Don't complain if we don't have permission or the info doesn't exist.
				continue
			}
			dlog.Errorf(ctx, "couldn't access usage for %d: %v", pid, err)
			continue
		}

		parts := strings.Fields(string(bytes))
		rssStr := ""
		for idx, field := range parts {
			if field == "Rss:" {
				rssStr = parts[idx+1]
			}
		}
		if rssStr == "" {
			continue
		}
		rss, err := strconv.ParseUint(rssStr, 10, 64)
		if err != nil {
			dlog.Errorf(ctx, "couldn't parse %s: %v", rssStr, err)
			continue
		}
		rss = rss * 1024
		result[pid] = &ProcessUsage{pid, GetCmdline(ctx, pid), memory(rss), 0}
	}

	return result
}

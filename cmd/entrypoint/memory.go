package entrypoint

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// The watchMemory function will check memory usage every 10 seconds and log it if it jumps more
// than 10Gi up or down. Additionally if memory usage exceeds 50% of the cgroup limit, it will log
// usage every minute. Usage is also unconditionally logged before returning. This function only
// returns if the context is canceled.
func watchMemory(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	usage := GetMemoryUsage()
	for {
		select {
		case now := <-ticker.C:
			usage.Refresh()
			usage.maybeDo(now, func() {
				log.Println(usage.String())
			})
		case <-ctx.Done():
			usage.Refresh()
			log.Println(usage.String())
			return
		}
	}
}

// Return true if conditions for action are satisifed. We take action if memory has changed more
// than 10Gi since our previous action. We also take action once per minute if usage is greather
// than 50% of our limit.
func (m *MemoryUsage) shouldDo(now time.Time) bool {
	const jump = 10 * 1024 * 1024
	delta := m.previous - m.Usage
	if delta >= jump || delta <= -jump {
		return true
	}

	if m.PercentUsed() > 50 && now.Sub(m.lastAction) >= 60*time.Second {
		return true
	}

	return false
}

// Do something if warranted.
func (m *MemoryUsage) maybeDo(now time.Time, f func()) {
	if m.shouldDo(now) {
		m.previous = m.Usage
		m.lastAction = now
		f()
	}
}

// The GetMemoryUsage function returns MemoryUsage info for the entire cgroup.
func GetMemoryUsage() *MemoryUsage {
	usage, limit := readUsage()
	return &MemoryUsage{usage, limit, perProcess(), 0, time.Time{}, readUsage, perProcess}
}

// The MemoryUsage struct to holds memory usage and memory limit information about a cgroup.
type MemoryUsage struct {
	Usage      memory
	Limit      memory
	PerProcess map[int]*ProcessUsage
	previous   memory
	lastAction time.Time

	// these allow mocking for tests
	readUsage  func() (memory, memory)
	perProcess func() map[int]*ProcessUsage
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

// Pretty print memory in gigabytes.
func (m memory) String() string {
	const GiB = 1024 * 1024 * 1024
	return fmt.Sprintf("%.2fGi", float64(m)/GiB)
}

// The MemoryUsage.Refresh method updates memory usage information.
func (m *MemoryUsage) Refresh() {
	usage, limit := m.readUsage()
	m.Usage = usage
	m.Limit = limit

	// GC process memory info that has been around for more than 10 refreshes.
	for pid, usage := range m.PerProcess {
		if usage.RefreshesSinceExit > 10 {
			// It's old, let's delete it.
			delete(m.PerProcess, pid)
		} else {

			// Increment the count in case the process has exited. If the process is still running,
			// this whole entry will get overwritted with a new one in the loop that follows this
			// one.
			usage.RefreshesSinceExit += 1
		}
	}

	for pid, usage := range m.perProcess() {
		// Overwrite any old process info with new/updated process info.
		m.PerProcess[pid] = usage
	}
}

// If there is no cgroups memory limit then the value in
// /proc/%d/root/sys/fs/cgroup/memory/memory.limit_in_bytes will be math.MaxInt64 rounded down to
// the nearest pagesize. We calculate this number so we can detect if there is no memory limit.
var unlimited memory = (memory(math.MaxInt64) / memory(os.Getpagesize())) * memory(os.Getpagesize())

// Pretty print a summary of memory usage suitable for logging.
func (m MemoryUsage) String() string {
	var msg strings.Builder
	if m.Limit == unlimited {
		msg.WriteString(fmt.Sprintf("Memory Usage %s", m.Usage.String()))
	} else {
		msg.WriteString(fmt.Sprintf("Memory Usage %s (%d%%)", m.Usage.String(), m.PercentUsed()))
	}

	pids := make([]int, 0, len(m.PerProcess))
	for pid := range m.PerProcess {
		pids = append(pids, pid)
	}

	sort.Ints(pids)

	for _, pid := range pids {
		usage := m.PerProcess[pid]
		msg.WriteString("\n  ")
		msg.WriteString(usage.String())
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
func (m MemoryUsage) PercentUsed() int {
	return int(float64(m.Usage) / float64(m.Limit) * 100)
}

// The GetCmdline helper returns the command line for a pid. If the pid does not exist or we don't
// have access to read /proc/<pid>/cmdline, then it returns the empty string.
func GetCmdline(pid int) []string {
	bytes, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		if errors.Is(err, os.ErrPermission) || errors.Is(err, os.ErrNotExist) {
			// Don't complain if we don't have permission or the info doesn't exist.
			return nil
		}
		log.Printf("couldn't access cmdline for %d: %v", pid, err)
		return nil
	}
	return strings.Split(strings.TrimSuffix(string(bytes), "\n"), "\x00")
}

// Helper to read the usage and limit for the cgroup.
func readUsage() (memory, memory) {
	pid := os.Getpid()

	limit, err := readMemory(fmt.Sprintf("/proc/%d/root/sys/fs/cgroup/memory/memory.limit_in_bytes", pid))
	if err != nil {
		if errors.Is(err, os.ErrPermission) || errors.Is(err, os.ErrNotExist) {
			// Don't complain if we don't have permission or the info doesn't exist.
			return 0, unlimited
		}
		log.Printf("couldn't access limit for %d: %v", pid, err)
		return 0, unlimited
	}
	usage, err := readMemory(fmt.Sprintf("/proc/%d/root/sys/fs/cgroup/memory/memory.usage_in_bytes", pid))
	if err != nil {
		if errors.Is(err, os.ErrPermission) || errors.Is(err, os.ErrNotExist) {
			// Don't complain if we don't have permission or the info doesn't exist.
			return 0, limit
		}
		log.Printf("couldn't access usage for %d: %v", pid, err)
		return 0, limit
	}

	return usage, limit
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

// The perProcess helper returns a map containing memory usage used for each process in the cgroup.
func perProcess() map[int]*ProcessUsage {
	result := map[int]*ProcessUsage{}

	files, err := ioutil.ReadDir("/proc")
	if err != nil {
		log.Printf("could not access memory info: %v", err)
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
			log.Printf("couldn't access usage for %d: %v", pid, err)
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
			log.Printf("couldn't parse %s: %v", rssStr, err)
			continue
		}
		rss = rss * 1024
		result[pid] = &ProcessUsage{pid, GetCmdline(pid), memory(rss), 0}
	}

	return result
}

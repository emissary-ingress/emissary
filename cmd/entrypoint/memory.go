package entrypoint

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

// The LogMemoryUsage function logs the memory usage of all processes whose usage as a percentage of
// limit exceeds the supplied threshold.
func LogMemoryUsage(threshold int) {
	for _, mu := range GetAllMemoryUsage() {
		if mu.PercentUsed() >= threshold {
			log.Println(mu.String())
		}
	}
}

// The MemoryUsage struct to holds memory usage and memory limit information about a process.
type MemoryUsage struct {
	PID     int
	Cmdline string
	Usage   memory
	Limit   memory
}

var unlimited memory = (memory(0x7FFFFFFFFFFFFFFF) / memory(os.Getpagesize())) * memory(os.Getpagesize())

func (m MemoryUsage) String() string {
	if m.Limit == unlimited {
		return fmt.Sprintf("PID %d, %s: %s", m.PID, m.Usage.String(), m.Cmdline)
	} else {
		return fmt.Sprintf("PID %d, %s, %d%%: %s", m.PID, m.Usage.String(), m.PercentUsed(), m.Cmdline)
	}
}

// The MemoryUsage.PercentUsed method returns memory usage as a percentage of memory limit.
func (m MemoryUsage) PercentUsed() int {
	return int(float64(m.Usage/m.Limit) * 100)
}

type memory uint64

const GB = 1024 * 1024 * 1024

// Pretty print memory in gigabytes.
func (m memory) String() string {
	return fmt.Sprintf("%.2fG", float64(m)/GB)
}

// The GetAllMemoryUsage function returns a slice containing MemoryUsage info for all processes.
func GetAllMemoryUsage() (result []MemoryUsage) {
	files, err := ioutil.ReadDir("/proc")
	if err != nil {
		log.Printf("could not access memory info: %v", err)
		return
	}

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(file.Name())
		if err != nil {
			continue
		}

		bytes, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if err != nil {
			if errors.Is(err, os.ErrPermission) || errors.Is(err, os.ErrNotExist) {
				// Don't complain if we don't have permission or the info doesn't exist.
				continue
			}
			log.Printf("couldn't access cmdline for %d: %v", pid, err)
			continue
		}
		cmdline := strings.TrimSuffix(string(bytes), "\n")

		limit, err := readMemory(fmt.Sprintf("/proc/%d/root/sys/fs/cgroup/memory/memory.limit_in_bytes", pid))
		if err != nil {
			if errors.Is(err, os.ErrPermission) || errors.Is(err, os.ErrNotExist) {
				// Don't complain if we don't have permission or the info doesn't exist.
				continue
			}
			log.Printf("couldn't access limit for %d: %v", pid, err)
			continue
		}
		usage, err := readMemory(fmt.Sprintf("/proc/%d/root/sys/fs/cgroup/memory/memory.usage_in_bytes", pid))
		if err != nil {
			if errors.Is(err, os.ErrPermission) || errors.Is(err, os.ErrNotExist) {
				// Don't complain if we don't have permission or the info doesn't exist.
				continue
			}
			log.Printf("couldn't access usage for %d: %v", pid, err)
			continue
		}

		result = append(result, MemoryUsage{pid, cmdline, usage, limit})
	}

	return
}

// Read a uint64 from a file and convert it to memory.
func readMemory(fpath string) (memory, error) {
	contentAsB, err := ioutil.ReadFile(fpath)
	if err != nil {
		return 0, err
	}
	contentAsStr := strings.TrimSuffix(string(contentAsB), "\n")
	m, err := strconv.ParseUint(contentAsStr, 10, 64)
	return memory(m), err
}

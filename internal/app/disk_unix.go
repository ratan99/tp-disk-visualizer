//go:build unix

package app

import (
	"fmt"
	"os"
	"syscall"
)

func diskUsage(path string) (total, free uint64) {
	// TPDV_DISK="total,free" (bytes) replaces the real numbers, so demo
	// screenshots don't reveal the machine's disk.
	if v := os.Getenv("TPDV_DISK"); v != "" {
		if n, _ := fmt.Sscanf(v, "%d,%d", &total, &free); n == 2 {
			return total, free
		}
	}
	var st syscall.Statfs_t
	if syscall.Statfs(path, &st) != nil {
		return 0, 0
	}
	return st.Blocks * uint64(st.Bsize), st.Bavail * uint64(st.Bsize)
}

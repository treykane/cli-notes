//go:build unix && !darwin

package app

import (
	"os"
	"syscall"
	"time"
)

func fileCreationTime(info os.FileInfo) (time.Time, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return time.Time{}, false
	}
	return time.Unix(int64(stat.Ctim.Sec), int64(stat.Ctim.Nsec)), true
}

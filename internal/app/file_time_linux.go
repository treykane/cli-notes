// file_time_linux.go provides Linux-specific file creation-time retrieval.
//
// Linux kernels/filesystems may expose birth time via statx (STATX_BTIME).
// When unavailable, callers fall back to modification time.

//go:build linux

package app

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// fileCreationTime attempts to read true file birth time using statx.
// Returns (zero,false) when unavailable so callers can fall back to ModTime.
func fileCreationTime(path string, _ os.FileInfo) (time.Time, bool) {
	if path == "" {
		return time.Time{}, false
	}

	var stat unix.Statx_t
	if err := unix.Statx(unix.AT_FDCWD, path, unix.AT_SYMLINK_NOFOLLOW, unix.STATX_BTIME, &stat); err != nil {
		return time.Time{}, false
	}
	return birthTimeFromStatx(stat)
}

func birthTimeFromStatx(stat unix.Statx_t) (time.Time, bool) {
	if stat.Mask&unix.STATX_BTIME == 0 {
		return time.Time{}, false
	}
	return time.Unix(stat.Btime.Sec, int64(stat.Btime.Nsec)), true
}

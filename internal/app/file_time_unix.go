// file_time_unix.go provides the fileCreationTime implementation for non-macOS
// Unix systems (Linux, FreeBSD, etc.).
//
// On these platforms, true file birth time (btime) is not universally available
// through the Go standard library's syscall.Stat_t. Instead, this
// implementation returns Ctim (the metadata-change time, commonly called
// "ctime"), which records the last time the file's inode metadata was modified
// (permissions, ownership, link count, etc.).
//
// Ctim is NOT the same as creation time — renaming a file, changing
// permissions, or creating a hard link will all update Ctim. However, it is
// the closest approximation available without platform-specific extensions
// (like statx on Linux 4.11+), and for typical note files that are created
// once and edited in-place, it often coincides with the true creation time.
//
// See file_time_darwin.go for the macOS implementation (which uses the true
// birth time) and file_time_other.go for the fallback on unsupported platforms.

//go:build unix && !darwin

package app

import (
	"os"
	"syscall"
	"time"
)

// fileCreationTime returns the metadata-change time (Ctim) for the given file
// on non-macOS Unix systems. This serves as a best-effort proxy for file
// creation time; see the file-level comment for caveats.
//
// Returns the timestamp and true on success, or a zero time and false if the
// underlying syscall.Stat_t is not available (e.g. on a non-native filesystem
// implementation).
func fileCreationTime(info os.FileInfo) (time.Time, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return time.Time{}, false
	}
	return time.Unix(int64(stat.Ctim.Sec), int64(stat.Ctim.Nsec)), true
}

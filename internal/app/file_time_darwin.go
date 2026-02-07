// file_time_darwin.go provides macOS-specific file creation time retrieval.
//
// On macOS (Darwin), the true file birth time is available via the
// Birthtimespec field of syscall.Stat_t. This gives an accurate creation
// timestamp that does not change when the file's metadata or content is
// modified, unlike Ctim (metadata-change time) used on other Unix systems.
//
// This implementation is selected at compile time via the "darwin" build tag.
// See file_time_unix.go for the non-Darwin Unix fallback and
// file_time_other.go for platforms that do not expose creation time at all.

//go:build darwin

package app

import (
	"os"
	"syscall"
	"time"
)

// fileCreationTime extracts the true file birth time from the macOS-specific
// Birthtimespec field in the underlying syscall.Stat_t structure.
//
// Returns the creation timestamp and true on success, or a zero time and false
// if the FileInfo does not carry a *syscall.Stat_t (e.g. when the FileInfo
// was synthesized rather than obtained from a real stat call).
//
// This value is used by the "created" sort mode (sortModeCreated) to order
// notes by when they were originally created, and by resolveCreatedAt as the
// preferred source of creation time on macOS.
func fileCreationTime(_ string, info os.FileInfo) (time.Time, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return time.Time{}, false
	}
	return time.Unix(int64(stat.Birthtimespec.Sec), int64(stat.Birthtimespec.Nsec)), true
}

// file_time_other.go provides a no-op fallback for platforms that do not expose
// file creation (birth) time through the standard syscall interface.
//
// This file is compiled on non-Unix, non-Darwin targets (e.g. Windows, Plan 9)
// via the build constraint "!unix && !darwin". On these platforms,
// fileCreationTime always returns (zero, false), causing resolveCreatedAt
// (in util.go) to fall back to the file's modification time instead.
//
// See file_time_darwin.go and file_time_unix.go for platforms that do provide
// creation-time information.

//go:build !unix && !darwin

package app

import (
	"os"
	"time"
)

// fileCreationTime attempts to retrieve the file's creation (birth) time from
// its os.FileInfo. On unsupported platforms this always returns the zero time
// and false, signaling the caller to use the modification time as a fallback.
func fileCreationTime(_ string, _ os.FileInfo) (time.Time, bool) {
	return time.Time{}, false
}

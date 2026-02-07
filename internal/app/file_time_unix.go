// file_time_unix.go provides a fallback fileCreationTime implementation for
// non-macOS, non-Linux Unix systems.
//
// On these platforms the app does not attempt a ctime-based approximation
// because ctime is metadata-change time, not creation time. Returning false
// here ensures callers fall back to ModTime.
//
// See file_time_darwin.go for the macOS implementation (which uses the true
// birth time) and file_time_other.go for the fallback on unsupported platforms.

//go:build unix && !darwin && !linux

package app

import (
	"os"
	"time"
)

// fileCreationTime returns false on non-Linux Unix targets. The caller falls
// back to ModTime for created-sort ordering.
func fileCreationTime(_ string, _ os.FileInfo) (time.Time, bool) {
	return time.Time{}, false
}

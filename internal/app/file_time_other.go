//go:build !unix && !darwin

package app

import (
	"os"
	"time"
)

func fileCreationTime(info os.FileInfo) (time.Time, bool) {
	return time.Time{}, false
}

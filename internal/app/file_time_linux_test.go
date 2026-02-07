//go:build linux

package app

import (
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestBirthTimeFromStatxRequiresBirthMask(t *testing.T) {
	stat := unix.Statx_t{}
	if got, ok := birthTimeFromStatx(stat); ok || !got.IsZero() {
		t.Fatalf("expected no birth time, got %v ok=%v", got, ok)
	}
}

func TestBirthTimeFromStatxReturnsBirthTime(t *testing.T) {
	stat := unix.Statx_t{}
	stat.Mask = unix.STATX_BTIME
	stat.Btime.Sec = 123
	stat.Btime.Nsec = 456

	got, ok := birthTimeFromStatx(stat)
	if !ok {
		t.Fatal("expected birth time")
	}
	want := time.Unix(123, 456)
	if !got.Equal(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

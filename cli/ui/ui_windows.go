//go:build windows

package ui

import (
	"os"
	"syscall"
	"unsafe"
)

// enableANSI turns on colour codes in the Windows console, which has them off by default.
func enableANSI() bool {
	k := syscall.NewLazyDLL("kernel32.dll")
	get, set := k.NewProc("GetConsoleMode"), k.NewProc("SetConsoleMode")
	good := false
	for _, f := range []*os.File{os.Stdout, os.Stderr} {
		var mode uint32
		if r, _, _ := get.Call(f.Fd(), uintptr(unsafe.Pointer(&mode))); r == 0 {
			continue
		}
		if r, _, _ := set.Call(f.Fd(), uintptr(mode|0x0004)); r != 0 {
			good = true
		}
	}
	return good
}

//go:build darwin

package scan

import (
	"runtime"
	"syscall"
	"unsafe"
)

// getattrlistbulk is called straight from libSystem (the same technique
// golang.org/x/sys/unix uses), so the binary needs no cgo toolchain.

//go:linkname syscall_syscall6 syscall.syscall6
func syscall_syscall6(fn, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err syscall.Errno)

var libc_getattrlistbulk_trampoline_addr uintptr

//go:cgo_import_dynamic libc_getattrlistbulk getattrlistbulk "/usr/lib/libSystem.B.dylib"

type attrList struct {
	bitmapCount uint16
	reserved    uint16
	commonAttr  uint32
	volAttr     uint32
	dirAttr     uint32
	fileAttr    uint32
	forkAttr    uint32
}

func getattrlistbulk(fd int, al *attrList, buf []byte) (int, error) {
	r1, _, e := syscall_syscall6(libc_getattrlistbulk_trampoline_addr,
		uintptr(fd), uintptr(unsafe.Pointer(al)), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0, 0)
	runtime.KeepAlive(al)
	runtime.KeepAlive(buf)
	if e != 0 {
		return 0, e
	}
	return int(r1), nil
}

//go:build darwin

package scan

import (
	"encoding/binary"
	"os"
	"sync"
	"syscall"
)

const (
	attrBitMapCount   = 5
	attrCmnReturned   = 0x80000000
	attrCmnName       = 0x00000001
	attrCmnDevID      = 0x00000002
	attrCmnObjType    = 0x00000008
	attrCmnFileID     = 0x02000000
	attrCmnError      = 0x20000000
	attrFileLinkCount = 0x00000001
	attrFileAllocSize = 0x00000004
	vDIR              = 2
	vLNK              = 5
)

var forceStd = os.Getenv("TPDV_STD") == "1"

var bulkAttrs = attrList{
	bitmapCount: attrBitMapCount,
	commonAttr:  attrCmnReturned | attrCmnName | attrCmnError | attrCmnDevID | attrCmnObjType | attrCmnFileID,
	fileAttr:    attrFileLinkCount | attrFileAllocSize,
}

var bulkBufs = sync.Pool{New: func() any { b := make([]byte, 256<<10); return &b }}

// readDir uses getattrlistbulk: one call returns names, types and
// allocated sizes for a whole batch of entries, replacing readdir plus
// an lstat per file.
func readDir(path string, fn func(*entry)) error {
	if forceStd {
		return readDirStd(path, fn)
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer syscall.Close(fd)

	bp := bulkBufs.Get().(*[]byte)
	defer bulkBufs.Put(bp)
	buf := *bp
	le := binary.LittleEndian
	al := bulkAttrs
	first := true
	var e entry
	for {
		n, err := getattrlistbulk(fd, &al, buf)
		if err != nil {
			if first && (err == syscall.ENOTSUP || err == syscall.EINVAL) {
				return readDirStd(path, fn) // filesystem without bulk support
			}
			return err
		}
		if n == 0 {
			return nil
		}
		first = false
		off := 0
		for i := 0; i < n; i++ {
			rec := buf[off : off+int(le.Uint32(buf[off:]))]
			off += len(rec)

			// Layout: length, attribute_set_t returned, [error], then the
			// requested attributes in bit order, each only if returned.
			common := le.Uint32(rec[4:])
			fileA := le.Uint32(rec[16:])
			p := 24
			e = entry{}
			if common&attrCmnError != 0 {
				e.err = le.Uint32(rec[p:]) != 0
				p += 4
			}
			if common&attrCmnName != 0 {
				start := p + int(int32(le.Uint32(rec[p:])))
				end := start + int(le.Uint32(rec[p+4:])) - 1 // drop NUL
				if end > start && end <= len(rec) {
					e.name = rec[start:end]
				}
				p += 8
			}
			if common&attrCmnDevID != 0 {
				e.dev = uint64(le.Uint32(rec[p:]))
				p += 4
			}
			if common&attrCmnObjType != 0 {
				t := le.Uint32(rec[p:])
				e.isDir = t == vDIR
				e.isLink = t == vLNK
				p += 4
			}
			if common&attrCmnFileID != 0 {
				e.ino = le.Uint64(rec[p:])
				p += 8
			}
			if fileA&attrFileLinkCount != 0 {
				e.nlink = le.Uint32(rec[p:])
				p += 4
			}
			if fileA&attrFileAllocSize != 0 {
				e.size = int64(le.Uint64(rec[p:]))
			}
			if len(e.name) == 0 {
				continue
			}
			fn(&e)
		}
	}
}

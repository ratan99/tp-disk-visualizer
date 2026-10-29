//go:build unix

package scan

import (
	"io"
	"os"
	"syscall"
)

// readDirStd is the portable reader: readdir plus one lstat per entry.
func readDirStd(path string, fn func(*entry)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for {
		ds, err := f.ReadDir(1024)
		for _, d := range ds {
			info, ierr := d.Info()
			if ierr != nil {
				continue
			}
			st, ok := info.Sys().(*syscall.Stat_t)
			if !ok {
				continue
			}
			fn(&entry{
				name:   []byte(d.Name()),
				isDir:  d.IsDir(),
				isLink: d.Type()&os.ModeSymlink != 0,
				size:   int64(st.Blocks) * 512,
				dev:    uint64(st.Dev),
				ino:    uint64(st.Ino),
				nlink:  uint32(st.Nlink),
			})
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func devOf(path string) (uint64, error) {
	var st syscall.Stat_t
	if err := syscall.Stat(path, &st); err != nil {
		return 0, err
	}
	return uint64(st.Dev), nil
}

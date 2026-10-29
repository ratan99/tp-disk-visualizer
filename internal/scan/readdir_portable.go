//go:build unix && !darwin

package scan

func readDir(path string, fn func(*entry)) error { return readDirStd(path, fn) }

package scan

// defaultExcludes skips pseudo filesystems, other volumes and the Data
// volume mount (already reachable through firmlinks like /Users), so a
// scan of "/" does not double count.
func defaultExcludes() map[string]bool {
	return map[string]bool{
		"/dev":            true,
		"/Volumes":        true,
		"/System/Volumes": true,
		"/Network":        true,
		"/.vol":           true,
	}
}

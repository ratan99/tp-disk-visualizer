//go:build !darwin

package scan

func defaultExcludes() map[string]bool {
	return map[string]bool{
		"/proc": true,
		"/sys":  true,
		"/dev":  true,
		"/run":  true,
	}
}

// Package web holds the single-page browser UI, shared by the live
// server and the static HTML report.
package web

import _ "embed"

//go:embed index.html
var Index string

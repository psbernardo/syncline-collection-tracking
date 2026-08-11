package webtemplates

import "embed"

//go:embed layout.html dashboard.html partials/*.html
var FS embed.FS

package webstatic

import "embed"

// FS contains the browser assets required by the web application.
//
// Embedding the files keeps the layout available regardless of the server's
// working directory.
//
//go:embed app.css app.js alpine.min.js htmx.min.js
var FS embed.FS

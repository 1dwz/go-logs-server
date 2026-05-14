package web

import "embed"

//go:embed index.html view.html app.css home.js view.js
var Files embed.FS

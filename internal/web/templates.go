package web

import (
	"embed"
)

//go:embed templates/*.html static/*.css
var templatesFS embed.FS


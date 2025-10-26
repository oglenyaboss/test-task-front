package web

import (
	"embed"
	"io/fs"
)

//go:embed swagger/*
var swaggerFS embed.FS

func SwaggerFS() (fs.FS, error) {
	return fs.Sub(swaggerFS, "swagger")
}

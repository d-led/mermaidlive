package mermaidlive

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
)

const uiSrc = "ui-src"
const dist = "dist"

//go:embed dist/*
var embeddedDist embed.FS

func GetFS() http.FileSystem {
	if DoEmbed {
		return getEmbeddedFS()
	}
	return getLocalFS()
}

func getLocalFS() http.FileSystem {
	return http.FS(os.DirFS(dist))
}

func getEmbeddedFS() http.FileSystem {
	sub, err := fs.Sub(embeddedDist, dist)

	if err != nil {
		panic(err)
	}

	return http.FS(sub)
}

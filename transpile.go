package mermaidlive

import (
	"log"
	"os"
	"path/filepath"

	"github.com/evanw/esbuild/pkg/api"
)

func Refresh() {
	log.Printf("transpiling & copying: %v, %v", staticFilesToCopy(), esbuildEntrypoints())
	transpile()
	copyStatic()
}

func copyStatic() {
	for _, f := range staticFilesToCopy() {
		text, err := os.ReadFile(filepath.Join(uiSrc, f))
		crashOnError(err)
		os.WriteFile(filepath.Join(dist, f), text, 0644)
	}
}

func transpile() {
	result := api.Build(api.BuildOptions{
		EntryPoints:      esbuildEntrypoints(),
		Bundle:           true,
		Outdir:           dist,
		MinifySyntax:     false,
		MinifyWhitespace: false,

		MinifyIdentifiers: false,
		Sourcemap:         api.SourceMapInline,
		Engines: []api.Engine{
			{Name: api.EngineChrome, Version: "58"},
			{Name: api.EngineFirefox, Version: "57"},
			{Name: api.EngineSafari, Version: "11"},
			{Name: api.EngineEdge, Version: "16"},
		},
		Write: true,
	})
	handleErrors(result.Errors)
}

func staticFilesToCopy() []string {
	var res = []string{
		"index.html",
		"index.css",
	}
	if ClusterObservabilityEnabled {
		res = append(res, "cluster.html")
	}
	return res
}

func esbuildEntrypoints() []string {
	var res = []string{
		filepath.Join(uiSrc, "index.ts"),
	}
	if ClusterObservabilityEnabled {
		res = append(res, filepath.Join(uiSrc, "cluster.ts"))
	}
	return res
}

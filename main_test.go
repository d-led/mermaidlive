package mermaidlive

import (
	"github.com/cucumber/godog"
	"github.com/spf13/pflag"
)

func init() {
	godog.BindCommandLineFlags("godog.", &opts)
	pflag.Parse()
	opts.Paths = pflag.Args()
}

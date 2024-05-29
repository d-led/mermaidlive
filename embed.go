//go:build embed
// +build embed

package main

import (
	"io"
	"log"
)

func init() {
	log.Println("using embedded resources")
	doEmbed = true
}

type noop struct{}

func (n *noop) Close() error {
	return nil
}

func startWatching() io.Closer {
	return &noop{}
}

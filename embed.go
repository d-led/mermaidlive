//go:build embed
// +build embed

package main

import (
	"io"
	"log"

	"github.com/cskr/pubsub/v2"
)

func init() {
	log.Println("using embedded resources")
	doEmbed = true
}

type noop struct{}

func (n *noop) Close() error {
	return nil
}

func startWatching(_ *pubsub.PubSub[string, Event]) io.Closer {
	return &noop{}
}

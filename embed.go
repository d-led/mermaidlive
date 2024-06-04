//go:build embed
// +build embed

package mermaidlive

import (
	"io"
	"log"

	"github.com/cskr/pubsub/v2"
)

func init() {
	log.Println("using embedded resources")
	DoEmbed = true
}

type noop struct{}

func (n *noop) Close() error {
	return nil
}

func StartWatching(_ *pubsub.PubSub[string, Event]) io.Closer {
	return &noop{}
}

//go:build !embed
// +build !embed

package mermaidlive

import (
	"log"

	"github.com/cskr/pubsub/v2"
	"github.com/fsnotify/fsnotify"
)

func init() {
	log.Println("using filesystem resources")
}

func StartWatching(eventPublisher *pubsub.PubSub[string, Event]) *fsnotify.Watcher {
	watcher, err := fsnotify.NewWatcher()
	crashOnError(err)

	// Start listening for events.
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) {
					log.Println("modified: ", event.Name)
					Refresh()
					eventPublisher.Pub(NewSimpleEvent("ResourcesRefreshed"), Topic)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error: ", err)
			}
		}
	}()

	err = watcher.Add(uiSrc)
	crashOnError(err)
	return watcher
}

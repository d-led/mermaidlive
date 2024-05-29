//go:build !embed
// +build !embed

package main

import (
	"log"

	"github.com/fsnotify/fsnotify"
)

func init() {
	log.Println("using filesystem resources")
}

func startWatching() *fsnotify.Watcher {
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
					refresh()
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

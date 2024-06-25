package mermaidlive

import (
	"log"

	"github.com/Arceliar/phony"
)

type PersistentClusterObserver struct {
	phony.Inbox
	identity string
}

func NewPersistentClusterObserver(identity string) *PersistentClusterObserver {
	return &PersistentClusterObserver{
		identity: identity,
	}
}

func (o *PersistentClusterObserver) AfterMessageSent(peer string, msg interface{}) {
	o.Act(o, func() {
		if message, ok := msg.([]byte); ok {
			log.Printf("Message sent to %s: %s", peer, string(message))
		}
	})
}

func (o *PersistentClusterObserver) AfterMessageReceived(peer string, msg interface{}) {
	o.Act(o, func() {
		if message, ok := msg.([]byte); ok {
			log.Printf("Message received from %s: %s", peer, message)
		}
	})
}

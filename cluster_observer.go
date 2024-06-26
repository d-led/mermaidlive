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

func (o *PersistentClusterObserver) AfterMessageSent(peer string, msg []byte) {
	o.Act(o, func() {
		log.Printf("Message sent to %s: %s", peer, string(msg))
	})
}

func (o *PersistentClusterObserver) AfterMessageReceived(peer string, msg []byte) {
	o.Act(o, func() {
		log.Printf("Message received from %s: %s", peer, msg)
	})
}

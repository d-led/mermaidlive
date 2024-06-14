package mermaidlive

import (
	"log"
	"net"
	"strings"
	"time"

	"github.com/cskr/pubsub/v2"
)

const peerUpdateDelay = 5 * time.Second

type PeerSource struct {
	domainName string
	events     *pubsub.PubSub[string, Event]
}

func NewPeerSource(events *pubsub.PubSub[string, Event], domainName string) *PeerSource {
	return &PeerSource{
		domainName: strings.TrimSpace(domainName),
		events:     events,
	}
}

func (ps *PeerSource) Start() {
	if ps.domainName == "" {
		log.Println("not polling for peers")
		return
	}

	log.Printf("Starting to poll for peers at %s", ps.domainName)
	go ps.pollForever()
}

func (ps *PeerSource) pollForever() {
	for {
		ps.getPeers()
		time.Sleep(peerUpdateDelay)
	}
}

func (ps *PeerSource) getPeers() {
	addrs, err := net.LookupHost(ps.domainName)
	if err != nil {
		log.Printf("DNS resolution error: %v", err)
		return
	}
	ps.events.Pub(NewEventWithParam("ReplicasActive", len(addrs)))
}

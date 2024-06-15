package mermaidlive

import (
	"log"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/cskr/pubsub/v2"
)

const peerUpdateDelay = 5 * time.Second

type PeerSource struct {
	domainName string
	events     *pubsub.PubSub[string, Event]
	peers      []string
}

func NewFlyPeerSource(events *pubsub.PubSub[string, Event]) *PeerSource {
	return &PeerSource{
		domainName: strings.TrimSpace(getFlyPeersDomain()),
		events:     events,
		peers:      []string{},
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
	myIp := getFlyPrivateIP()
	peers := []string{}
	for _, peer := range addrs {
		if peer != myIp {
			peers = append(peers, peer)
		}
	}
	slices.Sort(peers)
	if !slices.Equal(peers, ps.peers) {
		ps.peers = peers
		log.Printf("Peers changed to: %v", peers)
	}
	ps.events.Pub(NewEventWithParam("ReplicasActive", len(addrs)), Topic)
}

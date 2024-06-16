package mermaidlive

import (
	"fmt"
	"log"
	"net"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/cskr/pubsub/v2"
	"github.com/d-led/percounter"
)

const peerUpdateDelay = 5 * time.Second

type PeerSource struct {
	domainName string
	events     *pubsub.PubSub[string, Event]
	peers      []string
	counter    *percounter.ZmqSingleGcounter
}

func NewFlyPeerSource(events *pubsub.PubSub[string, Event]) *PeerSource {
	counterFilename := getCounterFilename()
	log.Println("Visitor counter filename:", counterFilename)
	return &PeerSource{
		domainName: strings.TrimSpace(getFlyPeersDomain()),
		events:     events,
		peers:      []string{},
		counter: percounter.NewObservableZmqSingleGcounter(
			getCounterIdentity(),
			counterFilename,
			getFlyZmqBindAddr(),
			NewCounterListener(events),
		),
	}
}

func getCounterIdentity() string {
	res := getFlyPrivateIP()
	if res != "" {
		return res
	}
	return "localhost"
}

func getFlyZmqBindAddr() string {
	zmqPort := getFlyZmqPort()
	return fmt.Sprintf("tcp://:%s", zmqPort)
}

func getFlyZmqPort() string {
	if port, ok := os.LookupEnv("ZMQ_PORT"); ok {
		return port
	}
	return "5000"
}

func (ps *PeerSource) Start() {
	go ps.listenToInternalEventsForever()

	if ps.domainName == "" {
		log.Println("not polling for peers")
		return
	}

	log.Printf("Starting to poll for peers")
	go ps.pollForever()
}

func (ps *PeerSource) pollForever() {
	for {
		ps.getPeers()
		time.Sleep(peerUpdateDelay)
	}
}

func (ps *PeerSource) listenToInternalEventsForever() {
	subscription := ps.events.Sub(InternalTopic)
	defer ps.events.Unsub(subscription, InternalTopic)
	for event := range subscription {
		if event.Name == "VisitorJoined" {
			ps.counter.Increment()
			ps.events.Pub(NewEventWithParam("TotalVisitors", ps.counter.Value()), Topic)
		}
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
	if !slices.Equal(peers, ps.peers) || (len(ps.peers) == 0 && len(peers) != 0) {
		log.Printf("Peers changed %v -> %v", ps.peers, peers)
		ps.peers = peers
		ps.counter.UpdatePeers(zmqPeers(peers))
	}
	ps.events.Pub(NewEventWithParam("ReplicasActive", len(addrs)), Topic)
}

func zmqAddressOf(peer string) string {
	return fmt.Sprintf("tcp://[%s]:%s", peer, getFlyZmqPort())
}

func zmqPeers(peers []string) []string {
	res := []string{}
	for _, peer := range peers {
		res = append(res, zmqAddressOf(peer))
	}
	return res
}

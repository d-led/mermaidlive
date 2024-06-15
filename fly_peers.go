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
)

const peerUpdateDelay = 5 * time.Second

type PeerSource struct {
	domainName string
	events     *pubsub.PubSub[string, Event]
	peers      []string
	udpServer  *UDPServer
	udpClient  *UDPClient
}

func NewFlyPeerSource(events *pubsub.PubSub[string, Event]) *PeerSource {
	return &PeerSource{
		domainName: strings.TrimSpace(getFlyPeersDomain()),
		events:     events,
		peers:      []string{},
		udpServer:  NewUDPServer(getFlyUDPBindAddr()),
		udpClient:  NewUDPClient(),
	}
}

func getFlyUDPBindAddr() string {
	udpPort := getFlyUDPPort()
	return fmt.Sprintf("fly-global-services:%s", udpPort)
}

func getFlyUDPPort() string {
	if udpPort, ok := os.LookupEnv("UDP_PORT"); ok {
		return udpPort
	}
	return "5000"
}

func (ps *PeerSource) Start() {
	if ps.domainName == "" {
		log.Println("not polling for peers")
		return
	}

	log.Printf("Starting to poll for peers at %s", ps.domainName)
	go ps.pollForever()
	go ps.udpServer.Start()
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
	if !slices.Equal(peers, ps.peers) || len(ps.peers) == 0 {
		ps.peers = peers
		log.Printf("Peers changed to: %v", peers)
		for _, peer := range peers {
			err := ps.udpClient.Send(
				fmt.Sprintf("[%s]:%s", peer, getFlyUDPPort()),
				[]byte(fmt.Sprintf("Hello from %s", getFlyPrivateIP())),
			)
			if err != nil {
				log.Printf("Failed sending a UDP hello to %s: %e", peer, err)
			}
		}
	}
	ps.events.Pub(NewEventWithParam("ReplicasActive", len(addrs)), Topic)
}

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
	zmqServer  *ZmqServer
	zmqClients map[string]*ZmqClient
}

func NewFlyPeerSource(events *pubsub.PubSub[string, Event]) *PeerSource {
	return &PeerSource{
		domainName: strings.TrimSpace(getFlyPeersDomain()),
		events:     events,
		peers:      []string{},
		zmqServer:  NewZmqServer(getFlyZmqBindAddr()),
		zmqClients: make(map[string]*ZmqClient),
	}
}

func getFlyZmqBindAddr() string {
	zmqPort := getFlyZmqPort()
	return fmt.Sprintf("tcp://0.0.0.0:%s", zmqPort)
}

func getFlyZmqPort() string {
	if port, ok := os.LookupEnv("ZMQ_PORT"); ok {
		return port
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
	go ps.zmqServer.Start()
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
		ps.updateZmqConnections()
		log.Printf("Peers changed to: %v", peers)
		for _, peer := range peers {
			err := ps.sendZmqMessage(peer, []byte(fmt.Sprintf("Hello from %s", getFlyPrivateIP())))
			if err != nil {
				log.Printf("Failed sending a ZMQ hello to %s: %e", peer, err)
			}
		}
	}
	ps.events.Pub(NewEventWithParam("ReplicasActive", len(addrs)), Topic)
}

func (ps *PeerSource) updateZmqConnections() {
	zmqPort := getFlyZmqPort()
	peers := setOf(ps.peers)
	// if not in new peers, close & remove the connection
	for clientPeer, conn := range ps.zmqClients {
		if _, ok := peers[clientPeer]; !ok {
			log.Println("Removing connection to", clientPeer)
			conn.Close()
			delete(peers, clientPeer)
		}
	}

	// if not connected, connect
	for _, peer := range ps.peers {
		if _, ok := ps.zmqClients[peer]; !ok {
			peerAddr := fmt.Sprintf("tcp://[%s]:%s", peer, zmqPort)
			client := NewZmqClient(peerAddr)
			log.Println("Connecting to", peerAddr)
			err := client.Connect()
			if err != nil {
				log.Printf("Could not connect to peer: %s: %v", peerAddr, err)
				continue
			}
			ps.zmqClients[peer] = client
		}
	}
}

func (ps *PeerSource) sendZmqMessage(peer string, msg []byte) error {
	client, ok := ps.zmqClients[peer]
	if !ok {
		return fmt.Errorf("could not find a client for %s", peer)
	}
	return client.Send(msg)
}

func setOf(s []string) map[string]bool {
	var res = make(map[string]bool)
	for _, e := range s {
		res[e] = true
	}
	return res
}

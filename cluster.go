package mermaidlive

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/cskr/pubsub/v2"
	"github.com/d-led/percounter"
	"github.com/d-led/zmqcluster"
)

const peerUpdateDelay = 5 * time.Second

type Cluster struct {
	events      *pubsub.PubSub[string, Event]
	peers       []string
	cluster     zmqcluster.Cluster
	counter     *percounter.ZmqMultiGcounter
	peerLocator PeerLocator
}

func NewCluster(events *pubsub.PubSub[string, Event], clusterEventObserver *PersistentClusterObserver, cluster zmqcluster.Cluster) *Cluster {
	counterDirectory := GetCounterDirectory()
	log.Println("Counter directory:", counterDirectory)
	identity := GetCounterIdentity()
	counterListener := NewCounterListener(events)
	counter := percounter.NewObservableZmqMultiGcounterInCluster(
		identity,
		counterDirectory,
		cluster,
		counterListener,
	)
	counter.ShouldPersistOnSignal()
	counter.SetClusterObserver(clusterEventObserver)
	if err := counter.LoadAllSync(); err != nil {
		log.Printf("failed to load all counters, continuing nonetheless: %v", err)
	}

	return &Cluster{
		peerLocator: ChoosePeerLocator(),
		events:      events,
		peers:       []string{},
		cluster:     cluster,
		counter:     counter,
	}
}

func GetCounterIdentity() string {
	return getPrivateReplicaId()
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

func (ps *Cluster) Start() {
	go ps.listenToInternalEventsForever()

	if ps.peerLocator == nil {
		log.Println("not polling for peers")
		return
	}

	log.Printf("Starting to poll for peers")

	go ps.pollForever()
}

func (ps *Cluster) pollForever() {
	ps.cluster.Start()
	defer ps.counter.Stop()
	for {
		ps.getPeers()
		time.Sleep(peerUpdateDelay)
	}
}

func (ps *Cluster) listenToInternalEventsForever() {
	subscription := ps.events.Sub(InternalTopic)
	defer ps.events.Unsub(subscription, InternalTopic)
	for event := range subscription {
		switch event.Name {
		case VisitorJoinedEvent:
			ps.counter.Increment(NewConnectionsCounter)
			ps.counter.Increment(StartedConnectionsCounter)
			ps.events.Pub(NewEventWithParam(TotalVisitorsEvent, ps.counter.Value(NewConnectionsCounter)), Topic)
			sendInitialClusterConnectionCount(ps.events, ps.counter)
		case VisitorLeftEvent:
			ps.counter.Increment(ClosedConnectionsCounter)
		}
	}
}

func (ps *Cluster) getPeers() {
	if ps.peerLocator == nil {
		return
	}

	peers, replicaCount, err := ps.peerLocator.GetPeers()
	if err != nil {
		log.Printf("Error getting peers: %v", err)
		// something might be off, better disconnect from everyone at this point
		peers = []string{}
	}

	slices.Sort(peers)

	if !slices.Equal(peers, ps.peers) || (len(ps.peers) == 0 && len(peers) != 0) {
		log.Printf("Peers changed %v -> %v", ps.peers, peers)
		ps.peers = peers
		ps.counter.UpdatePeers(zmqPeers(peers))
		ps.events.Pub(GetReplicasEvent(replicaCount), Topic)
	}
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

func ChoosePeerLocator() PeerLocator {
	flyDiscoveryDomainName := strings.TrimSpace(getFlyPeersDomain())
	if flyDiscoveryDomainName != "" {
		return NewFlyPeerLocator(flyDiscoveryDomainName)
	}
	traefikServicesUrl := getTraefikServicesUrl()
	if traefikServicesUrl != "" {
		return NewTraefikPeerLocator(traefikServicesUrl)
	}
	return nil
}

func getTraefikServicesUrl() string {
	return os.Getenv("TRAEFIK_SERVICES_URL")
}

func sendInitialClusterConnectionCount(
	events *pubsub.PubSub[string, Event],
	counter *percounter.ZmqMultiGcounter,
) {
	started := counter.Value(StartedConnectionsCounter)
	closed := counter.Value(ClosedConnectionsCounter)

	events.Pub(NewEventWithParam(TotalClusterVisitorsActiveEvent, started-closed))
}

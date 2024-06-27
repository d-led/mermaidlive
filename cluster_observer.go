package mermaidlive

import (
	"encoding/json"
	"log"

	"github.com/Arceliar/phony"
	"github.com/cskr/pubsub/v2"
	"github.com/d-led/percounter"
)

const maxEventsWithUnknownPeersBeforePublishingAllEvents = 8

type messageEvent struct {
	SeenAt string
	Src    string
	Dst    string
	Msg    string
}

type PersistentClusterObserver struct {
	phony.Inbox
	identity         string
	peerIpToIdentity map[string]string
	peerIdentities   map[string]bool
	messagesUpToNow  []*messageEvent
	events           *pubsub.PubSub[string, Event]
}

func NewPersistentClusterObserver(identity string, myIP string, events *pubsub.PubSub[string, Event]) *PersistentClusterObserver {
	return &PersistentClusterObserver{
		identity:         identity,
		peerIpToIdentity: map[string]string{myIP: identity},
		peerIdentities:   map[string]bool{identity: true},
		messagesUpToNow:  []*messageEvent{},
		events:           events,
	}
}

func (o *PersistentClusterObserver) AfterMessageSent(peer string, msg []byte) {
	o.Act(o, func() {
		msgString := string(msg)
		o.messagesUpToNow = append(o.messagesUpToNow,
			&messageEvent{SeenAt: o.identity, Src: o.identity, Dst: peer, Msg: msgString},
		)
		if peerIP, err := getIPOf(peer); err == nil {
			peerIdentity, ok := o.peerIpToIdentity[peerIP]
			if ok {
				peer = peerIdentity
			}
		}
		log.Printf("Message sent to %s: %s", peer, msgString)
	})
}

func (o *PersistentClusterObserver) AfterMessageReceived(peer string, msg []byte) {
	o.Act(o, func() {
		var counterMessage percounter.NetworkedGCounterState
		err := json.Unmarshal(msg, &counterMessage)
		if err == nil {
			peer = counterMessage.SourcePeer
			o.trackCounterIdentitySync(&counterMessage)
		}
		msgString := string(msg)
		o.messagesUpToNow = append(o.messagesUpToNow,
			&messageEvent{SeenAt: o.identity, Src: peer, Dst: o.identity, Msg: msgString},
		)
		log.Printf("Message received from %s: %s", peer, msgString)
	})
}

func (o *PersistentClusterObserver) trackCounterIdentitySync(msg *percounter.NetworkedGCounterState) {
	peerIpI, ok := msg.Metadata["my_ip"]
	if !ok {
		return
	}
	peerIP, ok := peerIpI.(string)
	if !ok {
		return
	}
	o.peerIpToIdentity[peerIP] = msg.SourcePeer
	o.peerIdentities[msg.SourcePeer] = true
	o.processEventsSync()
}

func (o *PersistentClusterObserver) processEventsSync() {
	if len(o.messagesUpToNow) > 0 &&
		(!o.anyUnkownPeersSync() ||
			o.countUnkownPeersSync() >= maxEventsWithUnknownPeersBeforePublishingAllEvents) {
		log.Println("publishing cluster messages")
		o.publishPendingEventsSync()
	}
}

func (o *PersistentClusterObserver) anyUnkownPeersSync() bool {
	if len(o.messagesUpToNow) == 0 {
		return false
	}
	for _, msg := range o.messagesUpToNow {
		if o.unknownPeerSync(msg.Dst) || o.unknownPeerSync(msg.Src) {
			return true
		}
	}
	return false
}

func (o *PersistentClusterObserver) countUnkownPeersSync() int {
	var count = 0
	if len(o.messagesUpToNow) == 0 {
		return 0
	}
	for _, msg := range o.messagesUpToNow {
		if o.unknownPeerSync(msg.Dst) || o.unknownPeerSync(msg.Src) {
			count++
		}
	}
	return count
}

func (o *PersistentClusterObserver) unknownPeerSync(peer string) bool {
	_, ipKnown := o.peerIpToIdentity[peer]
	return !ipKnown && !o.peerIdentities[peer]
}

func (o *PersistentClusterObserver) idOfSync(peer string) string {
	if id, ok := o.peerIpToIdentity[peer]; ok {
		return id
	}
	return peer
}

func (o *PersistentClusterObserver) publishPendingEventsSync() {
	for _, msg := range o.messagesUpToNow {
		e := NewSimpleEvent(ClusterMessageEvent)
		e.Properties = map[string]interface{}{
			"seen_at": msg.SeenAt,
			"src":     o.idOfSync(msg.Src),
			"dst":     o.idOfSync(msg.Dst),
			"msg":     msg.Msg,
		}
		o.events.Pub(e, ClusterMessageTopic)
	}
	o.messagesUpToNow = []*messageEvent{}
}

package mermaidlive

import (
	"encoding/json"
	"log"

	"github.com/Arceliar/phony"
	"github.com/d-led/percounter"
)

const maxEventsWithUnknownPeersBeforePublishingAllEvents = 8

type messageEvent struct {
	SeenAt string `json:"seen_at"`
	Src    string `json:"src"`
	Dst    string `json:"dst"`
	Msg    string `json:"msg"`
}

type PersistentClusterObserver struct {
	phony.Inbox
	identity         string
	peerIpToIdentity map[string]string
	messagesUpToNow  []*messageEvent
}

func NewPersistentClusterObserver(identity string, myIP string) *PersistentClusterObserver {
	return &PersistentClusterObserver{
		identity:         identity,
		peerIpToIdentity: map[string]string{myIP: identity},
		messagesUpToNow:  []*messageEvent{},
	}
}

func (o *PersistentClusterObserver) AfterMessageSent(peer string, msg []byte) {
	o.Act(o, func() {
		if peerIP, err := getIPOf(peer); err == nil {
			peerIdentity, ok := o.peerIpToIdentity[peerIP]
			if ok {
				peer = peerIdentity
			}
		}
		msgString := string(msg)
		o.messagesUpToNow = append(o.messagesUpToNow,
			&messageEvent{SeenAt: o.identity, Src: o.identity, Dst: peer, Msg: msgString},
		)
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
	o.processEventsSync()
}

func (o *PersistentClusterObserver) processEventsSync() {
	if len(o.messagesUpToNow) == 0 ||
		!o.anyUnkownPeersSync() ||
		o.countUnkownPeersSync() < maxEventsWithUnknownPeersBeforePublishingAllEvents {
		return
	}
	log.Println("ready to publish message events")
	o.publishPendingEventsSync()
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
	_, ok := o.peerIpToIdentity[peer]
	return !ok
}

func (o *PersistentClusterObserver) publishPendingEventsSync() {
	// clear the pending events for now
	o.messagesUpToNow = []*messageEvent{}
}

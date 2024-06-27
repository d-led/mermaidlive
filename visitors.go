package mermaidlive

import (
	"github.com/Arceliar/phony"
	"github.com/cskr/pubsub/v2"
)

type VisitorTracker struct {
	phony.Inbox
	events         *pubsub.PubSub[string, Event]
	visitorsActive int
}

func NewVisitorTracker(events *pubsub.PubSub[string, Event]) *VisitorTracker {
	return &VisitorTracker{
		events: events,
	}
}

func (v *VisitorTracker) Joined() {
	v.Act(v, func() {
		v.visitorsActive++
		v.events.Pub(NewEventWithParam(VisitorsActiveEvent, v.visitorsActive), Topic, ClusterMessageTopic)
		v.events.Pub(NewSimpleEvent(VisitorJoinedEvent), InternalTopic)
	})
}

func (v *VisitorTracker) Left() {
	v.Act(v, func() {
		v.visitorsActive--
		v.events.Pub(NewEventWithParam(VisitorsActiveEvent, v.visitorsActive), Topic, ClusterMessageTopic)
		v.events.Pub(NewSimpleEvent(VisitorLeftEvent), InternalTopic)
	})
}

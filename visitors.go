package main

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
		v.events.Pub(NewEventWithParam("VisitorsActive", v.visitorsActive), topic)
	})
}

func (v *VisitorTracker) Left() {
	v.Act(v, func() {
		v.visitorsActive--
		v.events.Pub(NewEventWithParam("VisitorsActive", v.visitorsActive), topic)
	})
}

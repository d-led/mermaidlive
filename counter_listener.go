package mermaidlive

import (
	"log"

	"github.com/cskr/pubsub/v2"
	"github.com/d-led/percounter"
)

type CounterListener struct {
	events *pubsub.PubSub[string, Event]
}

func NewCounterListener(events *pubsub.PubSub[string, Event]) *CounterListener {
	return &CounterListener{
		events: events,
	}
}

func (n *CounterListener) OnNewCount(ev percounter.CountEvent) {
	log.Println("New visitor count:", ev.Count)
	n.events.Pub(NewEventWithParam("TotalVisitors", ev.Count), Topic)
}

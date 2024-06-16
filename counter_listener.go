package mermaidlive

import (
	"log"

	"github.com/cskr/pubsub/v2"
)

type CounterListener struct {
	events *pubsub.PubSub[string, Event]
}

func NewCounterListener(events *pubsub.PubSub[string, Event]) *CounterListener {
	return &CounterListener{
		events: events,
	}
}

func (n *CounterListener) OnNewCount(count int64) {
	log.Println("New visitor count:", count)
	n.events.Pub(NewEventWithParam("TotalVisitors", count), Topic)
}

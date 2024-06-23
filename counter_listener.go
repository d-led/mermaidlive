package mermaidlive

import (
	"log"

	"github.com/Arceliar/phony"
	"github.com/cskr/pubsub/v2"
	"github.com/d-led/percounter"
)

type CounterListener struct {
	phony.Inbox
	events             *pubsub.PubSub[string, Event]
	startedConnections int64
	closedConnections  int64
}

func NewCounterListener(events *pubsub.PubSub[string, Event]) *CounterListener {
	return &CounterListener{
		events: events,
	}
}

func (n *CounterListener) OnNewCount(ev percounter.CountEvent) {
	n.Act(n, func() {
		switch ev.Name {
		case NewConnectionsCounter:
			log.Println("New visitor count:", ev.Count)
			n.events.Pub(NewEventWithParam(TotalVisitorsEvent, ev.Count), Topic)

		case StartedConnectionsCounter:
			log.Printf("started event: %v", ev)
			n.startedConnections = ev.Count
			n.events.Pub(NewEventWithParam(
				TotalClusterVisitorsActiveEvent,
				n.startedConnections-
					n.closedConnections,
			), Topic)

		case ClosedConnectionsCounter:
			log.Printf("closed event: %v", ev)
			n.closedConnections = ev.Count
			n.events.Pub(NewEventWithParam(
				TotalClusterVisitorsActiveEvent,
				n.startedConnections-
					n.closedConnections,
			), Topic)

		default:
			// ignore the event
			// log.Printf("New counter event: %v", ev)
		}
	})
}

package main

import "github.com/cskr/pubsub/v2"

type PubSubPublisher struct {
	events *pubsub.PubSub[string, Event]
}

func NewPubSubPublisher(events *pubsub.PubSub[string, Event]) *PubSubPublisher {
	return &PubSubPublisher{
		events: events,
	}
}

func (p *PubSubPublisher) Publish(e Event) {
	p.events.Pub(e, topic)
}

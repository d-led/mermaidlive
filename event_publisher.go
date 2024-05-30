package main

type EventPublisher interface {
	Publish(e Event)
}

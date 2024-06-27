package mermaidlive

import "time"

type Event struct {
	Timestamp  string                 `json:"timestamp"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
}

func NewSimpleEvent(name string) Event {
	return Event{now(), name, map[string]any{}}
}

func NewEventWithReason(name, reason string) Event {
	return Event{now(), name, map[string]any{"reason": reason}}
}

func NewEventWithParam(name string, p any) Event {
	return Event{now(), name, map[string]any{"param": p}}
}

func now() string {
	return time.Now().Format(time.RFC3339Nano)
}

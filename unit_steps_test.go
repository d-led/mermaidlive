package main

import (
	"context"
	"errors"
	"log"
	"testing"
	"time"

	"github.com/cskr/pubsub/v2"
	"github.com/cucumber/godog"
)

type sutKey struct{}
type observerKey struct{}
type listenerKey struct{}

var errSutNotFound = errors.New("SUT not found, check step definitions")
var errListenerNotFound = errors.New("listener not found, check step definitions")

func aMachineInState(ctx context.Context, state string) (context.Context, error) {
	observer := pubsub.New[string, Event](10 /*enough to buffer between steps*/)
	ctx = context.WithValue(ctx, observerKey{}, observer)
	sut := NewCustomAsyncFSM(observer, 100*time.Millisecond)
	ctx = context.WithValue(ctx, sutKey{}, sut)
	listener := observer.Sub(topic)
	ctx = context.WithValue(ctx, listenerKey{}, listener)

	var err error

	switch state {
	case "waiting":
		if !sut.IsWaiting() {
			err = errors.New("expected the machine to be in waiting state")
		}
	default:
		err = errors.New("unknown state: " + state)
	}

	return ctx, err
}

func theCommandIsCast(ctx context.Context, command string) (context.Context, error) {
	sut, ok := ctx.Value(sutKey{}).(*AsyncFSM)
	if !ok {
		return ctx, errSutNotFound
	}

	var err error

	switch command {
	case "abort":
		sut.AbortWork()
	case "start":
		sut.StartWork()
	default:
		err = errors.New("unknown command: " + command)
	}

	return ctx, err
}

func someWorkHasProgressed() error {
	return godog.ErrPending
}

func workIsCanceled() error {
	return godog.ErrPending
}

func theRequestIsIgnored(ctx context.Context) error {
	events, err := receiveEventsTill(ctx, "RequestIgnored", 1*time.Second)
	if err != nil {
		return err
	}
	return expectToFindEvent(events, "RequestIgnored")
}

func expectToFindEvent(events []Event, event string) error {
	for _, receivedEvent := range events {
		if receivedEvent.Name == event {
			return nil
		}
	}
	return errors.New("Expected to see a 'RequestIgnored' but it hasn't been published")
}

func receiveEventsTill(ctx context.Context, event string, timeout time.Duration) ([]Event, error) {
	res := []Event{}
	var err error
	done := false

	listener, ok := ctx.Value(listenerKey{}).(chan Event)
	if !ok || listener == nil {
		return res, errListenerNotFound
	}

	for {
		select {
		case receivedEvent := <-listener:
			res = append(res, receivedEvent)
			if receivedEvent.Name == event {
				done = true
			}
		case <-time.After(timeout):
			err = errors.New("timed out waiting for event: " + event)
		}
		if done {
			break
		}
	}
	return res, err
}

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t, // Testing instance that will run subtests.
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		listener, ok1 := ctx.Value(listenerKey{}).(chan Event)
		observer, ok2 := ctx.Value(observerKey{}).(*pubsub.PubSub[string, Event])
		if ok1 && listener != nil && ok2 && observer != nil {
			log.Println("unsubscribing listener")
			observer.Unsub(listener, topic)
		}
		ctx = context.WithValue(ctx, listenerKey{}, nil)
		return ctx, nil
	})
	ctx.Step(`^the system is in state "(\S+)"$`, aMachineInState)
	ctx.Step(`^the system "([^"]*)" is requested$`, theCommandIsCast)
	ctx.Step(`^the request is ignored$`, theRequestIsIgnored)
	ctx.Step(`^some work has progressed$`, someWorkHasProgressed)
	ctx.Step(`^work is canceled$`, workIsCanceled)
}

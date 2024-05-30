package main

import (
	"context"
	"errors"
	"log"
	"testing"

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
	sut := NewCustomAsyncFSM(observer, 0)
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
	default:
		err = errors.New("unknown command: " + command)
	}

	return ctx, err
}

func theRequestIsIgnored(ctx context.Context) error {
	events, err := receiveAllFiredEventsUpToNow(ctx, 1)
	if err != nil {
		return err
	}
	for _, receivedEvent := range events {
		if receivedEvent.Name == "RequestIgnored" {
			return nil
		}
	}
	return errors.New("Expected to see a 'RequestIgnored' it hasn't been published")
}

func receiveAllFiredEventsUpToNow(ctx context.Context, expectedCount int) ([]Event, error) {
	res := []Event{}
	done := false

	listener, ok := ctx.Value(listenerKey{}).(chan Event)
	if !ok || listener == nil {
		return res, errListenerNotFound
	}

	for {
		select {
		case receivedEvent := <-listener:
			res = append(res, receivedEvent)
		default:
			if len(res) >= expectedCount {
				done = true
			}
		}
		if done {
			break
		}
	}
	return res, nil
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
}

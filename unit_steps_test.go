//go:build !api_test
// +build !api_test

package mermaidlive

import (
	"context"
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/cskr/pubsub/v2"
	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
)

var opts = godog.Options{
	Output: colors.Colored(os.Stdout),
	Format: "pretty",
	Tags:   "~@api",
}

type sutKey struct{}
type observerKey struct{}
type listenerKey struct{}

var errSutNotFound = errors.New("SUT not found, check step definitions")
var errListenerNotFound = errors.New("listener not found, check step definitions")

func startFromMachineInState(ctx context.Context, state string) (context.Context, error) {
	const delay = 10 * time.Millisecond
	// enough to buffer between steps
	const pubSubChannelCapacity = 10
	ctx, _ = configureSUT(ctx, delay, pubSubChannelCapacity)
	return ctx, theSystemIsFoundInState(ctx, state)
}

func theSystemIsFoundInState(ctx context.Context, state string) error {
	sut, ok := ctx.Value(sutKey{}).(*AsyncFSM)
	if !ok {
		return errSutNotFound
	}

	var err error

	switch state {
	case "waiting":
		if !sut.IsWaiting() {
			err = errors.New("expected the machine to be in waiting state")
		}
	default:
		err = errors.New("unknown state: " + state)
	}

	return err
}

func configureSUT(ctx context.Context,
	delay time.Duration,
	pubSubChannelCapacity int) (context.Context, *AsyncFSM) {
	observer := pubsub.New[string, Event](pubSubChannelCapacity)
	ctx = context.WithValue(ctx, observerKey{}, observer)
	sut := NewCustomAsyncFSM(observer, delay)
	ctx = context.WithValue(ctx, sutKey{}, sut)
	listener := observer.Sub(Topic)
	ctx = context.WithValue(ctx, listenerKey{}, listener)
	return ctx, sut
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

func someWorkHasProgressed(ctx context.Context) error {
	_, err := receiveEventsTill(ctx, "Tick", 1*time.Second)
	return err
}

func workIsCanceled(ctx context.Context) error {
	_, err := receiveEventsTill(ctx, "WorkAborted", 1*time.Second)
	return err
}

func theRequestIsIgnored(ctx context.Context) error {
	_, err := receiveEventsTill(ctx, "RequestIgnored", 1*time.Second)
	return err
}

func workIsCompleted(ctx context.Context) error {
	_, err := receiveEventsTill(ctx, "WorkDone", 1*time.Second)
	return err
}

func receiveEventsTill(ctx context.Context, event string, timeout time.Duration) ([]Event, error) {
	res := []Event{}
	var err error
	done := false

	listener, ok := ctx.Value(listenerKey{}).(chan Event)
	if !ok || listener == nil {
		return res, errListenerNotFound
	}

	timedOut := time.After(timeout)

	for {
		select {
		case receivedEvent := <-listener:
			res = append(res, receivedEvent)
			if receivedEvent.Name == event {
				done = true
			}
		case <-timedOut:
			err = errors.New("timed out waiting for event: " + event)
			done = true
		}
		if done {
			break
		}
	}
	return res, err
}

func TestUnit(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options:             &opts,
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
			observer.Unsub(listener, Topic)
		}
		ctx = context.WithValue(ctx, listenerKey{}, nil)
		return ctx, nil
	})
	ctx.Step(`^a system in state "(\S+)"$`, startFromMachineInState)
	ctx.Step(`^the system is found in state "([^"]*)"$`, theSystemIsFoundInState)
	ctx.Step(`^the system "([^"]*)" is requested$`, theCommandIsCast)
	ctx.Step(`^the request is ignored$`, theRequestIsIgnored)
	ctx.Step(`^some work has progressed$`, someWorkHasProgressed)
	ctx.Step(`^work is completed$`, workIsCompleted)
	ctx.Step(`^work is canceled$`, workIsCanceled)
}

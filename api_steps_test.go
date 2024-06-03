//go:build api_test
// +build api_test

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
	Tags:   "~@ui",
}

const testPort = "8081"

var server *Server

// context keys
type clientKey struct{}
type secondClientKey struct{}
type serverKey struct{}

func aSystemInState(ctx context.Context, state string) (context.Context, error) {
	// client
	client := NewApiClient("http://localhost:" + testPort)
	ctx = context.WithValue(ctx, clientKey{}, client)

	if err := client.WaitForState(state); err != nil {
		return ctx, err
	}

	var err error
	return ctx, err
}

func someWorkHasProgressed(ctx context.Context) error {
	client, err := getClient(ctx)
	if err != nil {
		return err
	}
	return client.WaitForEventSeen("Tick")
}

func theRequestIsIgnored(ctx context.Context) error {
	client, err := getClient(ctx)
	if err != nil {
		return err
	}
	return client.WaitForEventSeen("RequestIgnored")
}

func theSystemIsFoundInState(ctx context.Context, state string) error {
	client, err := getClient(ctx)
	if err != nil {
		return err
	}

	if err := client.WaitForState(state); err != nil {
		return err
	}

	return nil
}

func theSystemIsRequested(ctx context.Context, command string) error {
	client, err := getClient(ctx)
	if err != nil {
		return err
	}
	return client.PostCommand(command)
}

func twoClientsHaveObserved(ctx context.Context, eventName string) error {
	client1, err := getClient(ctx)
	if err != nil {
		return err
	}
	client2, err := getSecondClient(ctx)
	if err != nil {
		return err
	}

	err = client1.WaitForEventSeen(eventName)
	if err != nil {
		return err
	}

	err = client2.WaitForEventSeen(eventName)
	if err != nil {
		return err
	}

	return nil
}

func twoConnectedClients(ctx context.Context) (context.Context, error) {
	client := NewApiClient("http://localhost:" + testPort)
	ctx = context.WithValue(ctx, secondClientKey{}, client)
	return ctx, nil
}

func workIsCanceled(ctx context.Context) error {
	client, err := getClient(ctx)
	if err != nil {
		return err
	}
	return client.WaitForEventSeen("WorkAborted")
}

func workIsCompleted(ctx context.Context) error {
	client, err := getClient(ctx)
	if err != nil {
		return err
	}
	return client.WaitForEventSeen("WorkDone")
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		// server
		ctx = context.WithValue(ctx, serverKey{}, server)
		return ctx, nil
	})
	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		if client1, err := getClient(ctx); err == nil {
			client1.Close()
		}

		if client2, err := getSecondClient(ctx); err == nil {
			client2.Close()
		}

		return ctx, nil
	})
	ctx.Step(`^a system in state "([^"]*)"$`, aSystemInState)
	ctx.Step(`^some work has progressed$`, someWorkHasProgressed)
	ctx.Step(`^the request is ignored$`, theRequestIsIgnored)
	ctx.Step(`^the system is found in state "([^"]*)"$`, theSystemIsFoundInState)
	ctx.Step(`^the system "([^"]*)" is requested$`, theSystemIsRequested)
	ctx.Step(`^two clients have observed "([^"]*)"$`, twoClientsHaveObserved)
	ctx.Step(`^two connected clients$`, twoConnectedClients)
	ctx.Step(`^work is canceled$`, workIsCanceled)
	ctx.Step(`^work is completed$`, workIsCompleted)
}

func TestApi(t *testing.T) {
	// start the server before the suite
	server = startServer(testPort)

	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options:             &opts,
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func getClient(ctx context.Context) (*ApiClient, error) {
	return getClientByKey(ctx, clientKey{})
}

func getSecondClient(ctx context.Context) (*ApiClient, error) {
	return getClientByKey(ctx, secondClientKey{})
}

func getClientByKey(ctx context.Context, key interface{}) (*ApiClient, error) {
	client, ok := ctx.Value(key).(*ApiClient)
	if !ok {
		return nil, errors.New("client not found in test context. Please check test definitions")
	}
	return client, nil
}

func startServer(testPort string) *Server {
	log.Println("Starting a new server")
	eventPublisher := pubsub.New[string, Event](1)
	server = NewServerWithOptions(testPort, eventPublisher, GetFS(true), 10*time.Millisecond)
	go server.Run(testPort)
	return server
}

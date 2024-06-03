// go:build api_test
//go:build api_test
// +build api_test

package mermaidlive

import (
	"context"
	"errors"
	"log"
	"os"
	"testing"

	"github.com/cskr/pubsub/v2"
	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
)

var opts = godog.Options{
	Output: colors.Colored(os.Stdout),
	Format: "pretty",
	Tags:   "~@ui && @only",
}

const testPort = "8081"

var server *Server

// context keys
type clientKey struct{}
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

func twoClientsHaveObservedTheSameEvents() error {
	return godog.ErrPending
}

func twoConnectedClients() error {
	return godog.ErrPending
}

func workIsCanceled() error {
	return godog.ErrPending
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
	ctx.Step(`^a system in state "([^"]*)"$`, aSystemInState)
	ctx.Step(`^some work has progressed$`, someWorkHasProgressed)
	ctx.Step(`^the request is ignored$`, theRequestIsIgnored)
	ctx.Step(`^the system is found in state "([^"]*)"$`, theSystemIsFoundInState)
	ctx.Step(`^the system "([^"]*)" is requested$`, theSystemIsRequested)
	ctx.Step(`^two clients have observed the same events$`, twoClientsHaveObservedTheSameEvents)
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
	client, ok := ctx.Value(clientKey{}).(*ApiClient)
	if !ok {
		return nil, errors.New("client not found in test context. Please check test definitions")
	}
	return client, nil
}

func startServer(testPort string) *Server {
	log.Println("Starting a new server")
	eventPublisher := pubsub.New[string, Event](1)
	server = NewServerWithOptions(testPort, eventPublisher, GetFS(true))
	go server.Run(testPort)
	return server
}

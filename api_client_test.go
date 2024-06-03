package mermaidlive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/Arceliar/phony"
	"github.com/go-resty/resty/v2"
)

type ApiClient struct {
	phony.Inbox
	baseUrl         string
	c               *resty.Client
	currentReader   io.ReadCloser
	currentBuffer   []byte
	currentResponse *resty.Response
	cancel          context.CancelFunc
	receivedEvents  []ReceivedEvent
}

func NewApiClient(baseUrl string) *ApiClient {
	client := &ApiClient{
		baseUrl:       baseUrl,
		c:             resty.New().SetBaseURL(baseUrl).SetTimeout(1 * time.Second),
		currentBuffer: make([]byte, 0),
		cancel:        func() {},
	}
	go client.subscribeToEvents()
	return client
}

func (a *ApiClient) WaitForState(expectedState string) error {
	var err error
	var found bool
	log.Printf("Waiting for state '%s' ...", expectedState)
	for i := 0; i < 10; i++ {
		phony.Block(a, func() {
			res, e := a.c.
				R().
				Get("/machine/state")
			if e != nil {
				err = e
				return
			}
			foundState := strings.TrimSpace(string(res.Body()))
			if expectedState == foundState {
				found = true
				return
			}
			log.Printf("Expected state '%v' but found '%v', sleeping to retry...", expectedState, foundState)
			time.Sleep(100 * time.Millisecond)
		})
	}
	if !found {
		return fmt.Errorf("could not find state: %v", expectedState)
	}
	return err
}

func (a *ApiClient) PostCommand(command string) error {
	var err error
	phony.Block(a, func() {
		log.Printf("Requesting start ...")
		_, e := a.c.
			R().
			Post("/commands/" + command)
		if e != nil {
			err = e
			return
		}
	})
	return err
}

func (a *ApiClient) WaitForEventSeen(eventName string) error {
	var found bool
	log.Printf("Waiting for event '%s' ...", eventName)
	for i := 0; i < 10; i++ {
		phony.Block(a, func() {
			pos := slices.IndexFunc(a.receivedEvents, func(c ReceivedEvent) bool {
				return c.Name == eventName
			})
			if pos != 1 {
				found = true
			}
		})
		time.Sleep(100 * time.Millisecond)
		if found {
			return nil
		}
	}
	if !found {
		return fmt.Errorf("Gave up waiting for event: %v", eventName)
	}
	return nil
}

func (a *ApiClient) BaseUrl() string {
	return a.baseUrl
}

func (a *ApiClient) subscribeToEvents() {
	a.Act(a, func() {
		a.tryCloseCurrentReaderSync()

		ctx, cancel := context.WithCancel(context.Background())
		a.cancel = cancel

		var err error
		a.currentResponse, err = a.c.
			R().
			SetContext(ctx).
			SetDoNotParseResponse(true).
			Get("/events")
		if err != nil {
			log.Println("Error subscribing to events:", err)
		}
		a.currentReader = a.currentResponse.RawResponse.Body
		a.scheduleNextReadSync()
	})
}

func (a *ApiClient) readNextEvent() {
	a.Act(a, func() {
		if a.currentReader == nil {
			log.Println("No open request")
			return
		}

		buf := make([]byte, 1024)
		n, err := a.currentReader.Read(buf)
		if err == io.EOF {
			a.Close()
			return
		}
		if err != nil {
			a.Close()
			log.Println("error reading event stream", err)
			return
		}
		a.currentBuffer = append(a.currentBuffer, buf[:n]...)
		err = a.tryExtractEventsSync()
		if err != nil {
			log.Println(err)
			return
		}
	})
}

func (a *ApiClient) scheduleNextReadSync() {
	go func() {
		time.Sleep(10 * time.Millisecond)
		a.readNextEvent()
	}()
}

func (a *ApiClient) Close() {
	a.cancel()
	a.Act(a, func() {
		a.tryCloseCurrentReaderSync()
	})
}

func (a *ApiClient) tryCloseCurrentReaderSync() {
	if a.currentReader == nil {
		return
	}
	a.currentReader.Close()
	a.currentReader = nil
	a.currentBuffer = make([]byte, 0)
	a.currentResponse = nil
}

func (a *ApiClient) tryExtractEventsSync() error {
	for {
		newlinePos := slices.IndexFunc(a.currentBuffer, func(c byte) bool {
			return c == '\n'
		})

		if newlinePos == -1 {
			break
		}

		eventLine := a.currentBuffer[:newlinePos]
		a.currentBuffer = a.currentBuffer[newlinePos+1:]

		var event ReceivedEvent
		err := json.Unmarshal(eventLine, &event)
		if err != nil {
			// fail fast
			return fmt.Errorf("Error unmarshalling event: %v", err)
		}
		a.receivedEvents = append(a.receivedEvents, event)
		// log.Println("Received event:", event)
	}
	a.scheduleNextReadSync()
	return nil
}

type ReceivedEvent struct {
	Timestamp  time.Time              `json:"timestamp"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
}

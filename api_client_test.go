package mermaidlive

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

type ApiClient struct {
	baseUrl string
	c       *resty.Client
}

func NewApiClient(baseUrl string) *ApiClient {
	return &ApiClient{
		baseUrl: baseUrl,
		c:       resty.New().SetBaseURL(baseUrl).SetTimeout(1 * time.Second),
	}
}

func (a *ApiClient) WaitForState(expectedState string) error {
	log.Printf("Waiting for state '%s' ...", expectedState)
	for i := 0; i < 10; i++ {
		res, err := a.c.
			R().
			Get("/machine/state")
		if err != nil {
			return err
		}
		foundState := strings.TrimSpace(string(res.Body()))
		if expectedState == foundState {
			return nil
		}
		log.Printf("Expected state '%v' but found '%v', sleeping to retry...", expectedState, foundState)
		log.Println("Body:", res.Body())
		time.Sleep(1 * time.Second)
	}
	return errors.New("could not find state " + expectedState)
}

func (a *ApiClient) PostCommand(command string) error {
	log.Printf("Requesting start ...")
	_, err := a.c.
		R().
		Post("/commands/" + command)
	if err != nil {
		return err
	}
	return nil
}

func (a *ApiClient) BaseUrl() string {
	return a.baseUrl
}

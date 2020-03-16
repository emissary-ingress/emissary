package longpoll

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
)

// Based on https://github.com/jcuga/golongpoll/blob/master/go-client/glpclient/client.go

// PollResponse is what the server sends back
type PollResponse struct {
	Events    []PollEvent `json:"events"`
	Timestamp int64       `json:"timestamp"`
}

// PollEvent represent one event in the stream
type PollEvent struct {
	Timestamp int64           `json:"timestamp"` // in ms to match JS Date.getTime()
	Category  string          `json:"category"`
	Data      json.RawMessage `json:"data"` // anything that json.Marshal() accepts
}

// Client represent the long poll client configuration and state
type Client struct {
	url               *url.URL
	category          string
	Timeout           int // polling timeout
	Reattempt         int // time between reconnects after failure
	EventsChan        chan *PollEvent
	runID             uint64 // Something something
	HTTPClient        *http.Client
	BasicAuthUsername string
	BasicAuthPassword string

	// nil to disable logging
	Logger interface {
		Printf(string, ...interface{})
		Println(...interface{})
	}
}

// NewClient creates a long poll client
func NewClient(url *url.URL, appName string, appNamespace string) *Client {
	return &Client{
		url:        url,
		category:   fmt.Sprintf("%s/%s", appNamespace, appName),
		Timeout:    120,
		Reattempt:  30,
		EventsChan: make(chan *PollEvent),
		HTTPClient: &http.Client{},
		Logger:     nil,
	}
}

// Start the polling of the events on the URL defined in the client
// Will send the events in the EventsChan of the client
func (c *Client) Start() {
	u := c.url
	if c.Logger != nil {
		c.Logger.Println("Now observing changes on", u.String())
	}

	atomic.AddUint64(&(c.runID), 1)
	currentRunID := atomic.LoadUint64(&(c.runID))

	go func(runID uint64, u *url.URL) {
		since := int64(0)
		for {
			pr, err := c.fetchEvents(since)

			if err != nil {
				if c.Logger != nil {
					c.Logger.Println(err)
					c.Logger.Printf("Reattempting to connect to %s in %d seconds", u.String(), c.Reattempt)
				}
				c.EventsChan <- nil
				time.Sleep(time.Duration(c.Reattempt) * time.Second)
				continue
			}

			// We check that its still the same runID as when this goroutine was started
			clientRunID := atomic.LoadUint64(&(c.runID))
			if clientRunID != runID {
				if c.Logger != nil {
					c.Logger.Printf("Client on URL %s has been stopped, not sending events", u.String())
				}
				return
			}

			if len(pr.Events) > 0 {
				if c.Logger != nil {
					c.Logger.Println("Got", len(pr.Events), "event(s) from URL", u.String())
				}
				for _, event := range pr.Events {
					since = event.Timestamp
					c.EventsChan <- &event
				}
			} else if pr.Timestamp > since {
				// Only push timestamp forward if its greater than the last we checked
				since = pr.Timestamp
			}
		}
	}(currentRunID, u)
}

// Stop the client for some reason
func (c *Client) Stop() {
	// Changing the runID will have any previous goroutine ignore any events it may receive
	atomic.AddUint64(&(c.runID), 1)
}

// Call the longpoll server to get the events since a specific timestamp
func (c Client) fetchEvents(since int64) (PollResponse, error) {
	u := c.url
	if c.Logger != nil {
		c.Logger.Println("Checking for events since", since, "on URL", u.String())
	}

	query := u.Query()
	query.Set("category", c.category)
	query.Set("since_time", fmt.Sprintf("%d", since))
	query.Set("timeout", fmt.Sprintf("%d", c.Timeout))
	u.RawQuery = query.Encode()

	req, _ := http.NewRequest("GET", u.String(), nil)
	if c.BasicAuthUsername != "" && c.BasicAuthPassword != "" {
		req.SetBasicAuth(c.BasicAuthUsername, c.BasicAuthPassword)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		msg := fmt.Sprintf("Error while connecting to %s to observe changes. Error was: %s", u, err)
		return PollResponse{}, errors.New(msg)
	}

	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Wrong status code received from longpoll server: %d", resp.StatusCode)
		return PollResponse{}, errors.New(msg)
	}

	decoder := json.NewDecoder(resp.Body)
	defer resp.Body.Close()

	var pr PollResponse
	err = decoder.Decode(&pr)
	if err != nil {
		if c.Logger != nil {
			c.Logger.Printf("Error while decoding poll response: %s", err)
		}
		return PollResponse{}, err
	}

	return pr, nil
}

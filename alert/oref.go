package alert

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	orefURL       = "https://www.oref.org.il/WarningMessages/alert/alerts.json"
	clientTimeout = 10 * time.Second
)

// OrefPoller polls the Pikud HaOref API for alerts.
type OrefPoller struct {
	client       *http.Client
	pollInterval time.Duration
	ch           chan Alert
	done         chan struct{}
}

// NewOrefPoller creates and starts a polling alert source with the given interval.
func NewOrefPoller(pollInterval time.Duration) *OrefPoller {
	p := &OrefPoller{
		client:       &http.Client{Timeout: clientTimeout},
		pollInterval: pollInterval,
		ch:           make(chan Alert, 1),
		done:         make(chan struct{}),
	}
	go p.loop()
	return p
}

func (p *OrefPoller) Alerts() <-chan Alert { return p.ch }

func (p *OrefPoller) Close() {
	close(p.done)
}

func (p *OrefPoller) loop() {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.done:
			return
		case <-ticker.C:
			alert, err := p.fetch()
			if err != nil {
				log.Printf("[oref] fetch error: %v", err)
				continue
			}
			if alert != nil {
				select {
				case p.ch <- *alert:
				default:
					// Drop if consumer is slow — avoids blocking the poll loop.
				}
			}
		}
	}
}

func (p *OrefPoller) fetch() (*Alert, error) {
	req, err := http.NewRequest("GET", orefURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", "https://www.oref.org.il/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Strip UTF-8 BOM and whitespace — the API returns BOM on empty responses.
	trimmed := strings.TrimSpace(strings.TrimPrefix(string(body), "\xef\xbb\xbf"))
	if trimmed == "" {
		return nil, nil
	}

	var a Alert
	if err := json.Unmarshal([]byte(trimmed), &a); err != nil {
		return nil, fmt.Errorf("parse error: %w (body: %q)", err, trimmed)
	}
	return &a, nil
}

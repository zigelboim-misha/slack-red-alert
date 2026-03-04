package alert

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	tzofarURL      = "wss://ws.tzevaadom.co.il/socket?platform=ANDROID"
	pingInterval   = 60 * time.Second
	reconnectDelay = 10 * time.Second
	maxReconnect   = 60 * time.Second
)

// TzofarWS connects to the Tzofar WebSocket for real-time alerts.
type TzofarWS struct {
	ch   chan Alert
	done chan struct{}
}

// NewTzofarWS creates and starts a WebSocket alert source.
func NewTzofarWS() *TzofarWS {
	t := &TzofarWS{
		ch:   make(chan Alert, 1),
		done: make(chan struct{}),
	}
	go t.loop()
	return t
}

func (t *TzofarWS) Alerts() <-chan Alert { return t.ch }

func (t *TzofarWS) Close() {
	close(t.done)
}

func (t *TzofarWS) loop() {
	delay := reconnectDelay

	for {
		select {
		case <-t.done:
			return
		default:
		}

		err := t.connect()
		if err != nil {
			log.Printf("[tzofar] connection error: %v, reconnecting in %v", err, delay)
			// Exponential backoff capped at maxReconnect.
			nextDelay := delay * 2
			if nextDelay > maxReconnect {
				nextDelay = maxReconnect
			}
			select {
			case <-t.done:
				return
			case <-time.After(delay):
			}
			delay = nextDelay
		} else {
			// Connection was established then dropped — reset backoff.
			delay = reconnectDelay
		}
	}
}

func (t *TzofarWS) connect() error {
	headers := http.Header{}
	headers.Set("Origin", "https://www.tzevaadom.co.il")
	headers.Set("User-Agent", "Mozilla/5.0 (Linux; Android 13) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36")

	conn, _, err := websocket.DefaultDialer.Dial(tzofarURL, headers)
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Println("[tzofar] connected")

	// Reset backoff on successful connection (handled by caller resetting delay after success).
	// Start pinger.
	pingDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			case <-pingDone:
				return
			case <-t.done:
				return
			}
		}
	}()
	defer close(pingDone)

	for {
		select {
		case <-t.done:
			return nil
		default:
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var a Alert
		if err := json.Unmarshal(msg, &a); err != nil {
			log.Printf("[tzofar] ignoring unparseable message: %s", string(msg))
			continue
		}

		if len(a.Data) == 0 {
			continue
		}

		select {
		case t.ch <- a:
		default:
		}
	}
}

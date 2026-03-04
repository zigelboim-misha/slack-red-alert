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
	pingInterval   = 30 * time.Second
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

		connected, err := t.connect()
		if err != nil {
			log.Printf("[tzofar] connection error: %v, reconnecting in %v", err, delay)
		}

		if connected {
			// Was connected then dropped — reset backoff.
			delay = reconnectDelay
		} else {
			// Never connected — exponential backoff capped at maxReconnect.
			select {
			case <-t.done:
				return
			case <-time.After(delay):
			}
			if delay < maxReconnect {
				delay = delay * 2
				if delay > maxReconnect {
					delay = maxReconnect
				}
			}
		}
	}
}

func (t *TzofarWS) connect() (bool, error) {
	headers := http.Header{}
	headers.Set("Origin", "https://www.tzevaadom.co.il")
	headers.Set("User-Agent", "Mozilla/5.0 (Linux; Android 13) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36")

	conn, _, err := websocket.DefaultDialer.Dial(tzofarURL, headers)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	log.Println("[tzofar] connected")
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
			return true, nil
		default:
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			return true, err
		}

		// Skip empty keepalive frames.
		if len(msg) == 0 {
			continue
		}

		var envelope struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(msg, &envelope); err != nil {
			log.Printf("[tzofar] ignoring unparseable message: %s", string(msg))
			continue
		}

		switch envelope.Type {
		case "ALERT":
			log.Printf("[tzofar] raw alert data: %s", string(envelope.Data))
			var a Alert
			if err := json.Unmarshal(envelope.Data, &a); err != nil {
				log.Printf("[tzofar] failed to parse alert data: %s", string(envelope.Data))
				continue
			}
			if len(a.Data) == 0 {
				log.Printf("[tzofar] alert has no cities, skipping")
				continue
			}
			log.Printf("[tzofar] alert: cat=%s title=%s cities=%v", a.Cat, a.Title, a.Data)
			select {
			case t.ch <- a:
			default:
			}
		default:
			log.Printf("[tzofar] %s received", envelope.Type)
		}
	}
}

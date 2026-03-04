package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mishazigelboim/slack-red-alert/alert"
	"github.com/mishazigelboim/slack-red-alert/status"
)

const (
	// How long after the last alert to auto-clear the status.
	clearDelay = 2 * time.Minute
)

func main() {
	slackToken := os.Getenv("SLACK_TOKEN")
	if slackToken == "" {
		log.Fatal("SLACK_TOKEN environment variable is required")
	}

	cities := parseCities(os.Getenv("ALERT_CITIES"))
	statusMessages := parseStatusMessages(os.Getenv("ALERT_STATUS_TEXTS"))

	startHealthServer()

	log.Printf("Monitoring cities: %v", cities)

	slackStatus := status.NewSlackStatus(slackToken)

	// Start both sources — WebSocket is primary, polling is fallback.
	tzofar := alert.NewTzofarWS()
	defer tzofar.Close()

	pollInterval := parsePollInterval(os.Getenv("POLL_INTERVAL_SECONDS"))
	oref := alert.NewOrefPoller(pollInterval)
	defer oref.Close()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	alertActive := false
	var clearTimer *time.Timer

	for {
		select {
		case a := <-tzofar.Alerts():
			if matchesCities(a, cities) {
				alertActive = setAlertStatus(slackStatus, statusMessages, alertActive)
				clearTimer = resetClearTimer(clearTimer)
			}

		case a := <-oref.Alerts():
			if matchesCities(a, cities) {
				alertActive = setAlertStatus(slackStatus, statusMessages, alertActive)
				clearTimer = resetClearTimer(clearTimer)
			}

		case <-timerChan(clearTimer):
			if alertActive {
				if err := slackStatus.Clear(); err != nil {
					log.Printf("Failed to clear status: %v", err)
				}
				alertActive = false
			}
			clearTimer = nil

		case <-sig:
			log.Println("Shutting down...")
			if alertActive {
				_ = slackStatus.Clear()
			}
			return
		}
	}
}

func parseCities(env string) []string {
	if env == "" {
		return []string{"תל אביב", "גבעתיים", "רמת גן"}
	}
	var cities []string
	for _, c := range strings.Split(env, ",") {
		if trimmed := strings.TrimSpace(c); trimmed != "" {
			cities = append(cities, trimmed)
		}
	}
	return cities
}

func matchesCities(a alert.Alert, cities []string) bool {
	for _, area := range a.Data {
		for _, city := range cities {
			if strings.Contains(area, city) {
				return true
			}
		}
	}
	return false
}

var defaultStatusMessages = []string{
	"Red Alert - seeking shelter",
	"BRB, dodging rockets",
	"In the safe room, back soon",
	"Taking cover - red alert",
	"Gone to the mamad, hold tight",
	"Rocket alert - be right back",
	"Currently sheltering in place",
}

func parseStatusMessages(env string) []string {
	if env == "" {
		return defaultStatusMessages
	}
	var msgs []string
	for _, m := range strings.Split(env, "|") {
		if trimmed := strings.TrimSpace(m); trimmed != "" {
			msgs = append(msgs, trimmed)
		}
	}
	return msgs
}

func setAlertStatus(s *status.SlackStatus, messages []string, alreadyActive bool) bool {
	if alreadyActive {
		return true
	}
	text := messages[rand.Intn(len(messages))]
	if err := s.SetAlert(text); err != nil {
		log.Printf("Failed to set status: %v", err)
		return false
	}
	return true
}

func startHealthServer() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})
	go http.ListenAndServe(":"+port, nil)
	log.Printf("Health server listening on :%s", port)
}

func parsePollInterval(env string) time.Duration {
	if env == "" {
		return 10 * time.Second
	}
	secs, err := strconv.Atoi(env)
	if err != nil || secs < 1 {
		log.Printf("Invalid POLL_INTERVAL_SECONDS=%q, using default 10s", env)
		return 10 * time.Second
	}
	return time.Duration(secs) * time.Second
}

func resetClearTimer(existing *time.Timer) *time.Timer {
	if existing != nil {
		existing.Stop()
	}
	return time.NewTimer(clearDelay)
}

// timerChan returns the timer's channel, or a nil channel (blocks forever) if timer is nil.
func timerChan(t *time.Timer) <-chan time.Time {
	if t == nil {
		return nil
	}
	return t.C
}

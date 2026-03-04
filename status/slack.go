package status

import (
	"log"
	"time"

	"github.com/slack-go/slack"
)

const statusExpiry = 10 * time.Minute

// SlackStatus manages the authenticated user's Slack custom status.
type SlackStatus struct {
	api *slack.Client
}

// NewSlackStatus creates a Slack status manager from a user OAuth token (xoxp-).
func NewSlackStatus(token string) *SlackStatus {
	return &SlackStatus{api: slack.New(token)}
}

// SetAlert sets the user's status to indicate a red alert.
func (s *SlackStatus) SetAlert(text string) error {
	expiration := time.Now().Add(statusExpiry).Unix()
	err := s.api.SetUserCustomStatus(text, ":rotating_light:", expiration)
	if err != nil {
		return err
	}
	log.Printf("[slack] status set: %s (expires in %v)", text, statusExpiry)
	return nil
}

// Clear removes the user's custom status.
func (s *SlackStatus) Clear() error {
	err := s.api.UnsetUserCustomStatus()
	if err != nil {
		return err
	}
	log.Println("[slack] status cleared")
	return nil
}

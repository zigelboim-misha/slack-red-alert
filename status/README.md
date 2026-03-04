# status

Manages the authenticated user's Slack custom status.

## `SlackStatus`

Wraps the `github.com/slack-go/slack` client to set and clear the user's profile status.

- **`SetAlert(text)`** — sets the status with the given text, a `:rotating_light:` emoji, and a 10-minute auto-expiry (safety net in case the process crashes mid-alert)
- **`Clear()`** — removes the custom status entirely

Requires a Slack **user OAuth token** (`xoxp-`) with the `users.profile:write` scope. Bot tokens cannot set user status.

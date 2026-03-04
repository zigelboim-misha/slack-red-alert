package alert

// Alert represents a red alert event from Tzofar.
type Alert struct {
	ID    string   `json:"id"`
	Cat   string   `json:"cat"`
	Title string   `json:"title"`
	Data  []string `json:"data"`
	Desc  string   `json:"desc"`
}

// Source delivers alerts on a channel.
type Source interface {
	// Alerts returns a receive-only channel that emits Alert events.
	Alerts() <-chan Alert
	// Close shuts down the source.
	Close()
}

package alert

// Alert represents a red alert event from Tzofar.
type Alert struct {
	Threat  int      `json:"threat"`
	IsDrill bool     `json:"isDrill"`
	Cities  []string `json:"cities"`
}

// Source delivers alerts on a channel.
type Source interface {
	// Alerts returns a receive-only channel that emits Alert events.
	Alerts() <-chan Alert
	// Close shuts down the source.
	Close()
}

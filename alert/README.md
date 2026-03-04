# alert

Alert sources for Israeli red alert (Tzeva Adom) events.

## Types

- **`Alert`** — represents a single alert event with ID, category, title, affected areas, and shelter instructions. City names are in Hebrew.
- **`Source`** — interface implemented by all alert sources. Exposes an `Alerts()` channel and a `Close()` method.

## Sources

### `TzofarWS`

Connects to the Tzofar WebSocket (`wss://ws.tzevaadom.co.il/socket?platform=ANDROID`) for real-time push alerts. No authentication or geo-restriction. Sends keepalive pings every 60 seconds and auto-reconnects with exponential backoff (10s → 60s cap) on disconnect.

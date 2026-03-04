# alert

Alert sources for Israeli red alert (Tzeva Adom) events.

## Types

- **`Alert`** — represents a single alert event with ID, category, title, affected areas, and shelter instructions. City names are in Hebrew.
- **`Source`** — interface implemented by all alert sources. Exposes an `Alerts()` channel and a `Close()` method.

## Sources

### `OrefPoller`

Polls the Pikud HaOref API (`oref.org.il/WarningMessages/alert/alerts.json`) at a configurable interval (default 10 seconds). Requires three HTTP headers (`Referer`, `X-Requested-With`, `User-Agent`) and an **Israeli IP address** — requests from outside Israel are blocked by Akamai WAF.

The API returns an empty body (or a UTF-8 BOM `\xef\xbb\xbf`) when no alerts are active, and a JSON object when alerts are firing.

### `TzofarWS`

Connects to the Tzofar WebSocket (`wss://ws.tzevaadom.co.il/socket?platform=ANDROID`) for real-time push alerts. No authentication or geo-restriction. Sends keepalive pings every 60 seconds and auto-reconnects with exponential backoff (10s → 60s cap) on disconnect.

package app

import "encoding/json"

// NotifyMessage is notify message structure
type NotifyMessage struct {
	Event  string     `json:"event"`
	AppID  string     `json:"app_id"`
	Config *AppConfig `json:"config,omitempty"`
}

// WSMessage is websocket message structure
type WSMessage struct {
	Action  string      `json:"action"`
	Payload interface{} `json:"payload"`
}

// WSMessageRaw is websocket raw message structure
type WSMessageRaw struct {
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload"`
}

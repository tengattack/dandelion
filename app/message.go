package app

// NotifyMessage is notify message structure
type NotifyMessage struct {
	Event  string    `json:"event"`
	AppID  string    `json:"app_id"`
	Config AppConfig `json:"config"`
}

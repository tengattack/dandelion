package main

import (
	"encoding/json"

	"../../app"
)

// HandleMessage handle dandelion messages
func HandleMessage(message string) error {
	var m app.NotifyMessage
	err := json.Unmarshal([]byte(message), &m)
	if err != nil {
		return err
	}

	switch m.Event {
	case "publish":
		fallthrough
	case "rollback":
		for _, config := range Conf.Configs {
			// TODO: check matching of host and instance_id
			if config.AppID == m.AppID {
				return CheckAppConfig(&config)
			}
		}
	}
	return nil
}

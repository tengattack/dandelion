package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/tengattack/dandelion/app"
	"github.com/tengattack/dandelion/client"
	"github.com/tengattack/dandelion/log"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleWebSocketMessage(conn *websocket.Conn, msg []byte) error {
	log.LogAccess.Debugf("websocket received message: %s", msg)
	var message app.WSMessageRaw
	err := json.Unmarshal(msg, &message)
	if err != nil {
		return err
	}
	type Status struct {
		ID          int64                 `json:"-" db:"id"`
		AppID       string                `json:"app_id" db:"app_id"`
		Host        string                `json:"host" db:"host"`
		InstanceID  string                `json:"instance_id" db:"instance_id"`
		ConfigID    int64                 `json:"config_id,omitempty" db:"config_id"`
		CommitID    string                `json:"commit_id,omitempty" db:"commit_id"`
		Status      client.InstanceStatus `json:"status" db:"status"`
		CreatedTime int64                 `json:"-" db:"created_time"`
		UpdatedTime int64                 `json:"-" db:"updated_time"`
	}
	switch message.Action {
	case "ping":
		if message.Payload != nil {
			var payload []Status
			err = json.Unmarshal(message.Payload, &payload)
			if err != nil {
				return err
			}
			for _, s := range payload {
				s.UpdatedTime = time.Now().Unix()
				_, err = DB.NamedExec("UPDATE "+TableNameInstances+
					" SET config_id = :config_id, commit_id = :commit_id, status = :status, updated_time = :updated_time"+
					" WHERE app_id = :app_id AND host = :host AND instance_id = :instance_id", &s)
				if err != nil {
					log.LogError.Errorf("update instance record failed: %v", err)
				}
			}
		}
		t := time.Now().Add(time.Second * 5)
		return conn.WriteControl(websocket.PongMessage, nil, t)
	case "status":
		var payload Status
		err = json.Unmarshal(message.Payload, &payload)
		if err != nil {
			return err
		}
		var row Status
		err = DB.Get(&row, "SELECT id, config_id, commit_id, status FROM "+TableNameInstances+
			" WHERE app_id = ? AND host = ? AND instance_id = ? LIMIT 1",
			payload.AppID, payload.Host, payload.InstanceID)
		if err == sql.ErrNoRows {
			row.AppID = payload.AppID
			row.Host = payload.Host
			row.InstanceID = payload.InstanceID
			row.Status = payload.Status
			row.CreatedTime = time.Now().Unix()
			row.UpdatedTime = row.CreatedTime
			_, err = DB.NamedExec("INSERT INTO "+TableNameInstances+" (app_id, host, instance_id, config_id, commit_id, status, created_time, updated_time)"+
				" VALUES (:app_id, :host, :instance_id, :config_id, :commit_id, :status, :created_time, :updated_time)", &row)
			if err != nil {
				return err
			}
		} else if err != nil {
			log.LogError.Errorf("get instance record failed: %v", err)
			return err
		} else {
			row.Status = payload.Status
			row.UpdatedTime = time.Now().Unix()
			if row.ConfigID != payload.ConfigID || row.CommitID != payload.CommitID {
				// update all
				row.ConfigID = payload.ConfigID
				row.CommitID = payload.CommitID
				_, err = DB.NamedExec("UPDATE "+TableNameInstances+
					" SET config_id = :config_id, commit_id = :commit_id, status = :status, updated_time = :updated_time "+
					" WHERE id = :id", &row)
			} else {
				// update status only
				_, err = DB.NamedExec("UPDATE "+TableNameInstances+
					" SET status = :status, updated_time = :updated_time "+
					" WHERE id = :id", &row)
			}
			if err != nil {
				log.LogError.Errorf("update instance record failed: %v", err)
				return err
			}
		}
	}
	return nil
}

func wsPushHandler(c *gin.Context) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.LogError.Errorf("Failed to set websocket upgrade: %+v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer conn.Close()

	for {
		t, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.LogError.Errorf("Unexpected close error: %v", err)
			}
			break
		}
		if t == websocket.TextMessage || t == websocket.BinaryMessage {
			err = handleWebSocketMessage(conn, msg)
			if err != nil {
				log.LogError.Errorf("websocket handle message error: %v", err)
			}
		}
	}
}

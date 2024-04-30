package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/tengattack/dandelion/app"
	"github.com/tengattack/dandelion/cmd/dandelion/config"
	"github.com/tengattack/tgo/logger"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type wsConn struct {
	appID      string
	instanceID string
	host       string
	configID   int64
	conn       *websocket.Conn
}

var wsConnPool map[string][]*wsConn
var wsConnPoolMutex sync.Mutex

func init() {
	wsConnPool = make(map[string][]*wsConn)
}

func updateConnPoolInfo(conn *websocket.Conn, s *app.Status) {
	wsConnPoolMutex.Lock()
	defer wsConnPoolMutex.Unlock()
	pool, ok := wsConnPool[s.AppID]
	if !ok {
		pool = make([]*wsConn, 0)
	}

	for _, c := range pool {
		if c.conn == conn {
			// found
			c.host = s.Host
			c.instanceID = s.InstanceID
			c.configID = s.ConfigID
			return
		}
	}

	pool = append(pool, &wsConn{
		appID:      s.AppID,
		host:       s.Host,
		instanceID: s.InstanceID,
		configID:   s.ConfigID,
		conn:       conn,
	})

	wsConnPool[s.AppID] = pool
}

func removeConnPoolInfo(conn *websocket.Conn) {
	wsConnPoolMutex.Lock()
	defer wsConnPoolMutex.Unlock()

	var removeKeys []string
	for k, pool := range wsConnPool {
		j := 0
		for _, c := range pool {
			if c.conn != conn {
				pool[j] = c
				j++
			}
		}
		if j == 0 {
			removeKeys = append(removeKeys, k)
		} else if j != len(pool) {
			wsConnPool[k] = pool[:j]
		}
	}
	if len(removeKeys) > 0 {
		for _, k := range removeKeys {
			delete(wsConnPool, k)
		}
	}
}

func connWrite(c *wsConn, message []byte) {
	err := c.conn.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		logger.WithFields(logger.Fields{
			"app_id":      c.appID,
			"host":        c.host,
			"instance_id": c.instanceID,
		}).Errorf("websocket conn write message error: %v", err)
		// PASS
	}
}

func notifyConn(m *app.NotifyMessage) {
	message, err := json.Marshal(m)
	if err != nil {
		logger.Errorf("encode message error: %v", err)
		return
	}
	if config.Conf.Kafka.Enabled {
		err = config.MQ.Publish(string(message))
		if err != nil {
			logger.Errorf("publish message error: %v", err)
			// PASS
		}
	}

	wsConnPoolMutex.Lock()
	defer wsConnPoolMutex.Unlock()

	pool, ok := wsConnPool[m.AppID]
	if ok {
		for _, c := range pool {
			go connWrite(c, message)
		}
	}
}

func handleWebSocketMessage(conn *websocket.Conn, msg []byte) error {
	logger.Debugf("websocket received message: %s", msg)
	var message app.WSMessageRaw
	err := json.Unmarshal(msg, &message)
	if err != nil {
		return err
	}

	switch message.Action {
	case "ping":
		if message.Payload != nil {
			var payload []app.Status
			err = json.Unmarshal(message.Payload, &payload)
			if err != nil {
				return err
			}
			for _, s := range payload {
				s.UpdatedTime = time.Now().Unix()
				updateConnPoolInfo(conn, &s)

				_, err = config.DB.NamedExec("UPDATE "+TableNameInstances()+
					" SET config_id = :config_id, commit_id = :commit_id, status = :status, updated_time = :updated_time"+
					" WHERE app_id = :app_id AND host = :host AND instance_id = :instance_id", &s)
				if err != nil {
					logger.Errorf("update instance record failed: %v", err)
					// PASS
				}
			}
		}
		t := time.Now().Add(time.Second * 5)
		return conn.WriteControl(websocket.PongMessage, nil, t)
	case "status":
		var payload app.Status
		err = json.Unmarshal(message.Payload, &payload)
		if err != nil {
			return err
		}
		updateConnPoolInfo(conn, &payload)

		var row app.Status
		err = config.DB.Get(&row, "SELECT id, config_id, commit_id, status FROM "+TableNameInstances()+
			" WHERE app_id = ? AND host = ? AND instance_id = ? LIMIT 1",
			payload.AppID, payload.Host, payload.InstanceID)
		if err == sql.ErrNoRows {
			row.AppID = payload.AppID
			row.Host = payload.Host
			row.InstanceID = payload.InstanceID
			row.Status = payload.Status
			row.CreatedTime = time.Now().Unix()
			row.UpdatedTime = row.CreatedTime
			_, err = config.DB.NamedExec("INSERT INTO "+TableNameInstances()+" (app_id, host, instance_id, config_id, commit_id, status, created_time, updated_time)"+
				" VALUES (:app_id, :host, :instance_id, :config_id, :commit_id, :status, :created_time, :updated_time)", &row)
			if err != nil {
				logger.Errorf("create new instance record failed: %v", err)
				return err
			}
		} else if err != nil {
			logger.Errorf("get instance record failed: %v", err)
			return err
		} else {
			row.Status = payload.Status
			row.UpdatedTime = time.Now().Unix()
			if row.ConfigID != payload.ConfigID || row.CommitID != payload.CommitID {
				// update all
				row.ConfigID = payload.ConfigID
				row.CommitID = payload.CommitID
				_, err = config.DB.NamedExec("UPDATE "+TableNameInstances()+
					" SET config_id = :config_id, commit_id = :commit_id, status = :status, updated_time = :updated_time "+
					" WHERE id = :id", &row)
			} else {
				// update status only
				_, err = config.DB.NamedExec("UPDATE "+TableNameInstances()+
					" SET status = :status, updated_time = :updated_time "+
					" WHERE id = :id", &row)
			}
			if err != nil {
				logger.Errorf("update instance record failed: %v", err)
				return err
			}
		}
	}
	return nil
}

func wsPushHandler(c *gin.Context) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Errorf("failed to set websocket upgrade: %+v", err)
		abortWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer func() {
		removeConnPoolInfo(conn)
		conn.Close()
	}()

	for {
		t, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				logger.Errorf("unexpected close error: %v", err)
			}
			break
		}
		if t == websocket.TextMessage || t == websocket.BinaryMessage {
			err = handleWebSocketMessage(conn, msg)
			if err != nil {
				logger.Errorf("websocket handle message error: %v", err)
			}
		}
	}
}

package controllers

import (
	"sort"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tengattack/dandelion/app"
)

func TestConnPool(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	wsConnPool = make(map[string][]*wsConn)

	conn1 := new(websocket.Conn)
	conn2 := new(websocket.Conn)
	conn3 := new(websocket.Conn)

	s1 := &app.Status{AppID: "s1"}
	s2 := &app.Status{AppID: "s2"}
	updateConnPoolInfo(conn1, s1)
	updateConnPoolInfo(conn2, s2)
	updateConnPoolInfo(conn3, s1)

	require.Len(wsConnPool, 2)
	var keys []string
	for k := range wsConnPool {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	require.Equal([]string{"s1", "s2"}, keys)
	assert.Len(wsConnPool["s1"], 2)
	assert.Len(wsConnPool["s2"], 1)

	removeConnPoolInfo(conn3)
	assert.Len(wsConnPool["s1"], 1)

	// remove not exists
	removeConnPoolInfo(conn3)
	assert.Len(wsConnPool["s1"], 1)

	// remove all conn for an appID
	removeConnPoolInfo(conn1)
	_, ok := wsConnPool["s1"]
	assert.False(ok, "conn pool for s1 should not exists")
}

func TestHandleWebSocketMessage(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	conn1 := new(websocket.Conn)

	err := handleWebSocketMessage(conn1, []byte(``))
	assert.Error(err)

	// action status
	err = handleWebSocketMessage(conn1,
		[]byte(`{"action":"status","payload":""}`))
	assert.Error(err)

	err = handleWebSocketMessage(conn1,
		[]byte(`{"action":"status","payload":{"app_id":"s1","host":"host1","instance_id":"instance1","config_id":1,"status":1}}`))
	require.NoError(err)

	err = handleWebSocketMessage(conn1,
		[]byte(`{"action":"status","payload":{"app_id":"s1","host":"host1","instance_id":"instance1","config_id":2,"status":0}}`))
	require.NoError(err)

	err = handleWebSocketMessage(conn1,
		[]byte(`{"action":"status","payload":{"app_id":"s1","host":"host1","instance_id":"instance1","config_id":2,"status":1}}`))
	require.NoError(err)

	// action ping
	err = handleWebSocketMessage(conn1,
		[]byte(`{"action":"ping"}`))
	assert.EqualError(err, "websocket: write timeout")

	err = handleWebSocketMessage(conn1,
		[]byte(`{"action":"ping","payload":[{"app_id":"s1","host":"host1","instance_id":"instance1","config_id":2,"status":1}]}`))
	assert.EqualError(err, "websocket: write timeout")
}

package controllers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tengattack/dandelion/cmd/dandelion/config"
)

func TestMain(m *testing.M) {
	config.InitTest()
	os.Exit(m.Run())
}

func TestIndexHandler(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	os.Setenv("DEPLOY_ENV", "test")

	var err error
	h := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(h)
	c.Request, err = http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(err)

	indexHandler(c)

	require.NotNil(h.Body)
	assert.Contains(h.Body.String(), `window.PUBLIC_URL = "https://dandelion.to/"`)
	assert.Contains(h.Body.String(), `window.DEPLOY_ENV = "test"`)
}

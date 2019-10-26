package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetLogger(t *testing.T) {
	assert := assert.New(t)

	SetLogger(nil)
	assert.NotNil(clientLogger)
	clientLogger.Debugf("debug")
	clientLogger.Infof("info")
	clientLogger.Errorf("error")
}

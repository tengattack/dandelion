package repository

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tengattack/dandelion/log"
	tlog "github.com/tengattack/tgo/log"
)

func clearTestRepo() {
	_ = os.RemoveAll("./repo1")
}

func TestMain(m *testing.M) {
	err := log.InitLog(tlog.DefaultConfig)
	if err != nil {
		panic(err)
	}
	clearTestRepo()
	code := m.Run()
	clearTestRepo()
	os.Exit(code)
}

func TestInitRepository(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	r, err := InitRepository(&Config{
		RepositoryPath: "./repo1",
		RemoteURL:      "https://github.com/tengattack/playground",
	})
	require.NoError(err)
	assert.NotNil(r)
}

package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tengattack/dandelion/cmd/dandelion/config"
)

func TestLessDashVersion(t *testing.T) {
	assert := assert.New(t)

	assert.False(lessDashVersion("1.2.3", "1.2.3"))
	assert.True(lessDashVersion("1.2.2-2", "1.2.3-11"))
	assert.True(lessDashVersion("1.2.3-2", "1.2.3-11"))
	assert.False(lessDashVersion("1.2.3-22", "1.2.3-11"))
	assert.False(lessDashVersion("1.2.4-22", "1.2.3-11"))
	assert.False(lessDashVersion("1.2.4-2", "1.2.3-11"))
	assert.False(lessDashVersion("1.2.3-2", "1.2.3-geoip"))
	assert.False(lessDashVersion("1.2.3-gepip", "1.2.3"))
	assert.False(lessDashVersion("1.2.3-1", "1.2.3"))
	assert.True(lessDashVersion("1.2.3", "1.2.3-geoip"))
	assert.True(lessDashVersion("1.2.3", "1.2.3-1"))
}

func TestHarborListTags(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	var conf *config.SectionRegistry
	var catalog string
	if conf == nil || catalog == "" {
		t.Skip()
	}

	c := NewClient(conf)
	tags, err := c.harborListTags(catalog)
	require.NoError(err)
	assert.NotEmpty(tags.Tags)
}

func TestRegistryListTags(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	var conf *config.SectionRegistry
	if conf == nil {
		t.Skip()
	}

	c := NewClient(conf)
	tags, err := c.registryListTags("dandelion")
	require.NoError(err)
	assert.NotEmpty(tags.Tags)
}

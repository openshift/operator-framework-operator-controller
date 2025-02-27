package source

import (
	"archive/tar"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/rand"
)

func TestApplyLayerFilter(t *testing.T) {
	h := tar.Header{
		Name: "foo/bar",
		Mode: 0000,
		Uid:  rand.Int(),
		Gid:  rand.Int(),
		Xattrs: map[string]string{ //nolint:staticcheck
			"foo": "bar",
		},
		PAXRecords: map[string]string{
			"fizz": "buzz",
		},
	}
	ok, err := applyLayerFilter()(&h)
	require.NoError(t, err)
	assert.True(t, ok)

	assert.Equal(t, "foo/bar", h.Name)
	assert.Equal(t, int64(0700), h.Mode)
	assert.Equal(t, os.Getuid(), h.Uid)
	assert.Equal(t, os.Getgid(), h.Gid)
	assert.Nil(t, h.PAXRecords)
	assert.Nil(t, h.Xattrs) //nolint:staticcheck
}

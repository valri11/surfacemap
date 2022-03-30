package cmd

import (
	"testing"

	lrucache "github.com/hashicorp/golang-lru"
	"github.com/stretchr/testify/assert"
)

func Test_cache_1(t *testing.T) {
	tileCache, err := lrucache.New(5)
	assert.Nil(t, err)

	data := []byte{1, 2, 3, 45}

	key1 := "key1"
	tileCache.Add(key1, data)

	dat1, ok := tileCache.Get(key1)
	assert.True(t, ok)

	dat2, ok := dat1.([]byte)
	assert.True(t, ok)

	assert.Equal(t, len(data), len(dat2))
}

package service_test

import (
	"testing"
	"time"

	"github.com/digitalcircle-com-br/service"
	"github.com/stretchr/testify/assert"
)

func TestDataAccess(t *testing.T) {
	service.Init("test")
	err := service.DataSet("a", "123", 2)
	assert.NoError(t, err)
	v, err := service.DataGet("a")
	assert.NoError(t, err)
	assert.Equal(t, "123", v)
	time.Sleep(time.Second * 3)
	v, err = service.DataGet("a")
	assert.Error(t, err)
	assert.Empty(t, v)
	i, err := service.DataHSet("amap", "k1", "v1", "k2", "v2")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), i)
	v, err = service.DataHGet("amap", "k1")
	assert.NoError(t, err)
	assert.Equal(t, "v1", v)
	m, err := service.DataHGetAll("amap")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(m))
	_, err = service.DataDel("amap")
	assert.NoError(t, err)
	m, err = service.DataHGetAll("amap")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(m))
}

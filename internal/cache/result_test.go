package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/go-dev-frame/sponge/pkg/gotest"
	"github.com/go-dev-frame/sponge/pkg/utils"

	"lol/internal/database"
	"lol/internal/model"
)

func newResultCache() *gotest.Cache {
	record1 := &model.Result{}
	record1.ID = 1
	record2 := &model.Result{}
	record2.ID = 2
	testData := map[string]interface{}{
		utils.Uint64ToStr(record1.ID): record1,
		utils.Uint64ToStr(record2.ID): record2,
	}

	c := gotest.NewCache(testData)
	c.ICache = NewResultCache(&database.CacheType{
		CType: "redis",
		Rdb:   c.RedisClient,
	})
	return c
}

func Test_resultCache_Set(t *testing.T) {
	c := newResultCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.Result)
	err := c.ICache.(ResultCache).Set(c.Ctx, record.ID, record, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	// nil data
	err = c.ICache.(ResultCache).Set(c.Ctx, 0, nil, time.Hour)
	assert.NoError(t, err)
}

func Test_resultCache_Get(t *testing.T) {
	c := newResultCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.Result)
	err := c.ICache.(ResultCache).Set(c.Ctx, record.ID, record, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	got, err := c.ICache.(ResultCache).Get(c.Ctx, record.ID)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, record, got)

	// zero key error
	_, err = c.ICache.(ResultCache).Get(c.Ctx, 0)
	assert.Error(t, err)
}

func Test_resultCache_MultiGet(t *testing.T) {
	c := newResultCache()
	defer c.Close()

	var testData []*model.Result
	for _, data := range c.TestDataSlice {
		testData = append(testData, data.(*model.Result))
	}

	err := c.ICache.(ResultCache).MultiSet(c.Ctx, testData, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	got, err := c.ICache.(ResultCache).MultiGet(c.Ctx, c.GetIDs())
	if err != nil {
		t.Fatal(err)
	}

	expected := c.GetTestData()
	for k, v := range expected {
		assert.Equal(t, got[utils.StrToUint64(k)], v.(*model.Result))
	}
}

func Test_resultCache_MultiSet(t *testing.T) {
	c := newResultCache()
	defer c.Close()

	var testData []*model.Result
	for _, data := range c.TestDataSlice {
		testData = append(testData, data.(*model.Result))
	}

	err := c.ICache.(ResultCache).MultiSet(c.Ctx, testData, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_resultCache_Del(t *testing.T) {
	c := newResultCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.Result)
	err := c.ICache.(ResultCache).Del(c.Ctx, record.ID)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_resultCache_SetCacheWithNotFound(t *testing.T) {
	c := newResultCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.Result)
	err := c.ICache.(ResultCache).SetPlaceholder(c.Ctx, record.ID)
	if err != nil {
		t.Fatal(err)
	}
	b := c.ICache.(ResultCache).IsPlaceholderErr(err)
	t.Log(b)
}

func TestNewResultCache(t *testing.T) {
	c := NewResultCache(&database.CacheType{
		CType: "",
	})
	assert.Nil(t, c)
	c = NewResultCache(&database.CacheType{
		CType: "memory",
	})
	assert.NotNil(t, c)
	c = NewResultCache(&database.CacheType{
		CType: "redis",
	})
	assert.NotNil(t, c)
}

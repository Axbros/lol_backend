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

func newSmsHistoryCache() *gotest.Cache {
	record1 := &model.SmsHistory{}
	record1.ID = 1
	record2 := &model.SmsHistory{}
	record2.ID = 2
	testData := map[string]interface{}{
		utils.Uint64ToStr(record1.ID): record1,
		utils.Uint64ToStr(record2.ID): record2,
	}

	c := gotest.NewCache(testData)
	c.ICache = NewSmsHistoryCache(&database.CacheType{
		CType: "redis",
		Rdb:   c.RedisClient,
	})
	return c
}

func Test_smsHistoryCache_Set(t *testing.T) {
	c := newSmsHistoryCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.SmsHistory)
	err := c.ICache.(SmsHistoryCache).Set(c.Ctx, record.ID, record, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	// nil data
	err = c.ICache.(SmsHistoryCache).Set(c.Ctx, 0, nil, time.Hour)
	assert.NoError(t, err)
}

func Test_smsHistoryCache_Get(t *testing.T) {
	c := newSmsHistoryCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.SmsHistory)
	err := c.ICache.(SmsHistoryCache).Set(c.Ctx, record.ID, record, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	got, err := c.ICache.(SmsHistoryCache).Get(c.Ctx, record.ID)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, record, got)

	// zero key error
	_, err = c.ICache.(SmsHistoryCache).Get(c.Ctx, 0)
	assert.Error(t, err)
}

func Test_smsHistoryCache_MultiGet(t *testing.T) {
	c := newSmsHistoryCache()
	defer c.Close()

	var testData []*model.SmsHistory
	for _, data := range c.TestDataSlice {
		testData = append(testData, data.(*model.SmsHistory))
	}

	err := c.ICache.(SmsHistoryCache).MultiSet(c.Ctx, testData, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	got, err := c.ICache.(SmsHistoryCache).MultiGet(c.Ctx, c.GetIDs())
	if err != nil {
		t.Fatal(err)
	}

	expected := c.GetTestData()
	for k, v := range expected {
		assert.Equal(t, got[utils.StrToUint64(k)], v.(*model.SmsHistory))
	}
}

func Test_smsHistoryCache_MultiSet(t *testing.T) {
	c := newSmsHistoryCache()
	defer c.Close()

	var testData []*model.SmsHistory
	for _, data := range c.TestDataSlice {
		testData = append(testData, data.(*model.SmsHistory))
	}

	err := c.ICache.(SmsHistoryCache).MultiSet(c.Ctx, testData, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_smsHistoryCache_Del(t *testing.T) {
	c := newSmsHistoryCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.SmsHistory)
	err := c.ICache.(SmsHistoryCache).Del(c.Ctx, record.ID)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_smsHistoryCache_SetCacheWithNotFound(t *testing.T) {
	c := newSmsHistoryCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.SmsHistory)
	err := c.ICache.(SmsHistoryCache).SetPlaceholder(c.Ctx, record.ID)
	if err != nil {
		t.Fatal(err)
	}
	b := c.ICache.(SmsHistoryCache).IsPlaceholderErr(err)
	t.Log(b)
}

func TestNewSmsHistoryCache(t *testing.T) {
	c := NewSmsHistoryCache(&database.CacheType{
		CType: "",
	})
	assert.Nil(t, c)
	c = NewSmsHistoryCache(&database.CacheType{
		CType: "memory",
	})
	assert.NotNil(t, c)
	c = NewSmsHistoryCache(&database.CacheType{
		CType: "redis",
	})
	assert.NotNil(t, c)
}

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

func newPaymentHistoryCache() *gotest.Cache {
	record1 := &model.PaymentHistory{}
	record1.ID = 1
	record2 := &model.PaymentHistory{}
	record2.ID = 2
	testData := map[string]interface{}{
		utils.Uint64ToStr(record1.ID): record1,
		utils.Uint64ToStr(record2.ID): record2,
	}

	c := gotest.NewCache(testData)
	c.ICache = NewPaymentHistoryCache(&database.CacheType{
		CType: "redis",
		Rdb:   c.RedisClient,
	})
	return c
}

func Test_paymentHistoryCache_Set(t *testing.T) {
	c := newPaymentHistoryCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.PaymentHistory)
	err := c.ICache.(PaymentHistoryCache).Set(c.Ctx, record.ID, record, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	// nil data
	err = c.ICache.(PaymentHistoryCache).Set(c.Ctx, 0, nil, time.Hour)
	assert.NoError(t, err)
}

func Test_paymentHistoryCache_Get(t *testing.T) {
	c := newPaymentHistoryCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.PaymentHistory)
	err := c.ICache.(PaymentHistoryCache).Set(c.Ctx, record.ID, record, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	got, err := c.ICache.(PaymentHistoryCache).Get(c.Ctx, record.ID)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, record, got)

	// zero key error
	_, err = c.ICache.(PaymentHistoryCache).Get(c.Ctx, 0)
	assert.Error(t, err)
}

func Test_paymentHistoryCache_MultiGet(t *testing.T) {
	c := newPaymentHistoryCache()
	defer c.Close()

	var testData []*model.PaymentHistory
	for _, data := range c.TestDataSlice {
		testData = append(testData, data.(*model.PaymentHistory))
	}

	err := c.ICache.(PaymentHistoryCache).MultiSet(c.Ctx, testData, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	got, err := c.ICache.(PaymentHistoryCache).MultiGet(c.Ctx, c.GetIDs())
	if err != nil {
		t.Fatal(err)
	}

	expected := c.GetTestData()
	for k, v := range expected {
		assert.Equal(t, got[utils.StrToUint64(k)], v.(*model.PaymentHistory))
	}
}

func Test_paymentHistoryCache_MultiSet(t *testing.T) {
	c := newPaymentHistoryCache()
	defer c.Close()

	var testData []*model.PaymentHistory
	for _, data := range c.TestDataSlice {
		testData = append(testData, data.(*model.PaymentHistory))
	}

	err := c.ICache.(PaymentHistoryCache).MultiSet(c.Ctx, testData, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_paymentHistoryCache_Del(t *testing.T) {
	c := newPaymentHistoryCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.PaymentHistory)
	err := c.ICache.(PaymentHistoryCache).Del(c.Ctx, record.ID)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_paymentHistoryCache_SetCacheWithNotFound(t *testing.T) {
	c := newPaymentHistoryCache()
	defer c.Close()

	record := c.TestDataSlice[0].(*model.PaymentHistory)
	err := c.ICache.(PaymentHistoryCache).SetPlaceholder(c.Ctx, record.ID)
	if err != nil {
		t.Fatal(err)
	}
	b := c.ICache.(PaymentHistoryCache).IsPlaceholderErr(err)
	t.Log(b)
}

func TestNewPaymentHistoryCache(t *testing.T) {
	c := NewPaymentHistoryCache(&database.CacheType{
		CType: "",
	})
	assert.Nil(t, c)
	c = NewPaymentHistoryCache(&database.CacheType{
		CType: "memory",
	})
	assert.NotNil(t, c)
	c = NewPaymentHistoryCache(&database.CacheType{
		CType: "redis",
	})
	assert.NotNil(t, c)
}

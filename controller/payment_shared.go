package controller

import (
	"net/url"
	"strconv"
	"sync"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/roseforljh/opencrab/setting/operation_setting"
)

const (
	PaymentMethodStripe = "stripe"
	PaymentMethodCreem  = "creem"
)

type CreemProduct struct {
	ProductId string  `json:"productId"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Currency  string  `json:"currency"`
	Quota     int64   `json:"quota"`
}

func GetEpayClient() *epay.Client {
	if operation_setting.PayAddress == "" || operation_setting.EpayId == "" || operation_setting.EpayKey == "" {
		return nil
	}
	withUrl, err := epay.NewClient(&epay.Config{
		PartnerID: operation_setting.EpayId,
		Key:       operation_setting.EpayKey,
	}, operation_setting.PayAddress)
	if err != nil {
		return nil
	}
	return withUrl
}

func genCreemLink(referenceId string, product *CreemProduct, email string, username string) (string, error) {
	u, _ := url.Parse("https://www.creem.io/checkout")
	q := u.Query()
	q.Set("order_id", referenceId)
	if product != nil {
		q.Set("product_id", product.ProductId)
		q.Set("product_name", product.Name)
		q.Set("price", strconv.FormatFloat(product.Price, 'f', 2, 64))
		q.Set("currency", product.Currency)
	}
	if email != "" {
		q.Set("email", email)
	}
	if username != "" {
		q.Set("username", username)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

var orderLocks sync.Map
var createLock sync.Mutex

type refCountedMutex struct {
	mu       sync.Mutex
	refCount int
}

func LockOrder(tradeNo string) {
	createLock.Lock()
	var rcm *refCountedMutex
	if v, ok := orderLocks.Load(tradeNo); ok {
		rcm = v.(*refCountedMutex)
	} else {
		rcm = &refCountedMutex{}
		orderLocks.Store(tradeNo, rcm)
	}
	rcm.refCount++
	createLock.Unlock()
	rcm.mu.Lock()
}

func UnlockOrder(tradeNo string) {
	v, ok := orderLocks.Load(tradeNo)
	if !ok {
		return
	}
	rcm := v.(*refCountedMutex)
	rcm.mu.Unlock()

	createLock.Lock()
	rcm.refCount--
	if rcm.refCount == 0 {
		orderLocks.Delete(tradeNo)
	}
	createLock.Unlock()
}

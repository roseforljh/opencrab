package model

type TopUp struct {
	Id            int     `json:"id"`
	UserId        int     `json:"user_id" gorm:"index"`
	Amount        int64   `json:"amount"`
	Money         float64 `json:"money"`
	TradeNo       string  `json:"trade_no" gorm:"unique;type:varchar(255);index"`
	PaymentMethod string  `json:"payment_method" gorm:"type:varchar(50)"`
	CreateTime    int64   `json:"create_time"`
	CompleteTime  int64   `json:"complete_time"`
	Status        string  `json:"status"`
}

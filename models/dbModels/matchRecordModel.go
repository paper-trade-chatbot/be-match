package dbModels

import (
	"time"

	"github.com/shopspring/decimal"
)

type TransactionType int

const (
	TransactionType_NONE          TransactionType = iota
	TransactionType_OpenPosition                  // 開倉
	TransactionType_ClosePosition                 // 關倉
)

type MatchStatus int

const (
	MatchStatus_None       MatchStatus = iota
	MatchStatus_Pending                // 待處理
	MatchStatus_Failed                 // 失敗
	MatchStatus_Finished               // 完成
	MatchStatus_Cancelled              // 取消
	MatchStatus_Rollbacked             // 回滾
)

type TradeType int

const (
	TradeType_None TradeType = iota
	TradeType_Buy
	TradeType_Sell
)

type MatchRecordModel struct {
	ID              uint64          `gorm:"column:id; primary_key"`
	OrderID         uint64          `gorm:"column:order_id"`
	MemberID        uint64          `gorm:"column:member_id"`
	MatchStatus     MatchStatus     `gorm:"column:match_status"`
	TransactionType TransactionType `gorm:"column:transaction_type"`
	ExchangeCode    string          `gorm:"column:exchange_code"`
	ProductCode     string          `gorm:"column:product_code"`
	TradeType       TradeType       `gorm:"column:trade_type"`
	Amount          decimal.Decimal `gorm:"column:amount"`
	CreatedAt       time.Time       `gorm:"column:created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at"`
}

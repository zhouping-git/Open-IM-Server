package relation

import (
	"context"
	"github.com/shopspring/decimal"
	"time"
)

type PointsWater struct {
	PointsWaterId   string          `gorm:"column:points_water_id;primary_key;type:varchar(64)" json:"pointsWaterId" binding:"required"`
	UserId          string          `gorm:"column:user_id;not null;type:varchar(64);index:index_user_points" json:"userId" binding:"required"`
	Points          decimal.Decimal `gorm:"column:points;not null;type:decimal(12,2)" json:"points" binding:"required"`
	Money           decimal.Decimal `gorm:"column:money;type:decimal(12,2)" json:"money"`                    // 金额，充值或提现时缓存
	PointsWaterType int8            `gorm:"column:points_water_type;not null" json:"pointsWaterType"`        // 流水类型：1、充值；2、提现；3、收红包；4、发红包；5、红包退回
	Source          int8            `gorm:"column:source" json:"source"`                                     // 金额来源，充值时存储类型：1、微信；2、支付宝；3、银行卡
	Target          int8            `gorm:"column:target" json:"target"`                                     // 金额去向，提现时存储类型：1、微信零钱；2、支付宝零钱；3、银行
	RelationAccount string          `gorm:"column:relation_account;type:varchar(64)" json:"relationAccount"` // 缓存关联的账户，如提现到微信则缓存微信账号
	RedPacketId     string          `gorm:"column:red_packet_id;type:varchar(64)" json:"redPacketId"`        // 发红包、收红包、红包退回时缓存红包标识。
	RedPacketType   int8            `gorm:"column:red_packet_type;type:varchar(20)" json:"redPacketType"`    // 缓存红包类型，同red_packet表
	BackCause       int8            `gorm:"column:back_cause" json:"backCause"`                              // 红包退回原因，points_water_type为5时记录。1、超时退回
	PointsWaterTime time.Time       `gorm:"column:points_water_time;not null" json:"pointsWaterTime"`
}

func (PointsWater) TableName() string {
	return "points_water"
}

type PointsWaterInterface interface {
	NewTx(tx any) PointsWaterInterface
	Add(ctx context.Context, pointsWater *PointsWater) error
	BatchAdd(ctx context.Context, waterList []*PointsWater) error
	Delete(ctx context.Context, redPacketId string) error
	TakeUserId(ctx context.Context, userId string, waterType int8, page int32, size int32) (uint32, []*PointsWater, error)
}

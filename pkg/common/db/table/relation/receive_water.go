package relation

import (
	"context"
	"github.com/shopspring/decimal"
	"time"
)

type ReceiveWater struct {
	ReceiveWaterId string          `gorm:"column:receive_water_id;primary_key;type:varchar(64)" json:"receiveWaterId" binding:"required"`
	RedPacketId    string          `gorm:"column:red_packet_id;not null;type:varchar(64);index:index_red_packet_id" json:"redPacketId" binding:"required"`
	ReceiveUserId  string          `gorm:"column:receive_user_id;not null;type:varchar(64)" json:"receiveUserId" binding:"required"`
	Points         decimal.Decimal `gorm:"column:points;not null;type:decimal(12,2)" json:"points" binding:"required"`
	CreateTime     time.Time       `gorm:"column:create_time" json:"createTime"`
	Name           string          `json:"nickName"`
	FaceUrl        string          `json:"faceUrl"`
}

func (ReceiveWater) TableName() string {
	return "receive_water"
}

type ReceiveWaterInterface interface {
	NewTx(tx any) ReceiveWaterInterface
	AddWater(ctx context.Context, receiveWater *ReceiveWater) error
	TakeRedPacketWater(ctx context.Context, redPacketId string) ([]*ReceiveWater, error)
}

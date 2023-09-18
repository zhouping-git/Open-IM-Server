package relation

import (
	"context"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/customtype"
	"github.com/shopspring/decimal"
	"time"
)

type RedPacket struct {
	RedPacketId    string               `gorm:"column:red_packet_id;primary_key;type:varchar(64)" json:"redPacketId" binding:"required" mapstructure:"redPacketId"`
	RedPacketType  int8                 `gorm:"column:red_packet_type;not null" json:"redPacketType" binding:"required" mapstructure:",omitempty"`             // 红包类型：1、群红包；2、群专属红包；3、好友红包
	RedPackerState int8                 `gorm:"column:red_packet_state;not null;default:1" json:"redPackerState" binding:"required" mapstructure:",omitempty"` // 红包状态：1、可使用；2、已完成；3、已过期
	GroupId        string               `gorm:"column:group_id;type:varchar(64);index:index_red_packet_group" json:"groupId" mapstructure:",omitempty"`
	SendUserId     string               `gorm:"column:send_user_id;type:varchar(64);not null;index:index_red_packet_send" json:"sendUserId" binding:"required" mapstructure:",omitempty"`
	ReceiveUserId  string               `gorm:"column:receive_user_id;type:varchar(64);index:index_red_packet_receive" json:"receiveUserId" mapstructure:",omitempty"`
	Points         decimal.Decimal      `gorm:"column:points;not null;type:decimal(12,2)" json:"points" binding:"required" mapstructure:"points"`
	RemainPoints   decimal.Decimal      `gorm:"column:remain_points;type:decimal(12,2)" json:"remainPoints" mapstructure:"remainPoints"`
	Count          int32                `gorm:"column:count" json:"count" mapstructure:"count"`
	RemainCount    int32                `gorm:"column:remain_count" json:"remainCount" mapstructure:"remainCount"`
	LastDigits     customtype.Int32Arr  `gorm:"column:last_digits;type:varchar(200)" json:"lastDigits" mapstructure:"lastDigits"`
	FixedIndex     customtype.Int32Arr  `gorm:"column:fixed_index;type:varchar(200)" json:"fixedIndex" mapstructure:"fixedIndex"`
	WhiteList      customtype.StringArr `gorm:"column:white_list;type:varchar(200)" json:"whiteList" mapstructure:"whiteList"`
	CreateTime     time.Time            `gorm:"column:create_time" json:"createTime" mapstructure:",omitempty"`
	UpdateTime     *time.Time           `gorm:"column:update_time" json:"updateTime" mapstructure:",omitempty"`
}

func (RedPacket) TableName() string {
	return "red_packet"
}

type RedPacketInterface interface {
	NewTx(tx any) RedPacketInterface
	AddRedPacket(ctx context.Context, groupRedPacket *RedPacket) error
	BatchAdd(ctx context.Context, redPackets []*RedPacket) error
	UpdateRemainPointsAndCount(ctx context.Context, redPacketId string, points decimal.Decimal, isComplete bool, args ...[]int32) error
	IsComplete(ctx context.Context, redPacketId string) (bool, error)
	TakeRedPacket(ctx context.Context, redPacketId string) (*RedPacket, error)
	UpdateRedPacketState(ctx context.Context, redPacketId string, redPackerState int8) error
	Delete(ctx context.Context, redPacketId string) error
}

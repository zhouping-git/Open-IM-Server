package relation

import (
	"context"
	"github.com/shopspring/decimal"
	"time"
)

type UserPoints struct {
	UserId      string          `gorm:"column:user_id;primary_key;type:varchar(64)" json:"userId" binding:"required"`
	PointsTotal decimal.Decimal `gorm:"column:points_total;not null;type:decimal(12,2)" json:"pointsTotal" binding:"required"`
	CreateTime  time.Time       `gorm:"column:create_time" json:"createTime"`
	UpdateTime  *time.Time      `gorm:"column:update_time" json:"updateTime"` // 日期设置为指针时值可以为null
}

func (UserPoints) TableName() string {
	return "user_points"
}

type UserPointsInterface interface {
	NewTx(tx any) UserPointsInterface
	Add(ctx context.Context, userPoints *UserPoints) error
	Update(ctx context.Context, userId string, points decimal.Decimal) error
	AddOrUpdate(ctx context.Context, userPoints *UserPoints) error
	TakeUserId(ctx context.Context, userId string) (*UserPoints, error)
}

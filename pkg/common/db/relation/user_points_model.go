package relation

import (
	"context"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/table/relation"
	"github.com/OpenIMSDK/tools/errs"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

func NewUserPoints(db *gorm.DB) relation.UserPointsInterface {
	return &UserPoints{NewMetaDB(db, &relation.UserPoints{})}
}

type UserPoints struct {
	*MetaDB
}

func (o UserPoints) NewTx(tx any) relation.UserPointsInterface {
	return &UserPoints{NewMetaDB(tx.(*gorm.DB), &relation.UserPoints{})}
}

func (o UserPoints) Add(ctx context.Context, userPoints *relation.UserPoints) error {
	return errs.Wrap(o.DB.WithContext(ctx).Create(userPoints).Error)
}

func (o UserPoints) Update(ctx context.Context, userId string, addPoints decimal.Decimal) error {
	return errs.Wrap(
		o.DB.WithContext(ctx).Model(&relation.UserPoints{}).Where("user_id = ?", userId).UpdateColumns(map[string]interface{}{
			"points_total": gorm.Expr("points_total + ?", addPoints),
			"update_time":  time.Now(),
		}).Error,
	)
}

func (o UserPoints) AddOrUpdate(ctx context.Context, userPoints *relation.UserPoints) error {
	return errs.Wrap(o.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"points_total": gorm.Expr("points_total + VALUES(points_total)"),
			"update_time":  time.Now(),
		}),
		//DoUpdates: clause.AssignmentColumns([]string{"points_total", "update_time"}),
	}).Create(userPoints).Error)
}

func (o UserPoints) TakeUserId(ctx context.Context, userId string) (*relation.UserPoints, error) {
	var u relation.UserPoints
	return &u, errs.Wrap(o.DB.WithContext(ctx).Where("user_id = ?", userId).Take(&u).Error)
}

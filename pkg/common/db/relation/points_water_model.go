package relation

import (
	"context"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/table/relation"
	"github.com/OpenIMSDK/tools/errs"
	"gorm.io/gorm"
)

func NewPointsWater(db *gorm.DB) relation.PointsWaterInterface {
	return &PointsWater{NewMetaDB(db, &relation.PointsWater{})}
}

type PointsWater struct {
	*MetaDB
}

func (o PointsWater) NewTx(tx any) relation.PointsWaterInterface {
	return &PointsWater{NewMetaDB(tx.(*gorm.DB), &relation.PointsWater{})}
}

func (o PointsWater) Add(ctx context.Context, pointsWater *relation.PointsWater) error {
	return errs.Wrap(o.DB.WithContext(ctx).Create(pointsWater).Error)
}

func (o PointsWater) BatchAdd(ctx context.Context, waterList []*relation.PointsWater) error {
	return errs.Wrap(o.DB.WithContext(ctx).CreateInBatches(waterList, len(waterList)).Error)
}

func (o PointsWater) Delete(ctx context.Context, redPacketId string) error {
	return errs.Wrap(o.DB.WithContext(ctx).Where("red_packet_id = ?", redPacketId).Delete(&relation.PointsWater{}).Error)
}

func (o PointsWater) TakeUserId(ctx context.Context, userId string, waterType int8, page int32, size int32) (uint32, []*relation.PointsWater, error) {
	var count int64
	var model relation.PointsWater
	countTx := o.DB.Model(&model).Where("user_id = ?", userId)
	if waterType > 0 {
		countTx = countTx.Where("points_water_type = ?", waterType)
	}
	if err := countTx.Count(&count).Error; err != nil {
		return 0, nil, errs.Wrap(err)
	}

	var p []*relation.PointsWater
	tx := o.DB.WithContext(ctx).Limit(int(size)).Offset(int((page-1)*size)).Where("user_id = ?", userId)
	if waterType > 0 {
		tx = tx.Where("points_water_type = ?", waterType)
	}
	tx = tx.Order("points_water_time desc")
	if err := tx.Find(&p).Error; err != nil {
		return 0, nil, errs.Wrap(err)
	}
	//if err := o.DB.WithContext(ctx).Limit(int(page)).Offset(int((size-1)*page)).Where("user_id = ?", userId).Where("points_water_type = ?", type).Find(&p).Error; err != nil {
	//	return 0, nil, errs.Wrap(err)
	//}
	return uint32(count), p, nil
}

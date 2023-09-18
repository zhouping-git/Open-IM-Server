package relation

import (
	"context"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/table/relation"
	"github.com/OpenIMSDK/tools/errs"
	"gorm.io/gorm"
)

func NewReceiveWater(db *gorm.DB) relation.ReceiveWaterInterface {
	return &ReceiveWater{NewMetaDB(db, &relation.ReceiveWater{})}
}

type ReceiveWater struct {
	*MetaDB
}

func (o ReceiveWater) NewTx(tx any) relation.ReceiveWaterInterface {
	return &ReceiveWater{NewMetaDB(tx.(*gorm.DB), &relation.ReceiveWater{})}
}

func (o ReceiveWater) TakeRedPacketWater(ctx context.Context, redPacketId string) ([]*relation.ReceiveWater, error) {
	var model []*relation.ReceiveWater
	tx := o.DB.WithContext(ctx).Where("red_packet_id = ?", redPacketId)
	tx = tx.Joins("left join user on user.user_id=receive_water.receive_user_id")
	tx = tx.Select("receive_water.*, user.name, user.face_url") // 返回用户表指定字段

	if err := tx.Find(&model).Error; err != nil {
		return nil, errs.Wrap(err)
	}
	return model, nil
}

func (o ReceiveWater) AddWater(ctx context.Context, receiveWater *relation.ReceiveWater) error {
	return errs.Wrap(o.DB.WithContext(ctx).Create(receiveWater).Error)
}

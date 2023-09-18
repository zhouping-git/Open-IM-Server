package relation

import (
	"context"
	"errors"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/table/relation"
	"github.com/OpenIMSDK/tools/errs"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"time"
)

func NewRedPacket(db *gorm.DB) relation.RedPacketInterface {
	return &RedPacket{NewMetaDB(db, &relation.RedPacket{})}
}

type RedPacket struct {
	*MetaDB
}

func (o RedPacket) NewTx(tx any) relation.RedPacketInterface {
	return &RedPacket{NewMetaDB(tx.(*gorm.DB), &relation.RedPacket{})}
}

func (o RedPacket) Delete(ctx context.Context, redPacketId string) error {
	return errs.Wrap(o.DB.WithContext(ctx).Where("red_packet_id = ?", redPacketId).Delete(&relation.RedPacket{}).Error)
}

func (o RedPacket) UpdateRedPacketState(ctx context.Context, redPacketId string, redPackerState int8) error {
	return errs.Wrap(
		o.DB.WithContext(ctx).Model(&relation.RedPacket{}).Where("red_packet_id = ?", redPacketId).UpdateColumns(map[string]interface{}{
			"red_packet_state": redPackerState,
			"update_time":      time.Now(),
		}).Error,
	)
}

func (o RedPacket) TakeRedPacket(ctx context.Context, redPacketId string) (*relation.RedPacket, error) {
	var model relation.RedPacket
	err := errs.Wrap(o.DB.WithContext(ctx).Where("red_packet_id = ?", redPacketId).Take(&model).Error)
	if err != nil {
		return nil, err
	}
	return &model, nil
}

func (o RedPacket) IsComplete(ctx context.Context, redPacketId string) (bool, error) {
	var model relation.RedPacket
	err := errs.Wrap(o.DB.WithContext(ctx).Where("red_packet_id = ?", redPacketId).Where("red_packet_state in (2,3)").Take(&model).Error)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (o RedPacket) AddRedPacket(ctx context.Context, groupRedPacket *relation.RedPacket) error {
	return errs.Wrap(o.DB.WithContext(ctx).Create(groupRedPacket).Error)
}

func (o RedPacket) BatchAdd(ctx context.Context, redPackets []*relation.RedPacket) error {
	return errs.Wrap(o.DB.WithContext(ctx).CreateInBatches(redPackets, len(redPackets)).Error)
}

func (o RedPacket) UpdateRemainPointsAndCount(ctx context.Context, redPacketId string, uPoints decimal.Decimal, isComplete bool, args ...[]int32) error {
	uCol := map[string]interface{}{
		"remain_points": gorm.Expr("remain_points - ?", uPoints),
		"remain_count":  gorm.Expr("remain_count - 1"),
		"update_time":   time.Now(),
	}
	if isComplete {
		uCol["red_packet_state"] = 3
	}
	if len(args) > 0 {
		uCol["FixedIndex"] = args[0]
	}
	return errs.Wrap(o.DB.WithContext(ctx).Model(&relation.RedPacket{}).Where("red_packet_id = ?", redPacketId).UpdateColumns(uCol).Error)
}

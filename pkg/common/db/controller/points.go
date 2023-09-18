package controller

import (
	"context"
	relationtb "github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/table/relation"
	"github.com/OpenIMSDK/tools/tx"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"time"
)

type PointsDatabaseInterface interface {
	UserPointsRecharge(ctx context.Context, pointsWater *relationtb.PointsWater) error
	UserPointsWithdraw(ctx context.Context, pointsWater *relationtb.PointsWater) error
	SendRedPacket(ctx context.Context, groupRedPacket *relationtb.RedPacket) error
	BatchSendRedPacket(ctx context.Context, sendUserId string, sumPoints decimal.Decimal, redPackets []*relationtb.RedPacket) error
	ResetRedPacket(ctx context.Context, redPacketId string) error
	GrabRedPacket(ctx context.Context, receiveWater *relationtb.ReceiveWater, isComplete bool, args ...[]int32) error
	ReceiveC2CRedPacket(ctx context.Context, pointsWater *relationtb.PointsWater) error
	GetGroupUsers(ctx context.Context, groupId string) ([]*relationtb.WhiteList, error)
	GetUserPoints(ctx context.Context, userId string) (*relationtb.UserPoints, error)
	PointsWaterForType(ctx context.Context, userId string, waterType int8, page int32, size int32) (uint32, []*relationtb.PointsWater, error)
	GetRedPacket(ctx context.Context, redPacketId string) (*relationtb.RedPacket, error)
	RedPacketOverTime(ctx context.Context, redPacketId string) error
	RedPacketWater(ctx context.Context, redPacketId string) ([]*relationtb.ReceiveWater, error)
}

func NewPointsDatabase(
	userPoints relationtb.UserPointsInterface,
	pointsWater relationtb.PointsWaterInterface,
	whiteList relationtb.WhiteListInterface,
	redPacket relationtb.RedPacketInterface,
	receiveWater relationtb.ReceiveWaterInterface,
	tx tx.Tx,
) PointsDatabaseInterface {
	return &pointsDatabase{
		userPoints:   userPoints,
		pointsWater:  pointsWater,
		whiteList:    whiteList,
		redPacket:    redPacket,
		receiveWater: receiveWater,
		tx:           tx,
	}
}

type pointsDatabase struct {
	tx           tx.Tx
	redPacket    relationtb.RedPacketInterface
	receiveWater relationtb.ReceiveWaterInterface
	userPoints   relationtb.UserPointsInterface
	pointsWater  relationtb.PointsWaterInterface
	whiteList    relationtb.WhiteListInterface
}

func (o *pointsDatabase) ResetRedPacket(ctx context.Context, redPacketId string) error {
	packet, err := o.redPacket.TakeRedPacket(ctx, redPacketId)
	if err != nil {
		return err
	}
	return o.tx.Transaction(func(tx any) error {
		// 已扣除金额返还
		if err := o.userPoints.Update(ctx, packet.SendUserId, packet.Points); err != nil {
			return err
		}
		// 删除积分记录
		if err := o.pointsWater.Delete(ctx, redPacketId); err != nil {
			return err
		}

		// todo 删除红包记录。理论上应该不存在，暂不处理

		if err := o.redPacket.Delete(ctx, redPacketId); err != nil {
			return err
		}
		return nil
	})
}

func (o *pointsDatabase) RedPacketWater(ctx context.Context, redPacketId string) ([]*relationtb.ReceiveWater, error) {
	return o.receiveWater.TakeRedPacketWater(ctx, redPacketId)
}

func (o *pointsDatabase) RedPacketOverTime(ctx context.Context, redPacketId string) error {
	data, err := o.redPacket.TakeRedPacket(ctx, redPacketId)
	if err != nil {
		return err
	}
	// 超期时红包还没有抢完，这执行超期处理
	if data.RedPackerState == 1 {
		return o.tx.Transaction(func(tx any) error {
			err := o.userPoints.NewTx(tx).Update(ctx, data.SendUserId, data.RemainPoints)
			if err != nil {
				return err
			}

			water := &relationtb.PointsWater{
				PointsWaterId:   uuid.New().String(),
				UserId:          data.SendUserId,
				Points:          data.RemainPoints,
				PointsWaterType: 5,
				RedPacketId:     data.RedPacketId,
				RedPacketType:   data.RedPacketType,
				BackCause:       1,
				PointsWaterTime: time.Now(),
			}
			if err := o.pointsWater.NewTx(tx).Add(ctx, water); err != nil {
				return err
			}

			if err := o.redPacket.NewTx(tx).UpdateRedPacketState(ctx, data.RedPacketId, 3); err != nil {
				return err
			}

			return nil
		})
	}
	return nil
}

func (o *pointsDatabase) GetRedPacket(ctx context.Context, redPacketId string) (*relationtb.RedPacket, error) {
	return o.redPacket.TakeRedPacket(ctx, redPacketId)
}

func (o *pointsDatabase) PointsWaterForType(ctx context.Context, userId string, waterType int8, page int32, size int32) (uint32, []*relationtb.PointsWater, error) {
	return o.pointsWater.TakeUserId(ctx, userId, waterType, page, size)
}

func (o *pointsDatabase) GetUserPoints(ctx context.Context, userId string) (*relationtb.UserPoints, error) {
	return o.userPoints.TakeUserId(ctx, userId)
}

func (o *pointsDatabase) GetGroupUsers(ctx context.Context, groupId string) ([]*relationtb.WhiteList, error) {
	return o.whiteList.GetGroupUsers(ctx, groupId)
}

func (o *pointsDatabase) UserPointsRecharge(ctx context.Context, pointsWater *relationtb.PointsWater) error {
	return o.tx.Transaction(func(tx any) error {
		if err := o.pointsWater.NewTx(tx).Add(ctx, pointsWater); err != nil {
			return err
		}

		userPoints := &relationtb.UserPoints{
			UserId:      pointsWater.UserId,
			PointsTotal: pointsWater.Points,
			CreateTime:  time.Now(),
		}
		if err := o.userPoints.NewTx(tx).AddOrUpdate(ctx, userPoints); err != nil {
			return err
		}
		return nil
	})
}

func (o *pointsDatabase) UserPointsWithdraw(ctx context.Context, pointsWater *relationtb.PointsWater) error {
	return o.tx.Transaction(func(tx any) error {
		if err := o.pointsWater.NewTx(tx).Add(ctx, pointsWater); err != nil {
			return err
		}

		userPoints := &relationtb.UserPoints{
			UserId:      pointsWater.UserId,
			PointsTotal: pointsWater.Points.Mul(decimal.NewFromInt32(-1)),
			CreateTime:  time.Now(),
		}
		if err := o.userPoints.NewTx(tx).AddOrUpdate(ctx, userPoints); err != nil {
			return err
		}
		return nil
	})
}

func (o *pointsDatabase) SendRedPacket(ctx context.Context, redPacket *relationtb.RedPacket) error {
	return o.tx.Transaction(func(tx any) error {
		if err := o.redPacket.NewTx(tx).AddRedPacket(ctx, redPacket); err != nil {
			return err
		}

		entity := &relationtb.PointsWater{
			PointsWaterId:   uuid.New().String(),
			UserId:          redPacket.SendUserId,
			Points:          redPacket.Points,
			PointsWaterType: 4,
			RedPacketId:     redPacket.RedPacketId,
			RedPacketType:   redPacket.RedPacketType,
			PointsWaterTime: time.Now(),
		}
		if err := o.pointsWater.NewTx(tx).Add(ctx, entity); err != nil {
			return err
		}

		if err := o.userPoints.NewTx(tx).Update(ctx, redPacket.SendUserId, redPacket.Points.Mul(decimal.NewFromInt32(-1))); err != nil {
			return err
		}
		return nil
	})
}

func (o *pointsDatabase) BatchSendRedPacket(ctx context.Context, sendUserId string, sumPoints decimal.Decimal, redPackets []*relationtb.RedPacket) error {
	return o.tx.Transaction(func(tx any) error {
		if err := o.redPacket.NewTx(tx).BatchAdd(ctx, redPackets); err != nil {
			return err
		}

		waterList := make([]*relationtb.PointsWater, len(redPackets))
		for _, redPacket := range redPackets {
			water := &relationtb.PointsWater{
				PointsWaterId:   uuid.New().String(),
				UserId:          redPacket.SendUserId,
				Points:          redPacket.Points,
				PointsWaterType: 4,
				RedPacketId:     redPacket.RedPacketId,
				RedPacketType:   redPacket.RedPacketType,
				PointsWaterTime: time.Now(),
			}
			waterList = append(waterList, water)
		}
		if err := o.pointsWater.NewTx(tx).BatchAdd(ctx, waterList); err != nil {
			return err
		}

		if err := o.userPoints.NewTx(tx).Update(ctx, sendUserId, sumPoints.Mul(decimal.NewFromInt32(-1))); err != nil {
			return err
		}
		return nil
	})
}

func (o *pointsDatabase) GrabRedPacket(ctx context.Context, receiveWater *relationtb.ReceiveWater, isComplete bool, arg ...[]int32) error {
	return o.tx.Transaction(func(tx any) error {
		if err := o.receiveWater.NewTx(tx).AddWater(ctx, receiveWater); err != nil {
			return err
		}
		if err := o.redPacket.NewTx(tx).UpdateRemainPointsAndCount(ctx, receiveWater.RedPacketId, receiveWater.Points, isComplete, arg...); err != nil {
			return err
		}

		userPoints := &relationtb.UserPoints{
			UserId:      receiveWater.ReceiveUserId,
			PointsTotal: receiveWater.Points,
			CreateTime:  time.Now(),
		}
		if err := o.userPoints.NewTx(tx).AddOrUpdate(ctx, userPoints); err != nil {
			return err
		}

		pointsWater := &relationtb.PointsWater{
			PointsWaterId:   uuid.New().String(),
			UserId:          receiveWater.ReceiveUserId,
			Points:          receiveWater.Points,
			PointsWaterType: 3,
			RedPacketId:     receiveWater.RedPacketId,
			RedPacketType:   1,
			PointsWaterTime: time.Now(),
		}
		if err := o.pointsWater.NewTx(tx).Add(ctx, pointsWater); err != nil {
			return err
		}
		return nil
	})
}

func (o *pointsDatabase) ReceiveC2CRedPacket(ctx context.Context, pointsWater *relationtb.PointsWater) error {
	return o.tx.Transaction(func(tx any) error {
		if err := o.redPacket.NewTx(tx).UpdateRedPacketState(ctx, pointsWater.RedPacketId, 2); err != nil {
			return err
		}

		userPoints := &relationtb.UserPoints{
			UserId:      pointsWater.UserId,
			PointsTotal: pointsWater.Points,
			CreateTime:  time.Now(),
		}
		if err := o.userPoints.NewTx(tx).AddOrUpdate(ctx, userPoints); err != nil {
			return err
		}

		if err := o.pointsWater.NewTx(tx).Add(ctx, pointsWater); err != nil {
			return err
		}
		return nil
	})
}

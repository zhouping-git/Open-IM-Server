package batchmsg

import (
	"context"
	"encoding/json"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/apistruct"
	localconstant "github.com/OpenIMSDK/Open-IM-Server/pkg/common/constant"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/controller"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/delayqueue"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/grabredpacket"
	relationtb "github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/table/relation"
	pointspb "github.com/OpenIMSDK/Open-IM-Server/pkg/proto/points"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/rpcclient"
	"github.com/OpenIMSDK/protocol/constant"
	"github.com/OpenIMSDK/tools/log"
	"github.com/OpenIMSDK/tools/utils"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"time"
)

func NewBatchRedPacketSender(
	pointsDb controller.PointsDatabaseInterface,
	msgRpcClient *rpcclient.MessageRpcClient,
	userRpcClient *rpcclient.UserRpcClient,
	delayQueue *delayqueue.RedPacketDelayQueue,
) *BatchRedPacketSender {
	return &BatchRedPacketSender{
		BatchMsgSender: rpcclient.NewBatchMsgSender(rpcclient.WithMsgRpcClient[apistruct.CustomElem](msgRpcClient), rpcclient.WithMsgUserRpcClient[apistruct.CustomElem](userRpcClient)),
		pointsDb:       pointsDb,
		delayQueue:     delayQueue,
	}
}

type BatchRedPacketSender struct {
	*rpcclient.BatchMsgSender[apistruct.CustomElem]
	pointsDb   controller.PointsDatabaseInterface
	delayQueue *delayqueue.RedPacketDelayQueue
}

func (p *BatchRedPacketSender) OnlyRedPacket(ctx context.Context, redPacket *relationtb.RedPacket, title string) (err error) {
	points, _ := redPacket.Points.Float64()

	switch redPacket.RedPacketType {
	case 1:
		msgData := apistruct.CustomContextElem{
			CustomType: localconstant.RedPacketMsg,
			Data: apistruct.RedPacketElem{
				RedPacketId:    redPacket.RedPacketId,
				RedPacketType:  int32(redPacket.RedPacketType),
				RedPacketState: int32(redPacket.RedPackerState),
				GroupId:        redPacket.GroupId,
				SendUserId:     redPacket.SendUserId,
				Points:         float32(points),
				Count:          redPacket.Count,
				Title:          title,
				LastDigits:     redPacket.LastDigits,
			},
		}
		dataStr, _ := json.Marshal(msgData)
		elem := apistruct.CustomElem{
			Data:        string(dataStr),
			Description: "",
			Extension:   "",
		}
		err = p.BatchMsgSender.SendOnlyMsg(ctx, redPacket.SendUserId, redPacket.GroupId, constant.Custom, constant.SuperGroupChatType, elem, rpcclient.WithOperateStatus(1))
	case 2:
		msgData := apistruct.CustomContextElem{
			CustomType: localconstant.RedPacketMsg,
			Data: apistruct.RedPacketElem{
				RedPacketId:    redPacket.RedPacketId,
				RedPacketType:  int32(redPacket.RedPacketType),
				RedPacketState: int32(redPacket.RedPackerState),
				GroupId:        redPacket.GroupId,
				SendUserId:     redPacket.SendUserId,
				ReceiveUserId:  redPacket.ReceiveUserId,
				Points:         float32(points),
				Title:          title,
			},
		}
		dataStr, _ := json.Marshal(msgData)
		elem := apistruct.CustomElem{
			Data:        string(dataStr),
			Description: "",
			Extension:   "",
		}
		err = p.BatchMsgSender.SendOnlyMsg(
			ctx,
			redPacket.SendUserId,
			redPacket.GroupId,
			constant.Custom,
			constant.SuperGroupChatType,
			elem,
			rpcclient.WithSpecifyRecipient([]string{redPacket.SendUserId, redPacket.ReceiveUserId}),
			rpcclient.WithOperateStatus(1),
		)
	case 3:
		msgData := apistruct.CustomContextElem{
			CustomType: localconstant.RedPacketMsg,
			Data: apistruct.RedPacketElem{
				RedPacketId:    redPacket.RedPacketId,
				RedPacketType:  int32(redPacket.RedPacketType),
				RedPacketState: int32(redPacket.RedPackerState),
				SendUserId:     redPacket.SendUserId,
				ReceiveUserId:  redPacket.ReceiveUserId,
				Points:         float32(points),
				Title:          title,
			},
		}
		dataStr, _ := json.Marshal(msgData)
		elem := apistruct.CustomElem{
			Data:        string(dataStr),
			Description: "",
			Extension:   "",
		}
		err = p.BatchMsgSender.SendOnlyMsg(ctx, redPacket.SendUserId, redPacket.ReceiveUserId, constant.Custom, constant.SingleChatType, elem, rpcclient.WithOperateStatus(1))
	}
	return err
}

func (p *BatchRedPacketSender) BatchRedPacket(ctx context.Context, req *pointspb.BatchRedPacketReq) (err error) {
	defer log.ZDebug(ctx, "return")
	defer func() {
		if err != nil {
			log.ZError(ctx, utils.GetFuncName(1)+" failed", err)
		}
	}()

	redPacket := &relationtb.RedPacket{
		RedPacketType:  1,
		RedPackerState: 1,
		GroupId:        req.GroupId,
		SendUserId:     req.SendUserId,
		Points:         decimal.NewFromFloat32(req.Points),
		RemainPoints:   decimal.NewFromFloat32(req.Points),
		Count:          req.Count,
		RemainCount:    req.Count,
		LastDigits:     req.LastDigits,
		CreateTime:     time.Now(),
	}

	// 构建固定尾数规则
	dLen := len(req.LastDigits)
	if dLen != 0 {
		// 获取白名单用户
		whiteList, err := p.pointsDb.GetGroupUsers(ctx, redPacket.GroupId)
		if err != nil {
			var wls []string
			for _, item := range whiteList {
				wls = append(wls, item.UserId)
			}
			redPacket.WhiteList = wls
		}
	}

	redPackets := make([]*relationtb.RedPacket, req.RedPacketCount)
	ids := make([]string, req.RedPacketCount)
	elems := make([]apistruct.CustomElem, req.RedPacketCount)
	for i := 0; i < int(req.RedPacketCount); i++ {
		// 逐一生成红包
		redPacketId := uuid.New().String()
		redPacket.RedPacketId = redPacketId
		// 放入循环体保证每个红包命中位数差异化
		if dLen != 0 {
			rs := grabredpacket.BuildReservoirSampling(int64(redPacket.Count))
			redPacket.FixedIndex = rs.Sampling(dLen)
		}

		redPackets = append(redPackets, redPacket)
		ids = append(ids, redPacketId)

		msgData := apistruct.CustomContextElem{
			CustomType: localconstant.RedPacketMsg,
			Data: apistruct.RedPacketElem{
				RedPacketId:    redPacketId,
				RedPacketType:  int32(redPacket.RedPacketType),
				RedPacketState: int32(redPacket.RedPackerState),
				GroupId:        redPacket.GroupId,
				SendUserId:     redPacket.SendUserId,
				Points:         req.Points,
				Count:          req.Count,
				Title:          req.Title,
				LastDigits:     req.LastDigits,
			},
		}
		dataStr, _ := json.Marshal(msgData)
		elem := apistruct.CustomElem{
			Data:        string(dataStr),
			Description: "",
			Extension:   "",
		}
		elems = append(elems, elem)
	}
	err = p.pointsDb.BatchSendRedPacket(ctx, req.SendUserId, decimal.NewFromFloat32(req.SumPoints), redPackets)
	if err == nil {
		// 将红包信息放入秒杀队列
		if _, err := grabredpacket.CacheRedPacketInfos(ctx, redPackets); err == nil {
			// 将红包放入延迟队列
			p.delayQueue.SendMsgs(ids)

			// 批量发送消息
			err = p.BatchMsgSender.SendBatchMsg(ctx, req.SendUserId, req.GroupId, constant.Custom, constant.SuperGroupChatType, elems, rpcclient.WithOperateStatus(1))
		}
	}

	return err
}

package points

import (
	"context"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/table/relation"
	pointspb "github.com/OpenIMSDK/Open-IM-Server/pkg/proto/points"
	"github.com/OpenIMSDK/protocol/msg"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"log"
	"time"
)

type SecKillMessage struct {
	Req           *pointspb.GrabRedPacketReq
	ReceivePoints decimal.Decimal
	IsComplete    bool
	NewFixedIndex []int32
	Server        *pointsServer
}

const maxMessageNum = 20000

var SecKillChannel = make(chan SecKillMessage, maxMessageNum)

func secKillConsumer() {
	for {
		message := <-SecKillChannel
		log.Println("Got one message: " + message.Req.RedPacketId)

		redPacketId := message.Req.RedPacketId
		userId := message.Req.ReceiveUserId
		receivePoints := message.ReceivePoints
		isComplete := message.IsComplete

		receiveWater := &relation.ReceiveWater{
			ReceiveWaterId: uuid.New().String(),
			RedPacketId:    redPacketId,
			ReceiveUserId:  userId,
			Points:         receivePoints,
			CreateTime:     time.Now(),
		}

		ctx := context.Background()
		var err error
		if message.NewFixedIndex == nil {
			err = message.Server.pointsDatabase.GrabRedPacket(ctx, receiveWater, isComplete)
		} else {
			err = message.Server.pointsDatabase.GrabRedPacket(ctx, receiveWater, isComplete, message.NewFixedIndex)
		}

		if err != nil {
			return
		}

		if isComplete {
			// 所有红包抢完则修改消息操作状态
			_, err = message.Server.msgClient.Client.SetMarkMsgOperateStatus(ctx, &msg.SetMarkMsgOperateStatusReq{
				ConversationID: message.Req.ConversationID,
				UserID:         message.Req.ReceiveUserId,
				Seq:            message.Req.Seq,
				State:          2,
				IsAddRead:      true,
			})
		} else {
			// 抢红包成功将用户加入红包已读用户组
			_, err = message.Server.msgClient.Client.SetMarkUserReadMsg(ctx, &msg.SetMarkUserReadMsgReq{
				ConversationID: message.Req.ConversationID,
				UserID:         message.Req.ReceiveUserId,
				Seq:            message.Req.Seq,
			})
		}

		if err != nil {
			return
		}
		message.Server.notification.GrabRedPacketNotification(ctx, message.Req)
	}
}

var isConsumerRun = false

func RunSecKillConsumer() {
	if !isConsumerRun {
		// 开启红包后置消费者队列，处理红包抢购后的DB处理业务
		go secKillConsumer()
		isConsumerRun = true
	}
}

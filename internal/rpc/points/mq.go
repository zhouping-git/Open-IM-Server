package points

import (
	"context"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/table/relation"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"log"
	"time"
)

type SecKillMessage struct {
	RedPacketId   string
	UserId        string
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
		log.Println("Got one message: " + message.RedPacketId)

		redPacketId := message.RedPacketId
		userId := message.UserId
		receivePoints := message.ReceivePoints
		isComplete := message.IsComplete

		receiveWater := &relation.ReceiveWater{
			ReceiveWaterId: uuid.New().String(),
			RedPacketId:    redPacketId,
			ReceiveUserId:  userId,
			Points:         receivePoints,
			CreateTime:     time.Now(),
		}

		var err error
		if message.NewFixedIndex == nil {
			err = message.Server.pointsDatabase.GrabRedPacket(context.Background(), receiveWater, isComplete)
		} else {
			err = message.Server.pointsDatabase.GrabRedPacket(context.Background(), receiveWater, isComplete, message.NewFixedIndex)
		}

		if err != nil {
			return
		}
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

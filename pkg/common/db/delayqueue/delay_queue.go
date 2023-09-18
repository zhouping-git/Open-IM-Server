package delayqueue

import (
	"context"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/config"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/grabredpacket"
	pointspb "github.com/OpenIMSDK/Open-IM-Server/pkg/proto/points"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/rpcclient"
	"github.com/OpenIMSDK/tools/discoveryregistry"
	"github.com/OpenIMSDK/tools/log"
	"github.com/OpenIMSDK/tools/mcontext"
	"github.com/hdt3213/delayqueue"
	"github.com/redis/go-redis/v9"
	"time"
)

const (
	QueueName = "redPacketDelayQueue"
	// SpacingInterval 设置延时时间
	//SpacingInterval = time.Hour * 24
	SpacingInterval = time.Minute * 10
)

var client redis.UniversalClient
var disconv discoveryregistry.SvcDiscoveryRegistry
var nctx context.Context

type RedPacketDelayQueue struct {
	queue *delayqueue.DelayQueue
}

func NewDelayQueue(r redis.UniversalClient, d discoveryregistry.SvcDiscoveryRegistry, ctx context.Context) (res *RedPacketDelayQueue) {
	client = r
	disconv = d
	nctx = mcontext.NewCtx("@@@" + mcontext.GetOperationID(ctx))

	var queue *delayqueue.DelayQueue
	if len(config.Config.Redis.Address) > 1 {
		queue = delayqueue.NewQueueOnCluster(QueueName, client.(*redis.ClusterClient), QueueCallBack)
	} else {
		queue = delayqueue.NewQueue(QueueName, client.(*redis.Client), QueueCallBack, delayqueue.UseHashTagKey())
	}
	// 设置并发数量
	queue.WithConcurrent(4)
	//queue.WithLogger(log.New(os.Stderr, "[DelayQueue]", log.LstdFlags))

	done := queue.StartConsume()
	go func() { //协程处理延时队列
		<-done
	}()
	//done := queue.StartConsume()
	//<-done 阻塞线程会导致panic
	return &RedPacketDelayQueue{
		queue: queue,
	}
}

func (q *RedPacketDelayQueue) SendMsg(payload string) {
	err := q.queue.SendDelayMsg(payload, SpacingInterval, delayqueue.WithRetryCount(3))
	if err != nil {
		log.ZError(context.Background(), "Delay queue send error", err)
		return
	}
}

func (q *RedPacketDelayQueue) SendMsgs(payloads []string) {
	for i := 0; i < len(payloads); i++ {
		err := q.queue.SendDelayMsg(payloads[i], SpacingInterval, delayqueue.WithRetryCount(3))
		if err != nil {
			log.ZError(context.Background(), "Delay queue send error", err)
			return
		}
	}
}

func (q *RedPacketDelayQueue) syncDbData() {

}

func QueueCallBack(payload string) bool {
	//ctx := context.Background()
	//val, err := grabredpacket.GetMapAll(ctx, payload)
	//if err == nil {
	//	return false
	//}
	//model := grabredpacket.RedPacketModel{}
	//err = mapstructure.WeakDecode(val, &model)
	//if err != nil {
	//	return false
	//}
	//// todo 判断红包是否被抢完
	//if model.RemainCount > 0 {
	//	//relation.RedPacket{}
	//}

	points := rpcclient.NewPointsRpcClient(disconv)
	req := &pointspb.RedPacketOverTimeReq{
		RedPacketId: payload,
	}
	_, err := points.Client.RedPacketOverTime(nctx, req)
	if err != nil {
		return false
	}

	uKey := grabredpacket.BuildGrabUserKey(payload)
	iKey := grabredpacket.BuildInfoKey(payload)
	client.Del(nctx, uKey, iKey)
	return true
}

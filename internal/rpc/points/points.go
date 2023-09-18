package points

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/controller"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/delayqueue"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/grabredpacket"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/relation"
	relationtb "github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/table/relation"
	pointspb "github.com/OpenIMSDK/Open-IM-Server/pkg/proto/points"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/rpcclient"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/rpcclient/batchmsg"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/rpcclient/notification"
	localutils "github.com/OpenIMSDK/Open-IM-Server/pkg/utils"
	"github.com/OpenIMSDK/protocol/sdkws"
	"github.com/OpenIMSDK/tools/discoveryregistry"
	"github.com/OpenIMSDK/tools/errs"
	"github.com/OpenIMSDK/tools/mcontext"
	"github.com/OpenIMSDK/tools/tx"
	"github.com/OpenIMSDK/tools/utils"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
	"time"
)

// var secKillSHA string

func Start(client discoveryregistry.SvcDiscoveryRegistry, server *grpc.Server) error {
	db, err := relation.NewGormDB()
	if err != nil {
		return err
	}
	tables := []any{
		&relationtb.UserPoints{},
		&relationtb.PointsWater{},
		&relationtb.RedPacket{},
		&relationtb.ReceiveWater{},
		&relationtb.WhiteList{},
	}
	// 进行数据库表比对，没有则创建表
	if err := db.AutoMigrate(tables...); err != nil {
		return err
	}

	rdb, err := grabredpacket.NewRedis()
	if err != nil {
		return err
	}

	pointsDatabase := controller.NewPointsDatabase(
		relation.NewUserPoints(db),
		relation.NewPointsWater(db),
		relation.NewWhiteList(db),
		relation.NewRedPacket(db),
		relation.NewReceiveWater(db),
		tx.NewGorm(db),
	)
	userRpcClient := rpcclient.NewUserRpcClient(client)
	msgRpcClient := rpcclient.NewMessageRpcClient(client)
	groupRpcClient := rpcclient.NewGroupRpcClient(client)
	pointsNotification := notification.NewPointsNotificationSender(
		pointsDatabase,
		&msgRpcClient,
		&userRpcClient,
		func(ctx context.Context, userIDs []string) ([]notification.CommonUser, error) {
			users, err := userRpcClient.GetUsersInfo(ctx, userIDs)
			if err != nil {
				return nil, err
			}
			return utils.Slice(users, func(e *sdkws.UserInfo) notification.CommonUser { return e }), nil
		},
		func(ctx context.Context, groupID string) (*sdkws.GroupInfo, error) {
			groupInfo, err := groupRpcClient.GetGroupInfo(ctx, groupID)
			if err != nil {
				return nil, err
			}
			return groupInfo, nil
		},
	)
	delayQueue := delayqueue.NewDelayQueue(rdb, client, context.Background())
	batchRedPacket := batchmsg.NewBatchRedPacketSender(
		pointsDatabase,
		delayQueue,
	)
	p := pointsServer{
		pointsDatabase: pointsDatabase,
		userClient:     userRpcClient,
		notification:   pointsNotification,
		batchMsg:       batchRedPacket,
		secKillSHA:     grabredpacket.PrepareScript(context.Background(), grabredpacket.SecKillScript), // 加载redis脚本
		delayQueue:     delayQueue,                                                                     // 启动红包超时延迟队列
	}
	pointspb.RegisterPointsServer(server, &p)

	// 加载redis脚本
	//secKillSHA = grabredpacket.PrepareScript(context.Background(), grabredpacket.GetScript())
	// 启动抢红包功能的消费者，异步更新数据库
	RunSecKillConsumer()
	// 启动红包超时延迟队列
	//delayqueue.NewDelayQueue(rdb)

	return nil
}

type pointsServer struct {
	*pointspb.UnimplementedPointsServer
	pointsDatabase controller.PointsDatabaseInterface
	userClient     rpcclient.UserRpcClient
	notification   *notification.PointsNotificationSender
	batchMsg       *batchmsg.BatchRedPacketSender
	delayQueue     *delayqueue.RedPacketDelayQueue
	secKillSHA     string
}

func (o *pointsServer) mustEmbedUnimplementedPointsServer() {}

func (o *pointsServer) UserPointsRecharge(ctx context.Context, req *pointspb.UserPointsRechargeReq) (*pointspb.UserPointsRechargeResp, error) {
	money := decimal.NewFromFloat32(req.Money)

	pointsWater := &relationtb.PointsWater{
		PointsWaterId:   uuid.New().String(),
		UserId:          req.UserId,
		Points:          money, // 后续如果存在充值送积分的情况可在此处定制逻辑
		Money:           money,
		PointsWaterType: 1,
		Source:          int8(req.Source),
		PointsWaterTime: time.Now(),
	}
	if err := o.pointsDatabase.UserPointsRecharge(ctx, pointsWater); err != nil {
		return nil, err
	}
	return &pointspb.UserPointsRechargeResp{
		Success: true,
	}, nil
}

func (o *pointsServer) UserPointsWithdraw(ctx context.Context, req *pointspb.UserPointsWithdrawReq) (*pointspb.UserPointsWithdrawResp, error) {
	points := decimal.NewFromFloat32(req.Points)
	pointsWater := &relationtb.PointsWater{
		PointsWaterId:   uuid.New().String(),
		UserId:          req.UserId,
		Points:          points,
		Money:           points, // 后续如果存在积分与金额计算的可在此处定制逻辑
		PointsWaterType: 2,
		Target:          int8(req.Target),
		RelationAccount: req.RelationAccount, // todo 此处缓存关联的收款账户信息，后续转接支付平台。当前值暂时不校验
		PointsWaterTime: time.Now(),
	}
	if err := o.pointsDatabase.UserPointsRecharge(ctx, pointsWater); err != nil {
		return nil, err
	}
	return &pointspb.UserPointsWithdrawResp{
		Success: true,
	}, nil
}

func (o *pointsServer) GetUserPoints(ctx context.Context, req *pointspb.UserPointsReq) (*pointspb.UserPointsResp, error) {
	data, err := o.pointsDatabase.GetUserPoints(ctx, req.UserId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { // 处理新用户没有初始充值的情况
			return &pointspb.UserPointsResp{
				Points: 0.00,
			}, nil
		}
		return nil, err
	}
	temp, ok := data.PointsTotal.Float64()
	if !ok {
		return nil, errs.ErrData.Wrap("user no points")
	}
	return &pointspb.UserPointsResp{
		Points: float32(temp),
	}, nil
}

func (o *pointsServer) PointsWaterForType(ctx context.Context, req *pointspb.PointsWaterForTypeReq) (*pointspb.PointsWaterForTypeResp, error) {
	if req.PointsWaterType == 0 {
		return nil, errs.ErrArgs.Wrap("param pointsWaterType is not null")
	}
	count, datas, err := o.pointsDatabase.PointsWaterForType(ctx, req.UserId, int8(req.PointsWaterType), req.Pagination.PageNumber, req.Pagination.ShowNumber)
	if err != nil {
		return nil, err
	}
	var records []*pointspb.PointsWater
	for _, item := range datas {
		tp, ok := item.Points.Float64()
		if !ok {
			return nil, errs.ErrData.Wrap("Data convert error")
		}

		addItem := &pointspb.PointsWater{
			PointsWaterId:   item.PointsWaterId,
			UserId:          item.UserId,
			Points:          float32(tp),
			PointsWaterType: int32(item.PointsWaterType),
			Source:          int32(item.Source),
			Target:          int32(item.Target),
			RelationAccount: item.RelationAccount,
			PointsWaterTime: item.PointsWaterTime.Format("2006-01-02 15:04:05"),
		}

		tm, ok := item.Money.Float64()
		if ok {
			addItem.Money = float32(tm)
		}
		records = append(records, addItem)
	}
	return &pointspb.PointsWaterForTypeResp{
		PointsWater: records,
		Count:       count,
	}, nil
}

func (o *pointsServer) GrabRedPacket(ctx context.Context, req *pointspb.GrabRedPacketReq) (*pointspb.GrabRedPacketResp, error) {
	// 计算红包积分
	model, tPoints, doReplace, flag := grabredpacket.ComputerRandPoints(ctx, req.RedPacketId, req.ReceiveUserId)
	if !flag {
		return nil, errors.New("redPacket computer error")
	}
	// 将对象转储为redis可存储对象
	var mData map[string]interface{}
	err := mapstructure.Decode(model, &mData)
	str, _ := json.Marshal(mData)
	code, err := grabredpacket.CacheAtomicSecKill(ctx, o.secKillSHA, req.RedPacketId, req.ReceiveUserId, string(str))
	if err == nil {
		// 协程处理DB数据库
		secKillMessage := SecKillMessage{
			RedPacketId:   req.RedPacketId,
			UserId:        req.ReceiveUserId,
			ReceivePoints: tPoints,
			IsComplete:    localutils.ThreeWayOperator(code == 2, true, false),
			Server:        o,
		}
		if doReplace {
			secKillMessage.NewFixedIndex = model.FixedIndex
		}
		SecKillChannel <- secKillMessage

		hLen := len(model.HistoryRewards)
		data := model.HistoryRewards[hLen-1]

		// 发送抢包通知
		go func() {
			nctx := mcontext.NewCtx("@@@" + mcontext.GetOperationID(ctx))
			o.notification.GrabRedPacketNotification(nctx, req)
		}()

		return &pointspb.GrabRedPacketResp{
			Success: true,
			Code:    int32(code),
			Data: &pointspb.GrabMessage{
				UserId: data["userId"].(string),
				Points: data["points"].(int32),
			},
		}, nil
	}
	return &pointspb.GrabRedPacketResp{
		Success: false,
		Code:    int32(code),
	}, err

	//errNum, err := grabredpacket.CacheAtomicSecKill(ctx, o.secKillSHA, req.RedPacketId, req.ReceiveUserId)
	//// 抢红包成功
	//if err == nil {
	//	// 计算红包积分
	//	model, tPoints, flag := grabredpacket.ComputerRandPoints(ctx, req.RedPacketId, req.ReceiveUserId)
	//	if !flag {
	//		return nil, errors.New("redPacket computer error")
	//	}
	//	_, err := grabredpacket.CacheInfoUpdate(ctx, req.RedPacketId, model)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	// 协程处理DB数据库
	//	SecKillChannel <- SecKillMessage{
	//		RedPacketId:   req.RedPacketId,
	//		UserId:        req.ReceiveUserId,
	//		ReceivePoints: tPoints,
	//		Server:        o,
	//	}
	//
	//	hLen := len(model.HistoryRewards)
	//	data := model.HistoryRewards[hLen-1]
	//
	//	return &pointspb.GrabRedPacketResp{
	//		Success: true,
	//		Data: &pointspb.GrabMessage{
	//			UserId: data["userId"].(string),
	//			Points: data["points"].(int32),
	//		},
	//	}, nil
	//} else {
	//	//if grabredpacket.IsRedisEvalError(err) {
	//	//
	//	//}
	//	return &pointspb.GrabRedPacketResp{
	//		Success: false,
	//		ErrNum:  int32(errNum),
	//	}, err
	//}
}

func (o *pointsServer) ReceiveC2CRedPacket(ctx context.Context, req *pointspb.ReceiveC2CRedPacketReq) (*pointspb.ReceiveC2CRedPacketResp, error) {
	redPacket, err := o.pointsDatabase.GetRedPacket(ctx, req.RedPacketId)
	if err != nil {
		return nil, err
	}
	if redPacket.RedPackerState > 1 {
		return nil, errs.ErrData.WithDetail("红包不是可接收状态")
	}
	pointsWater := &relationtb.PointsWater{
		PointsWaterId:   uuid.New().String(),
		UserId:          redPacket.ReceiveUserId,
		Points:          redPacket.Points,
		PointsWaterType: 3,
		RedPacketId:     redPacket.RedPacketId,
		RedPacketType:   redPacket.RedPacketType,
		PointsWaterTime: time.Now(),
	}
	err = o.pointsDatabase.ReceiveC2CRedPacket(ctx, pointsWater)
	if err != nil {
		return nil, err
	}
	return &pointspb.ReceiveC2CRedPacketResp{Success: true}, nil
}

func (o *pointsServer) SendRedPacket(ctx context.Context, data *pointspb.SendRedPacketRep) (*pointspb.SendRedPacketResp, error) {
	var redPacket *relationtb.RedPacket
	if data.RedPacketType == 1 { // 群拼手气红包
		redPacket = &relationtb.RedPacket{
			RedPacketId:    uuid.New().String(),
			RedPacketType:  int8(data.RedPacketType),
			RedPackerState: 1,
			GroupId:        data.GroupId,
			SendUserId:     data.SendUserId,
			Points:         decimal.NewFromFloat32(data.Points),
			RemainPoints:   decimal.NewFromFloat32(data.Points),
			Count:          data.Count,
			RemainCount:    data.Count,
			LastDigits:     data.LastDigits,
			CreateTime:     time.Now(),
		}

		// 构建固定尾数规则
		dLen := len(data.LastDigits)
		if dLen != 0 {
			rs := grabredpacket.BuildReservoirSampling(int64(redPacket.Count))
			redPacket.FixedIndex = rs.Sampling(dLen)
			// 获取白名单用户
			whiteList, err := o.pointsDatabase.GetGroupUsers(ctx, redPacket.GroupId)
			if err != nil {
				var wls []string
				for _, item := range whiteList {
					wls = append(wls, item.UserId)
				}
				redPacket.WhiteList = wls
			}
		}

		// 将红包信息放入秒杀队列
		if _, err := grabredpacket.CacheRedPacketInfo(ctx, redPacket); err != nil {
			return nil, err
		}
	} else if data.RedPacketType == 2 { // 群专属红包
		redPacket = &relationtb.RedPacket{
			RedPacketId:    uuid.New().String(),
			RedPacketType:  int8(data.RedPacketType),
			RedPackerState: 1,
			GroupId:        data.GroupId,
			SendUserId:     data.SendUserId,
			ReceiveUserId:  data.ReceiveUserId,
			Points:         decimal.NewFromFloat32(data.Points),
			CreateTime:     time.Now(),
		}
	} else { // 好友红包
		redPacket = &relationtb.RedPacket{
			RedPacketId:    uuid.New().String(),
			RedPacketType:  int8(data.RedPacketType),
			RedPackerState: 1,
			SendUserId:     data.SendUserId,
			ReceiveUserId:  data.ReceiveUserId,
			Points:         decimal.NewFromFloat32(data.Points),
			CreateTime:     time.Now(),
		}
	}
	if err := o.pointsDatabase.SendRedPacket(ctx, redPacket); err != nil {
		return nil, err
	}
	// 将红包放入延迟队列
	o.delayQueue.SendMsg(redPacket.RedPacketId)

	// 重构消息体
	return &pointspb.SendRedPacketResp{
		RedPacketId:    redPacket.RedPacketId,
		RedPacketState: int32(redPacket.RedPackerState),
	}, nil
}

func (o *pointsServer) BatchRedPacket(ctx context.Context, req *pointspb.BatchRedPacketReq) (*emptypb.Empty, error) {
	// 校验积分
	userPoints, err := o.pointsDatabase.GetUserPoints(ctx, req.SendUserId)
	if err != nil {
		return nil, err
	}
	if userPoints.PointsTotal.LessThan(decimal.NewFromFloat32(req.SumPoints)) {
		return nil, errs.ErrArgs.WithDetail("Beyond the usable points")
	}

	// 更新用户积分数据并批量发送消息
	err = o.batchMsg.BatchRedPacket(ctx, req)
	return &emptypb.Empty{}, err
}

func (o *pointsServer) ResetRedPacket(ctx context.Context, req *pointspb.ResetRedPacketReq) (*emptypb.Empty, error) {
	// 处理积分相关业务
	if err := o.pointsDatabase.ResetRedPacket(ctx, req.RedPacketId); err == nil {
		grabredpacket.DelKey(ctx, []string{req.RedPacketId}...)
	}
	return &emptypb.Empty{}, nil
}

func (o *pointsServer) GetRedPacket(ctx context.Context, req *pointspb.GetRedPacketReq) (*pointspb.GetRedPacketResp, error) {
	data, err := o.pointsDatabase.GetRedPacket(ctx, req.RedPacketId)
	if err != nil {
		return nil, err
	}

	points, _ := data.Points.Float64()
	remainPoints, _ := data.RemainPoints.Float64()
	var updateTime string
	if data.UpdateTime != nil {
		updateTime = data.UpdateTime.Format("2006-01-02 15:04:05")
	} else {
		updateTime = ""
	}

	return &pointspb.GetRedPacketResp{
		RedPacketId:    data.RedPacketId,
		RedPacketType:  int32(data.RedPacketType),
		RedPackerState: int32(data.RedPackerState),
		GroupId:        data.GroupId,
		SendUserId:     data.SendUserId,
		ReceiveUserId:  data.ReceiveUserId,
		Points:         float32(points),
		RemainPoints:   float32(remainPoints),
		Count:          data.Count,
		RemainCount:    data.RemainCount,
		LastDigits:     data.LastDigits,
		FixedIndex:     data.FixedIndex,
		WhiteList:      data.WhiteList,
		CreateTime:     data.CreateTime.Format("2006-01-02 15:04:05"),
		UpdateTime:     updateTime,
	}, nil
}

func (o *pointsServer) RedPacketOverTime(ctx context.Context, req *pointspb.RedPacketOverTimeReq) (*pointspb.RedPacketOverTimeResp, error) {
	if err := o.pointsDatabase.RedPacketOverTime(ctx, req.RedPacketId); err != nil {
		return nil, err
	}
	return &pointspb.RedPacketOverTimeResp{}, nil
}

func (o *pointsServer) GetRedPacketDetail(ctx context.Context, req *pointspb.GetRedPacketDetailReq) (*pointspb.GetRedPacketDetailResp, error) {
	// 群红包检索两个数据源
	if req.RedPacketType == 1 {
		// 首先判断redis中是否存在红包消息
		rModel, err := grabredpacket.GetRedPacketModel(ctx, req.RedPacketId)
		if err != nil {
			return nil, err
		}
		if rModel != nil {
			points, _ := rModel.Points.Float64()
			remainPoints, _ := rModel.RemainPoints.Float64()
			infoResp := &pointspb.GetRedPacketResp{
				RedPacketId:    rModel.RedPacketId,
				RedPacketType:  1,
				RedPackerState: 1,
				Points:         float32(points),
				RemainPoints:   float32(remainPoints),
				Count:          rModel.Count,
				RemainCount:    rModel.RemainCount,
			}

			if len(rModel.HistoryRewards) == 0 {
				return &pointspb.GetRedPacketDetailResp{
					Info: infoResp,
				}, nil
			}
			ids := make([]string, len(rModel.HistoryRewards))
			var detailData map[string]interface{}
			for _, item := range rModel.HistoryRewards {
				ids = append(ids, item["userId"].(string))
				detailData[item["userId"].(string)] = item
			}
			users, err := o.userClient.GetUsersInfo(ctx, ids)
			if err != nil {
				return nil, err
			}
			waterResp := make([]*pointspb.ReceiveWaterResp, len(rModel.HistoryRewards))
			for _, user := range users {
				cache := detailData[user.UserID].(map[string]interface{})
				waterResp = append(waterResp, &pointspb.ReceiveWaterResp{
					RedPacketId:   req.RedPacketId,
					ReceiveUserId: user.UserID,
					Points:        cache["points"].(float32),
					CreateTime:    cache["createTime"].(string),
					NickName:      user.Nickname,
					FaceUrl:       user.FaceURL,
				})
			}
			return &pointspb.GetRedPacketDetailResp{
				Info:  infoResp,
				Water: waterResp,
			}, nil
		}
		// redis中不存在则直接查询关系数据库
		// 获取红包信息
		infoReq := &pointspb.GetRedPacketReq{
			RedPacketId: req.RedPacketId,
		}
		info, err := o.GetRedPacket(ctx, infoReq)
		if err != nil {
			return nil, err
		}
		// 获取红包流水
		items, err := o.pointsDatabase.RedPacketWater(ctx, req.RedPacketId)
		if err != nil {
			return nil, err
		}
		water := make([]*pointspb.ReceiveWaterResp, len(items))
		for _, item := range items {
			points, ok := item.Points.Float64()
			if !ok {
				points = 0.00
			}
			data := &pointspb.ReceiveWaterResp{
				ReceiveWaterId: item.ReceiveWaterId,
				RedPacketId:    item.RedPacketId,
				ReceiveUserId:  item.ReceiveUserId,
				Points:         float32(points),
				NickName:       item.Name,
				FaceUrl:        item.FaceUrl,
				CreateTime:     item.CreateTime.Format("2006-01-02 15:04:05"),
			}
			water = append(water, data)
		}

		return &pointspb.GetRedPacketDetailResp{
			Info:  info,
			Water: water,
		}, nil
	} else {
		infoReq := &pointspb.GetRedPacketReq{
			RedPacketId: req.RedPacketId,
		}
		info, err := o.GetRedPacket(ctx, infoReq)
		if err != nil {
			return nil, err
		}
		return &pointspb.GetRedPacketDetailResp{
			Info: info,
		}, nil
	}
}

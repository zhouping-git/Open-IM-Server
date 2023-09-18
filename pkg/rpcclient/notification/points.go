package notification

import (
	"context"
	localconstant "github.com/OpenIMSDK/Open-IM-Server/pkg/common/constant"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/controller"
	pointspb "github.com/OpenIMSDK/Open-IM-Server/pkg/proto/points"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/rpcclient"
	"github.com/OpenIMSDK/protocol/sdkws"
	"github.com/OpenIMSDK/tools/log"
	"github.com/OpenIMSDK/tools/utils"
)

func NewPointsNotificationSender(
	pointsDb controller.PointsDatabaseInterface,
	msgRpcClient *rpcclient.MessageRpcClient,
	userRpcClient *rpcclient.UserRpcClient,
	getUsersInfo func(ctx context.Context, userIDs []string) ([]CommonUser, error),
	getGroupInfo func(ctx context.Context, groupID string) (*sdkws.GroupInfo, error),
) *PointsNotificationSender {
	return &PointsNotificationSender{
		NotificationSender: rpcclient.NewNotificationSender(rpcclient.WithRpcClient(msgRpcClient), rpcclient.WithUserRpcClient(userRpcClient)),
		pointsDb:           pointsDb,
		getUsersInfo:       getUsersInfo,
		getGroupInfo:       getGroupInfo,
	}
}

type PointsNotificationSender struct {
	*rpcclient.NotificationSender
	pointsDb     controller.PointsDatabaseInterface
	getUsersInfo func(ctx context.Context, userIDs []string) ([]CommonUser, error)
	getGroupInfo func(ctx context.Context, groupID string) (*sdkws.GroupInfo, error)
}

func (p *PointsNotificationSender) getRedPacket(ctx context.Context, redPacketId string) (*pointspb.GetRedPacketResp, error) {
	data, err := p.pointsDb.GetRedPacket(ctx, redPacketId)
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

func (p *PointsNotificationSender) getUsersInfoMap(ctx context.Context, userIDs []string) (map[string]*sdkws.UserInfo, error) {
	users, err := p.getUsersInfo(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[string]*sdkws.UserInfo)
	for _, user := range users {
		result[user.GetUserID()] = user.(*sdkws.UserInfo)
	}
	return result, nil
}

// GrabRedPacketNotification 抢红包成功通知
func (p *PointsNotificationSender) GrabRedPacketNotification(ctx context.Context, req *pointspb.GrabRedPacketReq) (err error) {
	defer log.ZDebug(ctx, "return")
	defer func() {
		if err != nil {
			log.ZError(ctx, utils.GetFuncName(1)+" failed", err)
		}
	}()

	redPacket, err := p.getRedPacket(ctx, req.RedPacketId)
	if err != nil {
		return err
	}
	userMap, err := p.getUsersInfoMap(ctx, []string{req.ReceiveUserId, req.SendUserId})
	if err != nil {
		return err
	}

	// 由于当前无法适配指定群用户发送消息，固先只发送一条全局消息
	//if req.ReceiveUserId != req.SendUserId {
	//	tips := &pointspb.GrabRedPacketTips{
	//		RedPacket:   redPacket,
	//		ReceiveUser: userMap[req.ReceiveUserId],
	//	}
	//	// 发红包用户通知
	//	if err = p.Notification(ctx, req.ReceiveUserId, req.SendUserId, localconstant.GrabToSenderNotification, tips, rpcclient.WithRpcGroupId(req.GroupId)); err != nil {
	//		return err
	//	}
	//}

	tips := &pointspb.GrabRedPacketTips{
		RedPacket:   redPacket,
		ReceiveUser: userMap[req.ReceiveUserId],
		SendUser:    userMap[req.SendUserId],
	}
	// 抢红包用户通知
	return p.Notification(ctx, req.ReceiveUserId, req.ReceiveUserId, localconstant.GrabToReceiveNotification, tips, rpcclient.WithRpcGroupId(req.GroupId))
}

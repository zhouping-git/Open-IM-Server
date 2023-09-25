package rpcclient

import (
	"context"
	"encoding/json"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/apistruct"
	localutils "github.com/OpenIMSDK/Open-IM-Server/pkg/utils"
	"github.com/OpenIMSDK/protocol/constant"
	"github.com/OpenIMSDK/protocol/msg"
	"github.com/OpenIMSDK/protocol/sdkws"
	"github.com/OpenIMSDK/tools/errs"
	"github.com/OpenIMSDK/tools/log"
	"github.com/OpenIMSDK/tools/mcontext"
	"github.com/OpenIMSDK/tools/utils"
	"github.com/go-playground/validator/v10"
	"google.golang.org/protobuf/proto"
	"reflect"
)

//////////////自定义批量消息发送实现//////////////////////////////////////////////

func newMessageTypeConf() map[int32]string {
	return map[int32]string{
		constant.Picture:                      "PictureElem",
		constant.Voice:                        "SoundElem",
		constant.Video:                        "VideoElem",
		constant.File:                         "FileElem",
		constant.Custom:                       "CustomElem",
		constant.Revoke:                       "RevokeElem",
		constant.OANotification:               "OANotificationElem",
		constant.CustomNotTriggerConversation: "CustomElem",
		constant.CustomOnlineOnly:             "CustomElem",
	}
}

type BatchMsgSender[T BatchMsgType] struct {
	validate        *validator.Validate
	messageTypeConf map[int32]string
	sendMsg         func(ctx context.Context, req *msg.SendMsgReq) (*msg.SendMsgResp, error)
	getUserInfo     func(ctx context.Context, userID string) (*sdkws.UserInfo, error)
}

type BatchMsgSenderOptions[T BatchMsgType] func(*BatchMsgSender[T])

func WithMsgLocalSendMsg[T BatchMsgType](sendMsg func(ctx context.Context, req *msg.SendMsgReq) (*msg.SendMsgResp, error)) BatchMsgSenderOptions[T] {
	return func(s *BatchMsgSender[T]) {
		s.sendMsg = sendMsg
	}
}

func WithMsgRpcClient[T BatchMsgType](msgRpcClient *MessageRpcClient) BatchMsgSenderOptions[T] {
	return func(s *BatchMsgSender[T]) {
		s.sendMsg = msgRpcClient.SendMsg
	}
}

func WithMsgUserRpcClient[T BatchMsgType](userRpcClient *UserRpcClient) BatchMsgSenderOptions[T] {
	return func(s *BatchMsgSender[T]) {
		s.getUserInfo = userRpcClient.GetUserInfo
	}
}

func NewBatchMsgSender[T BatchMsgType](opts ...BatchMsgSenderOptions[T]) *BatchMsgSender[T] {
	batchMsgSender := &BatchMsgSender[T]{messageTypeConf: newMessageTypeConf()}
	for _, opt := range opts {
		opt(batchMsgSender)
	}
	return batchMsgSender
}

type batchMsgOpt struct {
	WithSpecifyRecipient []string // 指定具体发送的用户
	WithOperateStatus    int32    // 指定操作状态
	IsOnlineOnly         bool
	NotOfflinePush       bool
	OfflinePushInfo      *sdkws.OfflinePushInfo
}

type BatchMsgOptions func(*batchMsgOpt)

func WithSpecifyRecipient(specifyRecipient []string) BatchMsgOptions {
	return func(opt *batchMsgOpt) {
		opt.WithSpecifyRecipient = specifyRecipient
	}
}

func WithOperateStatus(operateStatus int32) BatchMsgOptions {
	return func(opt *batchMsgOpt) {
		opt.WithOperateStatus = operateStatus
	}
}

func IsOnlineOnly(t bool) BatchMsgOptions {
	return func(opt *batchMsgOpt) {
		opt.IsOnlineOnly = t
	}
}

func NotOfflinePush(t bool) BatchMsgOptions {
	return func(opt *batchMsgOpt) {
		opt.NotOfflinePush = t
	}
}

func OfflinePushInfo(info *sdkws.OfflinePushInfo) BatchMsgOptions {
	return func(opt *batchMsgOpt) {
		opt.OfflinePushInfo = info
	}
}

func (s *BatchMsgSender[T]) SetOptions(options map[string]bool, value bool) {
	utils.SetSwitchFromOptions(options, constant.IsHistory, value)
	utils.SetSwitchFromOptions(options, constant.IsPersistent, value)
	utils.SetSwitchFromOptions(options, constant.IsSenderSync, value)
	utils.SetSwitchFromOptions(options, constant.IsConversationUpdate, value)
}

type BatchMsgType interface {
	apistruct.PictureElem | apistruct.SoundElem | apistruct.VideoElem | apistruct.FileElem | apistruct.RevokeElem | apistruct.OANotificationElem | apistruct.CustomElem
}

var contentTypeConfig = []int32{
	constant.Picture, constant.Voice, constant.Video, constant.File, constant.Custom, constant.Revoke, constant.OANotification, constant.CustomNotTriggerConversation, constant.CustomOnlineOnly,
}

var sessionTypeConfig = []int32{
	constant.SingleChatType, constant.NotificationChatType, constant.SuperGroupChatType,
}

func (s *BatchMsgSender[T]) buildBatchMsg(ctx context.Context, sendID, recvID string, contentType, sessionType int32, m T, opts ...BatchMsgOptions) (err error) {
	content, err := json.Marshal(&m)
	if err != nil {
		log.ZError(ctx, "MsgClient BatchMsg json.Marshal failed", err, "sendID", sendID, "recvID", recvID, "contentType", contentType, "msg", m)
		return err
	}
	batchMsgOpt := &batchMsgOpt{
		WithOperateStatus: 0,
		IsOnlineOnly:      false,
		OfflinePushInfo: &sdkws.OfflinePushInfo{
			Title: "",
			Desc:  "",
			Ex:    "",
		},
	}
	for _, opt := range opts {
		opt(batchMsgOpt)
	}

	var req msg.SendMsgReq
	var msg sdkws.MsgData

	// 配置项生成
	options := make(map[string]bool, 5)
	if batchMsgOpt.IsOnlineOnly {
		s.SetOptions(options, false)
	}
	if batchMsgOpt.NotOfflinePush {
		utils.SetSwitchFromOptions(options, constant.IsOfflinePush, false)
	}
	if contentType == constant.CustomOnlineOnly {
		s.SetOptions(options, false)
	} else if contentType == constant.CustomNotTriggerConversation {
		utils.SetSwitchFromOptions(options, constant.IsConversationUpdate, false)
	}
	msg.Options = options
	// 获取用户信息
	if s.getUserInfo != nil {
		userInfo, err := s.getUserInfo(ctx, sendID)
		if err != nil {
			log.ZWarn(ctx, "getUserInfo failed", err, "sendID", sendID)
		} else {
			msg.SenderNickname = userInfo.Nickname
			msg.SenderFaceURL = userInfo.FaceURL
		}
	}

	msg.SendID = sendID
	msg.RecvID = recvID
	msg.Content = content
	msg.MsgFrom = constant.SysMsgType
	msg.ContentType = contentType

	if constant.OANotification == contentType {
		msg.SessionType = constant.NotificationChatType
	} else {
		msg.SessionType = sessionType
	}
	if contentType == constant.OANotification {
		var tips sdkws.TipsComm
		tips.JsonDetail = string(content)
		msg.Content, err = proto.Marshal(&tips)
		if err != nil {
			log.ZError(ctx, "Marshal failed ", err, "tips", tips.String())
		}
	}

	if msg.SessionType == constant.SuperGroupChatType || msg.SessionType == constant.GroupChatType {
		msg.GroupID = recvID
	}
	msg.CreateTime = utils.GetCurrentTimestampByMill()
	msg.ClientMsgID = utils.GetMsgID(sendID)
	msg.OfflinePushInfo = batchMsgOpt.OfflinePushInfo

	// 自定义指定用户
	if len(batchMsgOpt.WithSpecifyRecipient) > 0 {
		msg.SpecifyRecipient = batchMsgOpt.WithSpecifyRecipient
	}
	msg.OperateStatus = batchMsgOpt.WithOperateStatus

	req.MsgData = &msg
	_, err = s.sendMsg(ctx, &req)
	if err == nil {
		log.ZDebug(ctx, "MsgClient Batch SendMsg success", "req", &req)
	} else {
		log.ZError(ctx, "MsgClient Batch SendMsg failed", err, "req", &req)
	}
	return err
}

func (s *BatchMsgSender[T]) SendBatchMsg(ctx context.Context, sendID, recvID string, contentType, sessionType int32, m []T, opts ...BatchMsgOptions) error {
	if !localutils.ElementInSlice(sessionTypeConfig, sessionType) {
		return errs.ErrArgs.WithDetail("sessionType is not in sessionTypeConfig")
	}
	if !localutils.ElementInSlice(contentTypeConfig, contentType) {
		return errs.ErrArgs.WithDetail("contentType is not in contentTypeConfig")
	}
	if reflect.TypeOf(m).Name() != s.messageTypeConf[contentType] {
		return errs.ErrArgs.WithDetail("Content is not contentType")
	}

	for _, item := range m {
		go func() {
			nctx := mcontext.NewCtx("@@@" + mcontext.GetOperationID(ctx))
			err := s.buildBatchMsg(nctx, sendID, recvID, contentType, sessionType, item, opts...)
			if err != nil {
				return
			}
		}()
	}
	return nil
}

func (s *BatchMsgSender[T]) SendOnlyMsg(ctx context.Context, sendID, recvID string, contentType, sessionType int32, m T, opts ...BatchMsgOptions) error {
	if !localutils.ElementInSlice(sessionTypeConfig, sessionType) {
		return errs.ErrArgs.WithDetail("sessionType is not in sessionTypeConfig")
	}
	if !localutils.ElementInSlice(contentTypeConfig, contentType) {
		return errs.ErrArgs.WithDetail("contentType is not in contentTypeConfig")
	}
	if reflect.TypeOf(m).Name() != s.messageTypeConf[contentType] {
		return errs.ErrArgs.WithDetail("Content is not contentType")
	}

	return s.buildBatchMsg(ctx, sendID, recvID, contentType, sessionType, m, opts...)
}

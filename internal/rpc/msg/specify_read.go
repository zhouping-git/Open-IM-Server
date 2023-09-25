package msg

import (
	"context"
	localconstant "github.com/OpenIMSDK/Open-IM-Server/pkg/common/constant"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/proto/specifyread"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/rpcclient"
	"github.com/OpenIMSDK/protocol/constant"
	"github.com/OpenIMSDK/protocol/msg"
	"github.com/OpenIMSDK/tools/log"
)

func (m *msgServer) SetMarkUserReadMsg(ctx context.Context, req *msg.SetMarkUserReadMsgReq) (*msg.SetMarkUserReadMsgResp, error) {
	conversation, err := m.Conversation.GetConversation(ctx, req.UserID, req.ConversationID)
	if err != nil {
		return nil, err
	}
	_, err = m.MsgDatabase.MarkUserReadMsg(ctx, req.ConversationID, req.UserID, req.Seq)
	if err != nil {
		return nil, err
	}
	//sData := sdkws.MsgData(docData)
	tip := &specifyread.MarkUserReadMsgTip{
		ConversationID: req.ConversationID,
		UserID:         req.UserID,
		Seq:            req.Seq,
		//MsgData:        ,
	}

	if conversation.ConversationType == constant.SingleChatType || conversation.ConversationType == constant.NotificationChatType {
		err = m.notificationSender.NotificationWithSesstionType(ctx, req.UserID, req.UserID, localconstant.UserReadReceipt, conversation.ConversationType, tip,
			rpcclient.WithRpcSpecifyRecipient([]string{req.UserID}))
	} else if conversation.ConversationType == constant.SuperGroupChatType {
		err = m.notificationSender.NotificationWithSesstionType(ctx, req.UserID, conversation.GroupID, localconstant.UserReadReceipt, conversation.ConversationType, tip,
			rpcclient.WithRpcSpecifyRecipient([]string{req.UserID}))
	}
	if err != nil {
		return nil, err
	}
	return &msg.SetMarkUserReadMsgResp{}, nil
}

func (m *msgServer) SetMarkMsgOperateStatus(ctx context.Context, req *msg.SetMarkMsgOperateStatusReq) (*msg.SetMarkMsgOperateStatusResp, error) {
	conversation, err := m.Conversation.GetConversation(ctx, req.UserID, req.ConversationID)
	if err != nil {
		return nil, err
	}
	if req.IsAddRead {
		_, err = m.MsgDatabase.MarkMsgOperateStatus(ctx, req.ConversationID, req.Seq, req.State, req.UserID)
	} else {
		_, err = m.MsgDatabase.MarkMsgOperateStatus(ctx, req.ConversationID, req.Seq, req.State)
	}
	if err != nil {
		return nil, err
	}
	tip := &specifyread.MarkUserReadMsgTip{
		ConversationID: req.ConversationID,
		UserID:         req.UserID,
		Seq:            req.Seq,
		//MsgData:        ,
	}

	recID := m.conversationAndGetRecvID(conversation, req.UserID)

	if conversation.ConversationType == constant.SingleChatType || conversation.ConversationType == constant.NotificationChatType {
		err = m.notificationSender.NotificationWithSesstionType(ctx, req.UserID, recID, localconstant.MsgStatusReceipt, conversation.ConversationType, tip)
		if err != nil {
			log.ZWarn(ctx, "C2C operate status err", err)
			return nil, err
		}
	} else if conversation.ConversationType == constant.SuperGroupChatType {
		if req.SpecifyRecipient != nil && len(req.SpecifyRecipient) > 0 {
			err = m.notificationSender.NotificationWithSesstionType(ctx, req.UserID, recID, localconstant.MsgStatusReceipt, conversation.ConversationType, tip,
				rpcclient.WithRpcSpecifyRecipient(req.SpecifyRecipient))
		} else {
			err = m.notificationSender.NotificationWithSesstionType(ctx, req.UserID, recID, localconstant.MsgStatusReceipt, conversation.ConversationType, tip)
		}
		if err != nil {
			log.ZWarn(ctx, "group operate status err", err)
			return nil, err
		}
	}
	return &msg.SetMarkMsgOperateStatusResp{}, nil
}

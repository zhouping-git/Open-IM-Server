package rpcclient

import (
	"context"
	"encoding/json"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/apistruct"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/config"
	pointspb "github.com/OpenIMSDK/Open-IM-Server/pkg/proto/points"
	"github.com/OpenIMSDK/tools/discoveryregistry"
	"github.com/OpenIMSDK/tools/utils"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/grpc"
)

type Points struct {
	conn   grpc.ClientConnInterface
	Client pointspb.PointsClient
	discov discoveryregistry.SvcDiscoveryRegistry
}

func NewPoints(discov discoveryregistry.SvcDiscoveryRegistry) *Points {
	conn, err := discov.GetConn(context.Background(), config.Config.RpcRegisterName.OpenImPointsName)
	if err != nil {
		panic(err)
	}
	client := pointspb.NewPointsClient(conn)
	return &Points{
		conn:   conn,
		Client: client,
		discov: discov,
	}
}

type PointsRpcClient Points

func NewPointsRpcClient(discov discoveryregistry.SvcDiscoveryRegistry) PointsRpcClient {
	return PointsRpcClient(*NewPoints(discov))
}

// SendRedPacket 提供给消息服务(websocket)执行红包发送操作--此方法暂时不使用
func (p *PointsRpcClient) SendRedPacket(ctx context.Context, req *apistruct.CustomContextElem) ([]byte, error) {
	data := apistruct.RedPacketElem{}
	if err := mapstructure.WeakDecode(req.Data, &data); err != nil {
		return nil, err
	}
	newReq := &pointspb.SendRedPacketRep{
		RedPacketId:   data.RedPacketId,
		RedPacketType: data.RedPacketType,
		GroupId:       data.GroupId,
		SendUserId:    data.SendUserId,
		ReceiveUserId: data.ReceiveUserId,
		Points:        data.Points,
		Count:         data.Count,
		Title:         data.Title,
		LastDigits:    data.LastDigits,
	}
	resp, err := p.Client.SendRedPacket(ctx, newReq)
	if err != nil {
		return nil, err
	}
	data.RedPacketId = resp.RedPacketId
	data.RedPacketState = 1
	//req.Data = utils.StructToJsonString(data)
	//return []byte(utils.StructToJsonString(req)), nil
	req.Data = data
	msgReq, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resultData := map[string]interface{}{
		"data": string(msgReq),
	}
	return []byte(utils.StructToJsonString(resultData)), nil
}

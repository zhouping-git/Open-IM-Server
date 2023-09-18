package api

import (
	"github.com/OpenIMSDK/Open-IM-Server/pkg/proto/points"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/rpcclient"
	"github.com/OpenIMSDK/tools/a2r"
	"github.com/gin-gonic/gin"
)

type PointsApi rpcclient.Points

func NewPointsApi(client rpcclient.Points) PointsApi {
	return PointsApi(client)
}

func (o *PointsApi) GetUserPoints(c *gin.Context) {
	a2r.Call(points.PointsClient.GetUserPoints, o.Client, c)
}

func (o *PointsApi) UserPointsRecharge(c *gin.Context) {
	a2r.Call(points.PointsClient.UserPointsRecharge, o.Client, c)
}

func (o *PointsApi) UserPointsWithdraw(c *gin.Context) {
	a2r.Call(points.PointsClient.UserPointsWithdraw, o.Client, c)
}

func (o *PointsApi) PointsWaterForType(c *gin.Context) {
	a2r.Call(points.PointsClient.PointsWaterForType, o.Client, c)
}

func (o *PointsApi) SendRedPacket(c *gin.Context) {
	a2r.Call(points.PointsClient.SendRedPacket, o.Client, c)
}

func (o *PointsApi) BatchRedPacket(c *gin.Context) {
	a2r.Call(points.PointsClient.BatchRedPacket, o.Client, c)
}

func (o *PointsApi) ResetRedPacket(c *gin.Context) {
	a2r.Call(points.PointsClient.ResetRedPacket, o.Client, c)
}

func (o *PointsApi) GrabRedPacket(c *gin.Context) {
	a2r.Call(points.PointsClient.GrabRedPacket, o.Client, c)
}

func (o *PointsApi) ReceiveC2CRedPacket(c *gin.Context) {
	a2r.Call(points.PointsClient.ReceiveC2CRedPacket, o.Client, c)
}

func (o *PointsApi) GetRedPacketDetail(c *gin.Context) {
	a2r.Call(points.PointsClient.GetRedPacketDetail, o.Client, c)
}

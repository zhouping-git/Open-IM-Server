package msg

import (
	"encoding/json"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/apistruct"
	"github.com/OpenIMSDK/protocol/sdkws"
	"github.com/mitchellh/mapstructure"
)

// 校验是否是指定的自定义消息扩展类型，并返回对应消息体
func parseCustomMessage(msgData *sdkws.MsgData, key int) (*apistruct.CustomContextElem, bool) {
	//var content interface{}
	content := make(map[string]interface{})
	if err := json.Unmarshal(msgData.Content, &content); err != nil {
		return nil, false
	}
	//customType := content["data"].(map[string]interface{})["customType"].(string)
	if err := json.Unmarshal([]byte(content["data"].(string)), &content); err != nil {
		return nil, false
	}
	resp := &apistruct.CustomContextElem{}
	if err := mapstructure.WeakDecode(content, &resp); err != nil {
		return nil, false
	}
	if resp.CustomType == int32(key) {
		return resp, true
	}
	return nil, false
}

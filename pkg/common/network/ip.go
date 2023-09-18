package network

import (
	"github.com/OpenIMSDK/Open-IM-Server/pkg/utils"
	"github.com/OpenIMSDK/protocol/constant"
)

// GetRpcRegisterIP 获取本地RPC注册IP，重新实现调用重写的获取IP方法
func GetRpcRegisterIP(configIP string) (string, error) {
	registerIP := configIP
	if registerIP == "" {
		ip, err := utils.GetLocalIP()
		if err != nil {
			return "", err
		}
		registerIP = ip
	}
	return registerIP, nil
}

func GetListenIP(configIP string) string {
	if configIP == "" {
		return constant.LocalHost
	} else {
		return configIP
	}
}

package apistruct

//// CustomDataType 自定义消息扩展类型适配的消息体类型
//type CustomDataType interface {
//	RedPacketElem
//}

// CustomContextElem 自定义消息解析定义的具体类型
type CustomContextElem struct {
	CustomType int32       `mapstructure:"customType" json:"customType"`
	Data       interface{} `mapstructure:"data" json:"data"`
}

// RedPacketElem 红包消息反射体
type RedPacketElem struct {
	RedPacketId    string  `mapstructure:"redPacketId" json:"redPacketId"`
	RedPacketType  int32   `mapstructure:"redPacketType" json:"redPacketType" validate:"required"`
	RedPacketState int32   `mapstructure:"redPacketState" json:"redPacketState"`
	GroupId        string  `mapstructure:"groupId" json:"groupId"`
	SendUserId     string  `mapstructure:"sendUserId" json:"sendUserId" validate:"required"`
	ReceiveUserId  string  `mapstructure:"receiveUserId" json:"receiveUserId"`
	Points         float32 `mapstructure:"points" json:"points" validate:"required"`
	Count          int32   `mapstructure:"count" json:"count"`
	Title          string  `mapstructure:"title" json:"title"`
	LastDigits     []int32 `mapstructure:"lastDigits" json:"lastDigits"`
}

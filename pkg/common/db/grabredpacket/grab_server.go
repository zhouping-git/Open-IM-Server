package grabredpacket

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/table/relation"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/utils"
	"github.com/shopspring/decimal"
	"math/rand"
	"strconv"
	"time"
)

const (
	redPacketUserFolder = "RED_PACKET_USER"
	redPacketInfoFolder = "RED_PACKET_INFO"
)

func BuildGrabUserKey(redPacketId string) string {
	return fmt.Sprintf("%s:%s-has", redPacketUserFolder, redPacketId)
}

func BuildInfoKey(redPacketId string) string {
	return fmt.Sprintf("%s:%s-info", redPacketInfoFolder, redPacketId)
}

// CacheRedPacketGrabUser 缓存红包已抢用户
func CacheRedPacketGrabUser(ctx context.Context, receiveWater *relation.ReceiveWater) (int64, error) {
	key := BuildGrabUserKey(receiveWater.RedPacketId)
	val, err := SetAdd(ctx, key, receiveWater.ReceiveUserId)
	return val, err
}

// CacheRedPacketInfo 缓存红包详细信息
func CacheRedPacketInfo(ctx context.Context, redPacket *relation.RedPacket) (bool, error) {
	key := BuildInfoKey(redPacket.RedPacketId)

	//var mData map[string]interface{}
	//err := mapstructure.WeakDecode(redPacket, &mData)
	//if err != nil {
	//	return false, err
	//}
	//
	//conv := utils.NewStructToMap(redPacket)
	//exclude := []string{"GroupId", "SendUserId", "ReceiveUserId", "RedPacketType", "RedPackerState", "CreateTime", "UpdateTime"}
	//newData := conv.ToConvert(utils.WithExcludeFields(exclude))
	//fmt.Println(newData)

	points, _ := redPacket.Points.Float64()
	remainPoints, _ := redPacket.RemainPoints.Float64()
	mData := map[string]interface{}{
		"redPacketId":  redPacket.RedPacketId,
		"points":       points,
		"remainPoints": remainPoints,
		"count":        redPacket.Count,
		"remainCount":  redPacket.RemainCount,
	}
	lastDigits, err := json.Marshal(redPacket.LastDigits)
	if err == nil {
		mData["lastDigits"] = lastDigits
	}
	fixedIndex, err := json.Marshal(redPacket.FixedIndex)
	if err == nil {
		mData["fixedIndex"] = fixedIndex
	}
	whiteList, err := json.Marshal(redPacket.WhiteList)
	if err == nil {
		mData["whiteList"] = whiteList
	}

	return SetMapForever(ctx, key, mData)
}

func CacheRedPacketInfos(ctx context.Context, redPackets []*relation.RedPacket) (bool, error) {
	var dataMap map[string]map[string]interface{}
	for _, redPacket := range redPackets {
		key := BuildInfoKey(redPacket.RedPacketId)

		points, _ := redPacket.Points.Float64()
		remainPoints, _ := redPacket.RemainPoints.Float64()
		mData := map[string]interface{}{
			"redPacketId":  redPacket.RedPacketId,
			"points":       points,
			"remainPoints": remainPoints,
			"count":        redPacket.Count,
			"remainCount":  redPacket.RemainCount,
		}
		lastDigits, err := json.Marshal(redPacket.LastDigits)
		if err == nil {
			mData["lastDigits"] = lastDigits
		}
		fixedIndex, err := json.Marshal(redPacket.FixedIndex)
		if err == nil {
			mData["fixedIndex"] = fixedIndex
		}
		whiteList, err := json.Marshal(redPacket.WhiteList)
		if err == nil {
			mData["whiteList"] = whiteList
		}

		dataMap[key] = mData
	}

	_, err := BatchSetMapForever(ctx, dataMap)
	if err != nil {
		return false, err
	}
	return true, nil
}

func CacheInfoUpdate(ctx context.Context, redPacketId string, model *RedPacketModel) (bool, error) {
	key := BuildInfoKey(redPacketId)
	mData, _ := json.Marshal(model.HistoryRewards)
	fields := map[string]interface{}{
		"count":          model.Count,
		"points":         model.Points,
		"remainCount":    model.RemainCount,
		"remainPoints":   model.RemainPoints,
		"historyRewards": mData,
	}
	return SetMapForever(ctx, key, fields)
}

// CacheAtomicSecKill 原子秒杀实现
func CacheAtomicSecKill(ctx context.Context, secKillSHA, redPacketId, receiveUserId, modelStr string) (int64, error) {
	grabKey := BuildGrabUserKey(redPacketId)
	redPacketKey := BuildInfoKey(redPacketId)
	res, err := EvalSHA(ctx, secKillSHA, []string{grabKey, redPacketKey, receiveUserId, modelStr})
	if err != nil {
		return -5, redisEvalError{}
	}

	redPacketPointsRes, flag := res.(int64)
	if !flag {
		return -6, redPacketPointsResError{redPacketPointsRes}
	}

	// 此处的-1, -2, -3, -4 和 >=0的判断依据, 与secKillSHA变量lua脚本的返回值保持一致
	switch {
	case redPacketPointsRes == -1:
		return -1, userHasRedPacketError{userId: receiveUserId, redPacketId: redPacketId}
	case redPacketPointsRes == -2:
		return -2, noSuchRedPacketError{userId: receiveUserId, redPacketId: redPacketId}
	case redPacketPointsRes == -3:
		return -3, noRedPacketPointsError{userId: receiveUserId, redPacketId: redPacketId}
	case redPacketPointsRes == -4:
		return -4, redPacketCompleteError{userId: receiveUserId, redPacketId: redPacketId}
	case redPacketPointsRes == 1: // 抢红包成功并存在可抢红包
		return 1, nil
	case redPacketPointsRes == 2: // 抢红包成功，后续已无可抢红包处理后续DB落库业务
		return 2, nil
	default:
		return -1, redPacketPointsResError{redPacketPointsRes}
	}
}

type RedPacketModel struct {
	RedPacketId    string                   `mapstructure:"redPacketId"`
	Points         decimal.Decimal          `mapstructure:"points"`         //红包积分
	RemainPoints   decimal.Decimal          `mapstructure:"remainPoints"`   //剩余红包积分
	Count          int32                    `mapstructure:"count"`          //红包个数
	RemainCount    int32                    `mapstructure:"remainCount"`    //剩余红包个数
	LastDigits     []int32                  `mapstructure:"lastDigits"`     // 尾数规则切片
	UsedLastDigits []int32                  `mapstructure:"usedLastDigits"` // 已使用固定尾数
	FixedIndex     []int32                  `mapstructure:"fixedIndex"`     // 固定尾数的红包索引
	WhiteList      []string                 `mapstructure:"whiteList"`      //不会抢到固定位数的用户白名单
	HistoryRewards []map[string]interface{} `mapstructure:"historyRewards"` //历史红包记录
}

const RedPacketMinMoney float32 = 0.01

func GetRedPacketModel(ctx context.Context, redPacketId string) (*RedPacketModel, error) {
	key := BuildInfoKey(redPacketId)
	data, err := GetMapAll(ctx, key)
	if err != nil {
		return nil, err
	}

	result := RedPacketModel{}
	for k, v := range data {
		switch k {
		case "redPacketId":
			result.RedPacketId = v
		case "points":
			result.Points, _ = decimal.NewFromString(v)
		case "remainPoints":
			result.RemainPoints, _ = decimal.NewFromString(v)
		case "count":
			tempVal, err := strconv.Atoi(v)
			if err == nil {
				result.Count = int32(tempVal)
			}
		case "remainCount":
			tempVal, err := strconv.Atoi(v)
			if err == nil {
				result.RemainCount = int32(tempVal)
			}
		case "lastDigits":
			result.LastDigits = []int32{}
			if v != "null" {
				var tempVal []int32
				err := json.Unmarshal([]byte(v), &tempVal)
				if err == nil {
					result.LastDigits = tempVal
				}
			}
		case "usedLastDigits":
			result.UsedLastDigits = []int32{}
			if v != "null" {
				var tempVal []int32
				err := json.Unmarshal([]byte(v), &tempVal)
				if err == nil {
					result.UsedLastDigits = tempVal
				}
			}
		case "fixedIndex":
			result.FixedIndex = []int32{}
			if v != "null" {
				var tempVal []int32
				err := json.Unmarshal([]byte(v), &tempVal)
				if err == nil {
					result.FixedIndex = tempVal
				}
			}
		case "whiteList":
			result.WhiteList = []string{}
			if v != "null" {
				var tempVal []string
				err := json.Unmarshal([]byte(v), &tempVal)
				if err == nil {
					result.WhiteList = tempVal
				}
			}
		case "historyRewards":
			result.HistoryRewards = []map[string]interface{}{}
			if v != "null" {
				var tempVal []map[string]interface{}
				err := json.Unmarshal([]byte(v), &tempVal)
				if err == nil {
					result.HistoryRewards = tempVal
				}
			}
		}
	}
	return &result, nil

	//result := &RedPacketModel{}
	//utils.MapToStruct(data, &result)

	//result := RedPacketModel{}
	//err = mapstructure.WeakDecode(data, &result)
	//if err != nil {
	//	return nil, err
	//}
	//return &result, nil

	//values, err := GetMap(
	//	ctx,
	//	BuildInfoKey(redPacketId),
	//	"count", "points", "remainCount", "remainPoints", "bestLuckPoints", "bestLuckIndex", "historyRewards")
	//if err != nil {
	//	return nil, err
	//}
	//count, err := strconv.ParseInt(values[0].(string), 10, 64)
	//if err != nil {
	//	return nil, errors.New("data param error")
	//}
	////points, err := strconv.ParseInt(values[1].(string), 10, 64)
	//points, err := decimal.NewFromString(values[1].(string))
	//if err != nil {
	//	return nil, errors.New("data param error")
	//}
	//remainCount, err := strconv.ParseInt(values[2].(string), 10, 64)
	//if err != nil {
	//	return nil, errors.New("data param error")
	//}
	////remainPoints, err := strconv.ParseInt(values[3].(string), 10, 64)
	//remainPoints, err := decimal.NewFromString(values[3].(string))
	//if err != nil {
	//	return nil, errors.New("data param error")
	//}
	////bestLuckPoints, err := strconv.ParseInt(values[4].(string), 10, 64)
	//
	////historyRewards := make(map[string]interface{})
	//var historyRewards []map[string]interface{}
	//if values[6] == nil {
	//	historyRewards = []map[string]interface{}{}
	//} else {
	//	history := values[6].(string)
	//	err = json.Unmarshal([]byte(history), &historyRewards)
	//	if err != nil {
	//		return nil, errors.New("data param error")
	//	}
	//}
	//
	//return &RedPacketModel{
	//	Count:          int32(count),
	//	Points:         points,
	//	RemainCount:    int32(remainCount),
	//	RemainPoints:   remainPoints,
	//	HistoryRewards: historyRewards,
	//}, nil
}

// ComputerRandPoints 2倍均值法计算红包积分数
func ComputerRandPoints(ctx context.Context, redPacketId string, userId string) (*RedPacketModel, decimal.Decimal, bool, bool) {
	doReplace := false
	minPoints := decimal.NewFromFloat32(RedPacketMinMoney)
	model, err := GetRedPacketModel(ctx, redPacketId)
	if err != nil {
		return nil, decimal.Decimal{}, doReplace, false
	}

	if model.RemainCount == 0 {
		return model, decimal.Decimal{}, doReplace, false
	}

	var tPoints decimal.Decimal
	if model.RemainCount == 1 {
		tPoints = model.RemainPoints
	} else {
		//最大可用金额 = 剩余红包金额 - 后续多少个没拆的包所需要的保底金额
		//目的是为了保证后续的包至少都能分到最低保底金额,避免后续未拆的红包出现金额0
		maxCanUsePoints := model.RemainPoints.Sub(minPoints.Mul(decimal.NewFromInt32(model.RemainCount)))

		//2倍均值基础金额
		maxAvg := maxCanUsePoints.Div(decimal.NewFromInt32(model.RemainCount))

		//2倍均值范围数额
		maxPoints := maxAvg.Mul(decimal.NewFromInt32(2)).Add(minPoints)

		//随机红包数额
		rand.NewSource(time.Now().UnixNano())
		tPoints = minPoints.Add(decimal.NewFromFloat32(rand.Float32()).Round(2).Mul(maxPoints.Sub(minPoints)))
		tPoints = tPoints.Round(2)

		// 判断是否生成固定尾数
		nowIndex := model.Count - model.RemainCount - 1
		set := utils.NewGenericsSet(model.FixedIndex)
		if set.Contains(nowIndex) { // 生成固定尾数
			// 判断是否是白名单用户
			if utils.ElementInSlice(model.WhiteList, userId) {
				replaceIndex := nowIndex
				for {
					replaceIndex++
					if !set.Contains(replaceIndex) && replaceIndex < model.Count {
						// 替换索引
						set.Replace(nowIndex, replaceIndex)
						model.FixedIndex = set.ToSlice()
						// 生成非固定尾数
						tPoints = buildNoFixed(tPoints, model)
						doReplace = true
						break
					} else if replaceIndex < model.Count {
						break
					}
				}
			}

			if !doReplace {
				newPoints, selected := buildFixed(tPoints, model)
				tPoints = newPoints

				model.UsedLastDigits = append(model.UsedLastDigits, selected)
			}
		} else { // 排除固定尾数
			tPoints = buildNoFixed(tPoints, model)
		}
	}

	model.RemainPoints = model.RemainPoints.Sub(tPoints)
	model.RemainCount--
	temp := map[string]interface{}{
		"userId":     userId,
		"points":     tPoints,
		"createTime": time.Now().Format("2006-01-02 15:04:05"),
	}
	model.HistoryRewards = append(model.HistoryRewards, temp)
	return model, tPoints, doReplace, true
}

// 生成固定尾数
func buildFixed(tPoints decimal.Decimal, model *RedPacketModel) (decimal.Decimal, int32) {
	pointStr := tPoints.String()
	selected := model.LastDigits[len(model.LastDigits)-len(model.UsedLastDigits)-1]
	joinStr := pointStr[0 : len(pointStr)-1]
	buildLast := strconv.FormatInt(int64(selected), 32)
	buildPoints, _ := decimal.NewFromString(joinStr + buildLast)
	return buildPoints, selected
}

// 如果是固定尾数替换为随机尾数
func buildNoFixed(tPoints decimal.Decimal, model *RedPacketModel) decimal.Decimal {
	pointStr := tPoints.String()
	lastChar, _ := strconv.ParseInt(string([]rune(pointStr)[len(pointStr)-1]), 10, 32)
	if utils.ElementInSlice(model.LastDigits, int32(lastChar)) {
		joinStr := pointStr[0 : len(pointStr)-1]
		removeRepeatDigits := utils.RemoveSliceRepeat(model.LastDigits)
		noFixedDigits := []int32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		for _, d := range removeRepeatDigits {
			noFixedDigits = utils.RemoveSliceElement(noFixedDigits, d)
		}
		replaceStr := noFixedDigits[rand.Int31n(int32(len(noFixedDigits)))]
		buildPoints, _ := decimal.NewFromString(joinStr + strconv.FormatInt(int64(replaceStr), 32))
		return buildPoints
	}
	return tPoints
}

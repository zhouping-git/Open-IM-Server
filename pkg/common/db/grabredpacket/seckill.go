package grabredpacket

import "fmt"

const TestLua = `
	-- local cjson = require "cjson";
	local cjson2 = cjson.new();
	local lua_string = "I am test1280";
	print(cjson2.encode(lua_string));
	print("test");
	return cjson2.encode(lua_string);
`

const SecKillScript = `
    -- Check if User has redPacket
    -- KEYS[1]: hasUserKey "{redPacketId}-has"
	-- KEYS[2]: redPacketKey    "{redPacketId}-info"
	-- KEYS[3]: grabUserId    "{userId}"
	-- KEYS[4]: grabUserId    "{RedPacketModel}"
    -- 返回值有-1, -2, -3, -4 都代表抢红包失败
    -- 返回值为1, 2代表抢红包成功

	local hasKeys = redis.call("EXISTS", KEYS[1], KEYS[2]);
	if (hasKeys < 2)
	then
		return -4;
	end

    -- Check if the user has got the redPacket --
	local userGrab = redis.call("SISMEMBER", KEYS[1], KEYS[3]);
	if (userGrab == 1)
	then
		return -1;
	end

    -- Check if redPacket exists and is cached
	local remainCount = redis.call("hget", KEYS[2], "remainCount");
	if (remainCount == false)
	then
		return -2;  -- No such redPacket
	end
	if (tonumber(remainCount) == 0)  --- redPacketPoints是字符串类型
    then
		return -3;  --  No RedPacket Points.
	end

	local cjson1 = cjson.new();
	local saveModel = cjson1.decode(KEYS[4]);
	
	-- Is complete to delete
	if (tonumber(saveModel["remainCount"]) == 0)
	then
		redis.call("DEL", KEYS[1], KEYS[2]);
		return 2;
	end
	-- Is not complete to save
	redis.call("SADD", KEYS[1], KEYS[3]);
	redis.call("HSET", KEY[2], saveModel);
	return 1;
`

// 自定义抢红包相关异常
type redisEvalError struct{}

func (e redisEvalError) Error() string {
	return "Error when executing redisService eval."
}

type userHasRedPacketError struct {
	userId      string
	redPacketId string
}

func (e userHasRedPacketError) Error() string {
	return fmt.Sprintf("User %s has had red packet %s.", e.userId, e.redPacketId)
}

type noSuchRedPacketError struct {
	userId      string
	redPacketId string
}

func (e noSuchRedPacketError) Error() string {
	return fmt.Sprintf("Red packet %s created by %s doesn't exist.", e.redPacketId, e.userId)
}

type noRedPacketPointsError struct {
	userId      string
	redPacketId string
}

func (e noRedPacketPointsError) Error() string {
	return fmt.Sprintf("No red packet %s created by %s points.", e.redPacketId, e.userId)
}

type redPacketCompleteError struct {
	userId      string
	redPacketId string
}

func (e redPacketCompleteError) Error() string {
	return fmt.Sprintf("Red packet %s created by %s is complete.", e.redPacketId, e.userId)
}

type redPacketPointsResError struct {
	redPacketPointsRes interface{}
}

func (e redPacketPointsResError) Error() string {
	switch e.redPacketPointsRes.(type) {
	case int32, int:
		return fmt.Sprintf("Unexpected redPacketPointsRes Num: %v.", e.redPacketPointsRes)
	default:
		return fmt.Sprintf("redPacketPointsRes : %v with wrong type.", e.redPacketPointsRes)
	}
}

func IsRedisEvalError(err error) bool {
	switch err.(type) {
	case redisEvalError:
		return true
	default:
		return false
	}
}

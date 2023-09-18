package grabredpacket

import (
	"context"
	"errors"
	"fmt"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/config"
	"github.com/OpenIMSDK/tools/errs"
	"github.com/OpenIMSDK/tools/mw/specialerror"
	"github.com/redis/go-redis/v9"
	"time"
)

var client redis.UniversalClient

const (
	maxRetry = 10 // number of retries
)

func NewRedis() (redis.UniversalClient, error) {
	if len(config.Config.Redis.Address) == 0 {
		return nil, errors.New("redis address is empty")
	}
	specialerror.AddReplace(redis.Nil, errs.ErrRecordNotFound)
	if len(config.Config.Redis.Address) > 1 {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:      config.Config.Redis.Address,
			Username:   config.Config.Redis.Username,
			Password:   config.Config.Redis.Password, // no password set
			PoolSize:   50,
			MaxRetries: maxRetry,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:       config.Config.Redis.Address[0],
			Username:   config.Config.Redis.Username,
			Password:   config.Config.Redis.Password, // no password set
			DB:         0,                            // use default DB
			PoolSize:   100,                          // 连接池大小
			MaxRetries: maxRetry,
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	err := client.Ping(ctx).Err()
	if err != nil {
		return nil, fmt.Errorf("redis ping %w", err)
	}

	return client, nil
}

// PrepareScript 加载lua脚本
func PrepareScript(ctx context.Context, script string) string {
	scriptExists, err := client.ScriptExists(ctx, script).Result()
	if err != nil {
		panic("Failed to check if script exists: " + err.Error())
	}
	if !scriptExists[0] {
		scriptSHA, err := client.ScriptLoad(ctx, script).Result()
		if err != nil {
			panic("Failed to load script " + script + " err: " + err.Error())
		}
		return scriptSHA
	}
	print("Script Exists.")
	return ""
}

// EvalSHA 执行lua脚本
func EvalSHA(ctx context.Context, sha string, args []string) (interface{}, error) {
	val, err := client.EvalSha(ctx, sha, args).Result()
	if err != nil {
		print("Error executing evalSHA... " + err.Error())
		return nil, err
	}
	return val, nil
}

// SetForever redis set
func SetForever(ctx context.Context, key string, value interface{}) (string, error) {
	return client.Set(ctx, key, value, 0).Result()
}

// SetMapForever redis hmset 存储map对象
func SetMapForever(ctx context.Context, key string, field map[string]interface{}) (bool, error) {
	//data, _ := json.Marshal(field)
	//return client.HSet(ctx, key, field).Result()
	return client.HMSet(ctx, key, field).Result()
}

// BatchSetMapForever 批量添加map对象
func BatchSetMapForever(ctx context.Context, fieldMap map[string]map[string]interface{}) (int, error) {
	pipeline := client.Pipeline()
	for key, field := range fieldMap {
		pipeline.HMSet(ctx, key, field)
	}
	r, err := pipeline.Exec(ctx)
	return len(r), err
}

// GetMapAll redis 获取key的map结构
func GetMapAll(ctx context.Context, key string) (map[string]string, error) {
	return client.HGetAll(ctx, key).Result()
}

// GetMap redis hmget 获取map对象的指定属性值
func GetMap(ctx context.Context, key string, fields ...string) ([]interface{}, error) {
	return client.HMGet(ctx, key, fields...).Result()
}

// SetAdd redis sadd
func SetAdd(ctx context.Context, key string, field string) (int64, error) {
	return client.SAdd(ctx, key, field).Result()
}

// SetIsMember redis SIsMember 判断field是否存在
func SetIsMember(ctx context.Context, key string, field string) (bool, error) {
	return client.SIsMember(ctx, key, field).Result()
}

// GetSetMember redis SMembers 根据key获取对象
func GetSetMember(ctx context.Context, key string) ([]string, error) {
	return client.SMembers(ctx, key).Result()
}

func DelKey(ctx context.Context, keys ...string) {
	client.Del(ctx, keys...)
}

// Package dbengine lengdanran 2024/3/28 10:13
package dbengine

import (
	"fmt"
	"github.com/lengdanran/gredis/interface/redis"
	"github.com/lengdanran/gredis/lib/hashmap"
	"github.com/lengdanran/gredis/lib/timewheel"
	"github.com/lengdanran/gredis/redis/protocol"
	"log/slog"
	"runtime/debug"
	"strings"
	"time"
)

/* ---- TTL Functions ---- */

func genExpireTask(key string) string {
	return "expire:" + key
}

type RedisEngine struct {
	DBEngine
	// key -> DataEntity
	Data *hashmap.HashMap
	// key -> expireTime (time.Time)
	TtlMap *hashmap.HashMap
}

func NewRedisEngine() *RedisEngine {
	engine := &RedisEngine{
		Data:   hashmap.NewHashMap(),
		TtlMap: hashmap.NewHashMap(),
	}
	return engine
}

func (engine *RedisEngine) Exec(cmdLine [][]byte) (result redis.Reply) {
	defer func() {
		if err := recover(); err != nil {
			slog.Warn(fmt.Sprintf("error occurs: %v\n%s", err, string(debug.Stack())))
			result = &protocol.UnknownErrReply{}
		}
	}()
	exeName := strings.ToLower(string(cmdLine[0]))
	etr, ok := ExecutorMap[exeName]
	if !ok {
		return protocol.MakeErrReply("ERR unknown command '" + exeName + "'")
	}
	result = etr.ExecF(engine, cmdLine[1:])
	return result
}

// Remove the given key from db
func (engine *RedisEngine) Remove(key string) {
	_ = engine.Data.Del(key)
	engine.TtlMap.Del(key)
	taskKey := genExpireTask(key)
	timewheel.Cancel(taskKey)
}

func (engine *RedisEngine) IsExpired(key string) bool {
	rawExpireTime := engine.TtlMap.Get(key)
	if rawExpireTime == nil {
		return false
	}
	expireTime, _ := rawExpireTime.(time.Time)
	expired := time.Now().After(expireTime)
	if expired {
		engine.Remove(key)
	}
	return expired
}

func (engine *RedisEngine) Get(key string) (*DataEntity, bool) {
	raw := engine.Data.Get(key)
	if raw == nil {
		return nil, false
	}
	if engine.IsExpired(key) {
		return nil, false
	}
	entity, _ := raw.(*DataEntity)
	return entity, true
}

func (engine *RedisEngine) PutEntity(key string, val *DataEntity) {
	engine.Data.Put(hashmap.Entry{Key: key, Value: val})
}

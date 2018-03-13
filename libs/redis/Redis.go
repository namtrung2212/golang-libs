package Redis

import (
	"sync"
	"time"

	Const "iparking/share/const"

	"github.com/go-redis/redis"
)

type RedisClient struct {
	Client        *redis.Client
	ClusterClient *redis.ClusterClient
	Lock          sync.RWMutex
}

func init() {

}

func (this *RedisClient) Connect(config *RedisConfig) error {

	switch config.Type {
	case 1:
		this.NewSentinelClient(config.Sentinel)
	case 2:
		this.NewClusterClient(config.Cluster)
	default:
		this.NewStandaloneClient(config.Standalone)
	}

	if !this.Ping() {
		return Const.ErrRedis_CanNotPing
	}
	return nil
}

func (this *RedisClient) NewStandaloneClient(opts *redis.Options) {

	this.Lock.Lock()
	defer this.Lock.Unlock()

	if opts == nil {
		opts = &redis.Options{}
	}

	if this.Client != nil {
		this.Client.Close()
	}

	this.Client = redis.NewClient(opts)
}

func (this *RedisClient) NewSentinelClient(opt *redis.FailoverOptions) {

	this.Lock.Lock()
	defer this.Lock.Unlock()

	if opt == nil {
		opt = &redis.FailoverOptions{}
	}

	if this.Client != nil {
		this.Client.Close()
	}

	this.Client = redis.NewFailoverClient(opt)
}

func (this *RedisClient) NewClusterClient(opt *redis.ClusterOptions) {

	this.Lock.Lock()
	defer this.Lock.Unlock()

	if opt == nil {
		opt = &redis.ClusterOptions{}
	}

	if this.ClusterClient != nil {
		this.ClusterClient.Close()
	}

	this.ClusterClient = redis.NewClusterClient(opt)
}

func (this *RedisClient) Get(key string) *redis.StringCmd {

	this.Lock.RLock()
	defer this.Lock.RUnlock()

	if this.Client != nil {
		return this.Client.Get(key)
	}

	if this.ClusterClient != nil {
		return this.ClusterClient.Get(key)
	}

	return nil
}

func (this *RedisClient) Set(key string, val interface{}, exp time.Duration) *redis.StatusCmd {

	this.Lock.RLock()
	defer this.Lock.RUnlock()

	if this.Client != nil {
		return this.Client.Set(key, val, exp)
	}

	if this.ClusterClient != nil {
		return this.ClusterClient.Set(key, val, exp)
	}

	return nil
}

func (this *RedisClient) Del(keys ...string) *redis.IntCmd {

	this.Lock.RLock()
	defer this.Lock.RUnlock()

	if this.Client != nil {
		return this.Client.Del(keys...)
	}

	if this.ClusterClient != nil {
		return this.ClusterClient.Del(keys...)
	}

	return nil
}

// ZAdd add members to zsorted list (list defers each other from key)
func (this *RedisClient) ZAdd(key string, Members ...redis.Z) *redis.IntCmd {

	this.Lock.RLock()
	defer this.Lock.RUnlock()

	if this.Client != nil {
		return this.Client.ZAdd(key, Members...)
	}

	if this.ClusterClient != nil {
		return this.ClusterClient.ZAdd(key, Members...)
	}

	return nil
}

// ZRem remove members from zsorted list (list defers each other from key)
func (this *RedisClient) ZRem(key string, Members ...interface{}) *redis.IntCmd {

	this.Lock.RLock()
	defer this.Lock.RUnlock()

	if this.Client != nil {
		return this.Client.ZRem(key, Members...)
	}

	if this.ClusterClient != nil {
		return this.ClusterClient.ZRem(key, Members...)
	}

	return nil
}

// ZRangeWithScore ...
func (this *RedisClient) ZRangeWithScore(key string, start, stop int64) *redis.ZSliceCmd {

	this.Lock.RLock()
	defer this.Lock.RUnlock()

	if this.Client != nil {
		return this.Client.ZRangeWithScores(key, start, stop)
	}

	if this.ClusterClient != nil {
		return this.ClusterClient.ZRangeWithScores(key, start, stop)
	}

	return nil
}

// Ping connection test
func (this *RedisClient) Ping() bool {

	this.Lock.RLock()
	defer this.Lock.RUnlock()

	if this.Client != nil {
		_, err := this.Client.Ping().Result()
		return err == nil
	}

	if this.ClusterClient != nil {
		_, err := this.ClusterClient.Ping().Result()
		return err == nil
	}

	return false
}

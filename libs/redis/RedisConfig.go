package Redis

import (
	"github.com/go-redis/redis"
)

type RedisConfig struct {

	// 0: standalone
	// 1: sentinel
	// 2: cluster
	Type int

	Sentinel *redis.FailoverOptions

	Cluster *redis.ClusterOptions

	Standalone *redis.Options
}

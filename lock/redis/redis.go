package redis

import (
	"time"

	"github.com/boxgo/kit/logger"
	"github.com/boxgo/kit/redis"
)

type (
	// Locker redis lock
	Locker struct {
		*redis.Redis
	}
)

var (
	// Default redis locker
	Default = New("redis")
)

// Lock key锁定
func (l *Locker) Lock(key string, ttl time.Duration) (bool, error) {
	logger.Default.Debugf("redislock.Lock key: %s ttl: %s", key, ttl)
	return l.SetNX(key, time.Now().Unix(), ttl).Result()
}

// IsLocked key是否被锁定
func (l *Locker) IsLocked(key string) (bool, error) {
	result, err := l.Get(key).Result()

	if err != nil {
		if err.Error() == "redis: nil" {
			return false, nil
		}

		return false, err
	}

	if len(result) != 0 {
		return true, nil
	}

	return false, nil
}

// UnLock 解锁key
func (l *Locker) UnLock(key string) error {
	_, err := l.Del(key).Result()
	logger.Default.Debugf("redislock.UnLock key: %s err: %#v", key, err)

	return err
}

// New a redis lock
func New(name string) *Locker {
	return &Locker{
		Redis: redis.New(name),
	}
}

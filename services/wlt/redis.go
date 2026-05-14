package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"time"
)

var rdb *redis.Client

func initRedis(redisURL string) error {
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	rdb = redis.NewClient(&redis.Options{
		Addr: redisURL,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := rdb.Ping(ctx).Err()

	if err != nil {
		return fmt.Errorf("redis connections failed: %v", err)
	}

	logger.Info("Connected to redis", zap.String("addr", redisURL))
	return nil
}

func getBalanceFromCache(walletID int) (*WalletBalanceCache, error) {
	key := fmt.Sprintf("wallet:balance:%d", walletID)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

	defer cancel()

	val, err := rdb.Get(ctx, key).Result()

	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}

		logger.Warn("redis get error", zap.Error(err))
		return nil, err
	}

	var cache WalletBalanceCache
	err = json.Unmarshal([]byte(val), &cache)
	if err != nil {
		logger.Error("failed to unmarshal cache", zap.Error(err))
		return nil, nil
	}

	return &cache, nil
}

func saveBalanceToCache(walletID int, cache *WalletBalanceCache) error {
	key := fmt.Sprintf("wallet:balance:%d", walletID)

	data, err := json.Marshal(cache)
	if err != nil {
		logger.Error("Failed to marshal cache", zap.Error(err))
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = rdb.Set(ctx, key, string(data), 5*time.Minute).Err()
	if err != nil {
		logger.Warn("failed to cache balance", zap.Error(err))
		return nil
	}

	return nil
}

func invalidateWalletCache(walletID int) error {

	key := fmt.Sprintf("wallet:balance:%d", walletID)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := rdb.Del(ctx, key).Err()
	if err != nil {
		logger.Warn("failed to invalidate cache", zap.Error(err))
		return nil
	}

	return nil
}

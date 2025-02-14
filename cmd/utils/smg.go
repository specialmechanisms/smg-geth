package utils

import (
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/urfave/cli/v2"
)

func SetRedisConfig(ctx *cli.Context) {
	if ctx.IsSet(RedisHostNameFlag.Name) {
		filters.DefaultRedisConfig.Host = ctx.String(RedisHostNameFlag.Name)
	}

	if ctx.IsSet(RedisPortFlag.Name) {
		filters.DefaultRedisConfig.Port = ctx.Int(RedisPortFlag.Name)
	}

	if ctx.IsSet(RedisPasswordFlag.Name) {
		filters.DefaultRedisConfig.Password = ctx.String(RedisPasswordFlag.Name)
	}

	if ctx.IsSet(RedisDBFlag.Name) {
		filters.DefaultRedisConfig.DB = ctx.Int(RedisDBFlag.Name)
	}

	if ctx.IsSet(OrderAggregatorServiceFlag.Name) {
		filters.OrderBookAggregatorServiceEnabled = true
	}
}

package grpc

import (
	"math/rand"
	"time"
)

// 这个是backoffStrategy 指 任何一个连接失败后，会一直重试，每次重试的间隔会增加，由此策略决定
//默认是DefaultBackoffConfig， 根据重试次数的不同会从1s到120s递增，每次乘以1.6， 并且会增加一些抖动增加[-0.2,0.2]倍
// DefaultBackoffConfig uses values specified for backoff in
// https://github.com/grpc/grpc/blob/master/doc/connection-backoff.md.
var (
	DefaultBackoffConfig = BackoffConfig{
		MaxDelay:  120 * time.Second,
		baseDelay: 1.0 * time.Second,
		factor:    1.6,
		jitter:    0.2,
	}
)
// backoffStrategy defines the methodology for backing off after a grpc
// connection failure.
//
// This is unexported until the gRPC project decides whether or not to allow
// alternative backoff strategies. Once a decision is made, this type and its
// method may be exported.
//这是小写字母开头的，即未导出的，直到gRPC项目决定是否允许替代退避策略。 一旦做出决定，就可以导出该类型及其方法。
type backoffStrategy interface {
	// backoff returns the amount of time to wait before the next retry given
	// the number of consecutive failures.
	//retries是连续失败的次数，返回 这次失败后下次重试等待的间隔
	backoff(retries int) time.Duration
}

// BackoffConfig defines the parameters for the default gRPC backoff strategy.
type BackoffConfig struct {
	// MaxDelay is the upper bound of backoff delay.
	MaxDelay time.Duration

	// TODO(stevvooe): The following fields are not exported, as allowing
	// changes would violate the current gRPC specification for backoff. If
	// gRPC decides to allow more interesting backoff strategies, these fields
	// may be opened up in the future.

	// baseDelay is the amount of time to wait before retrying after the first
	// failure.
	baseDelay time.Duration

	// factor is applied to the backoff after each retry.
	//每次重试间隔都会乘以factor  递增
	factor float64

	// jitter provides a range to randomize backoff delays.
	//计算出间隔时间后，加上一些抖动时间,防止多个链接重试的时候都跑到一起
	jitter float64
}

func setDefaults(bc *BackoffConfig) {
	md := bc.MaxDelay
	*bc = DefaultBackoffConfig

	if md > 0 {
		bc.MaxDelay = md
	}
}

func (bc BackoffConfig) backoff(retries int) time.Duration {
	if retries == 0 {
		return bc.baseDelay//第一次重试的间隔是baseDelay
	}
	//计算公式backoff=baseDelay*pow(factor,retries)
	backoff, max := float64(bc.baseDelay), float64(bc.MaxDelay)
	for backoff < max && retries > 0 {
		backoff *= bc.factor
		retries--
	}
	if backoff > max {
		backoff = max //重试的最大间隔是MaxDelay
	}
	// Randomize backoff delays so that if a cluster of requests start at
	// the same time, they won't operate in lockstep.
	backoff *= 1 + bc.jitter*(rand.Float64()*2-1)
	if backoff < 0 {
		return 0
	}
	return time.Duration(backoff)
}

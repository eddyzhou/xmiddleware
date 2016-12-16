package interceptor

import (
	"sync"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var (
	rateLimiter         *RateLimiter
	initRateLimiterOnce sync.Once
)

// Simple and trivial rate limiter.
type RateLimiter struct {
	Interval time.Duration
	MaxCount uint

	mu       sync.Mutex
	count    uint
	lastTick time.Time
}

func (r RateLimiter) Allowed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if time.Since(r.lastTick) > r.Interval {
		r.lastTick = time.Now()
		r.count = 0
		return true
	}

	r.count++
	return r.count < r.MaxCount
}

func InitRateLimiter(rl RateLimiter) {
	initRateLimiterOnce.Do(func() {
		rateLimiter = &rl
	})
}

func RateLimit(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	if !rateLimiter.Allowed() {
		return nil, grpc.Errorf(codes.Internal, "rate limited")
	}
	resp, err = handler(ctx, req)
	return resp, err
}

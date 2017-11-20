package interceptor

import (
	"time"

	"github.com/eddyzhou/ratelimit"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type RateLimiter struct {
	bucket *ratelimit.Bucket
}

func NewRateLimiter(fillInterval time.Duration, capacity int64, quantum int64) *RateLimiter {
	if capacity < 1 {
		panic("xmiddleware/ratelimit: capacity expects to be positive")
	}
	if quantum < 1 {
		panic("xmiddleware/ratelimit: quantum expects to be positive")
	}

	return &RateLimiter{
		bucket: ratelimit.NewBucketWithQuantum(fillInterval, capacity, quantum),
	}
}

func (r *RateLimiter) RateLimit(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	t := r.bucket.Take(1)
	if t <= 0 {
		resp, err = handler(ctx, req)
		return resp, err
	}

	select {
	case <-time.After(t):
		resp, err = handler(ctx, req)
		return resp, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

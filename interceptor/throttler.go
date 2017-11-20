package interceptor

import (
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type token struct{}

type Throttler struct {
	tokens         chan token
	backlogTokens  chan token
	backlogTimeout time.Duration
}

func NewThrottler(limit int, backlogLimit int, backlogTimeout time.Duration) *Throttler {
	if limit < 1 {
		panic("xmiddleware/throttler: Throttle expects limit > 0")
	}

	if backlogLimit < 0 {
		panic("xmiddleware/throttler: Throttle expects backlogLimit to be positive")
	}

	t := Throttler{
		tokens:         make(chan token, limit),
		backlogTokens:  make(chan token, limit+backlogLimit),
		backlogTimeout: backlogTimeout,
	}

	// Filling tokens.
	for i := 0; i < limit+backlogLimit; i++ {
		if i < limit {
			t.tokens <- token{}
		}
		t.backlogTokens <- token{}
	}

	return &t
}

func (t *Throttler) Throttle(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	select {
	case btok := <-t.backlogTokens:
		timer := time.NewTimer(t.backlogTimeout)

		defer func() {
			t.backlogTokens <- btok
		}()

		select {
		case <-timer.C:
			return nil, grpc.Errorf(codes.ResourceExhausted, "Concurrent RPC limit exceeded")
		case <-ctx.Done():
			return nil, ctx.Err()
		case tok := <-t.tokens:
			defer func() {
				t.tokens <- tok
			}()

			resp, err = handler(ctx, req)
			return resp, err
		}

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

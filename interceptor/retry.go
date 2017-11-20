package interceptor

import (
	"fmt"
	"math/rand"
	"time"

	"xlbj-gitlab.xunlei.cn/shoulei-service/xmiddleware/utils"

	"github.com/eddyzhou/log"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const (
	AttemptMetadataKey = "x-retry-attempty"
)

var (
	DefaultRetriableCodes = []codes.Code{codes.ResourceExhausted, codes.Unavailable}

	defaultOptions = &retryOptions{
		max:            0, // disabed
		perCallTimeout: 0, // disabled
		includeHeader:  true,
		codes:          DefaultRetriableCodes,
		backoffFunc:    BackoffUniformRandom(50*time.Millisecond, 0.10),
	}
)

type BackoffFunc func(attempt uint) time.Duration

type retryOptions struct {
	max            uint
	perCallTimeout time.Duration
	includeHeader  bool
	codes          []codes.Code
	backoffFunc    BackoffFunc
}

type CallOption struct {
	grpc.CallOption
	applyFunc func(opt *retryOptions)
}

func WithRetryDisable() CallOption {
	return WithRetryMax(0)
}

func WithRetryMax(maxRetries uint) CallOption {
	return CallOption{applyFunc: func(o *retryOptions) {
		o.max = maxRetries
	}}
}

func WithRetryBackoff(bf BackoffFunc) CallOption {
	return CallOption{applyFunc: func(o *retryOptions) {
		o.backoffFunc = bf
	}}
}

func WithRetryCodes(retryCodes ...codes.Code) CallOption {
	return CallOption{applyFunc: func(o *retryOptions) {
		o.codes = retryCodes
	}}
}

func WithPerRetryTimeout(timeout time.Duration) CallOption {
	return CallOption{applyFunc: func(o *retryOptions) {
		o.perCallTimeout = timeout
	}}
}

func BackoffUniformRandom(backoff time.Duration, jitter float64) BackoffFunc {
	return func(attempt uint) time.Duration {
		multiplier := jitter * (rand.Float64() - 0.5) * 2
		return time.Duration(float64(backoff) * (1 + multiplier))
	}
}

func mergeCallOptions(opt *retryOptions, callOptions ...CallOption) *retryOptions {
	if len(callOptions) == 0 {
		return opt
	}
	optCopy := &retryOptions{}
	*optCopy = *opt
	for _, f := range callOptions {
		f.applyFunc(optCopy)
	}
	return optCopy
}

func splitCallOptions(callOptions []grpc.CallOption) (grpcOptions []grpc.CallOption, retryOptions []CallOption) {
	for _, opt := range callOptions {
		if co, ok := opt.(CallOption); ok {
			retryOptions = append(retryOptions, co)
		} else {
			grpcOptions = append(grpcOptions, opt)
		}
	}
	return grpcOptions, retryOptions
}

// -----------------

func UnaryClientRetry(optFuncs ...CallOption) grpc.UnaryClientInterceptor {
	initOpts := mergeCallOptions(defaultOptions, optFuncs...)
	return func(parentCtx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		grpcOpts, retryOpts := splitCallOptions(opts)
		callOpts := mergeCallOptions(initOpts, retryOpts...)
		if callOpts.max == 0 {
			return invoker(parentCtx, method, req, reply, cc, grpcOpts...)
		}

		var lastErr error
		for attempt := uint(0); attempt < callOpts.max; attempt++ {
			if err := waitRetryBackoff(attempt, parentCtx, callOpts); err != nil {
				return err
			}

			callCtx := perCallContext(parentCtx, callOpts, attempt)
			lastErr = invoker(callCtx, method, req, reply, cc, grpcOpts...)
			if lastErr == nil {
				return nil
			}

			log.Warnf("gRPC retry attempt: %d, err: %v", attempt, lastErr)
			if isContextError(lastErr) {
				if parentCtx.Err() != nil {
					log.Warnf("gRPC retry attempt: %d, parent context error: %v", attempt, parentCtx.Err())
					// its the parent context deadline or cancellation.
					return lastErr
				} else {
					log.Warnf("gRPC retry attempt: %d, context error from retry call", attempt)
					// its the callCtx deadline or cancellation, in which case try again.
					continue
				}
			}
			if !isRetriable(lastErr, callOpts) {
				return lastErr
			}
		}
		return lastErr
	}
}

func isRetriable(err error, callOpts *retryOptions) bool {
	errCode := grpc.Code(err)
	if isContextError(err) {
		return false
	}
	for _, code := range callOpts.codes {
		if code == errCode {
			return true
		}
	}
	return false
}

func waitRetryBackoff(attempt uint, parentCtx context.Context, callOpts *retryOptions) error {
	var waitTime time.Duration = 0
	if attempt > 0 {
		waitTime = callOpts.backoffFunc(attempt)
	}
	if waitTime > 0 {
		log.Infof("gRPC retry attempt: %d, backoff for %v", attempt, waitTime)
		select {
		case <-parentCtx.Done():
			return convToGrpcErr(parentCtx.Err())
		case <-time.Tick(waitTime):
		}
	}
	return nil
}

func isContextError(err error) bool {
	return grpc.Code(err) == codes.DeadlineExceeded || grpc.Code(err) == codes.Canceled
}

func perCallContext(parentCtx context.Context, callOpts *retryOptions, attempt uint) context.Context {
	ctx := parentCtx
	if callOpts.perCallTimeout != 0 {
		ctx, _ = context.WithTimeout(ctx, callOpts.perCallTimeout)
	}
	if attempt > 0 && callOpts.includeHeader {
		ctx = utils.Set(ctx, AttemptMetadataKey, fmt.Sprintf("%d", attempt))
	}
	return ctx
}

func convToGrpcErr(err error) error {
	switch err {
	case context.DeadlineExceeded:
		return grpc.Errorf(codes.DeadlineExceeded, err.Error())
	case context.Canceled:
		return grpc.Errorf(codes.Canceled, err.Error())
	default:
		return grpc.Errorf(codes.Unknown, err.Error())
	}
}

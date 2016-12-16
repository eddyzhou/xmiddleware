package interceptor

import (
	"errors"
	"fmt"
	"log"
	"runtime"
	"runtime/debug"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const (
	MAXSTACKSIZE = 4096
)

// Recovery interceptor to handle grpc panic
func Recovery(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	// recovery func
	defer func() {
		if r := recover(); r != nil {
			// log stack
			stack := make([]byte, MAXSTACKSIZE)
			stack = stack[:runtime.Stack(stack, false)]
			errStack := string(stack)
			log.Printf("panic grpc invoke: %s, err=%v, stack:\n%s", info.FullMethod, r, errStack)

			if monitor != nil {
				// report to sentry
				func() {
					defer func() {
						if r := recover(); err != nil {
							log.Printf("report sentry failed: %s, trace:\n%s", r, debug.Stack())
							err = grpc.Errorf(codes.Internal, "panic error: %v", r)
						}
					}()
					switch rval := r.(type) {
					case error:
						monitor.ObserveError(info.FullMethod, rval)
					default:
						monitor.ObserveError(info.FullMethod, errors.New(fmt.Sprint(rval)))
					}
				}()
			}

			// if panic, set custom error to 'err', in order that client and sense it.
			err = grpc.Errorf(codes.Internal, "panic error: %v", r)
		}
	}()

	return handler(ctx, req)
}

package interceptor

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// UnaryServerInterceptorChain build the multi interceptors into one interceptor chain.
func UnaryServerChain(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			chain = buildServerChain(interceptors[i], chain, info)
		}
		return chain(ctx, req)
	}
}

// build is the interceptor chain helper
func buildServerChain(c grpc.UnaryServerInterceptor, n grpc.UnaryHandler, info *grpc.UnaryServerInfo) grpc.UnaryHandler {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		return c(ctx, req, info, n)
	}
}

func WithUnaryServerInterceptors(interceptors ...grpc.UnaryServerInterceptor) grpc.ServerOption {
	return grpc.UnaryInterceptor(UnaryServerChain(interceptors...))
}

// -------------

func UnaryClientChain(interceptors ...grpc.UnaryClientInterceptor) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		chain := invoker
		for i := len(interceptors) - 1; i >= 0; i-- {
			chain = buildClientChain(interceptors[i], chain)
		}
		return chain(ctx, method, req, reply, cc, opts...)
	}
}

func buildClientChain(current grpc.UnaryClientInterceptor, next grpc.UnaryInvoker) grpc.UnaryInvoker {
	return func(currentCtx context.Context, currentMethod string, currentReq, currentRepl interface{}, currentConn *grpc.ClientConn, currentOpts ...grpc.CallOption) error {
		return current(currentCtx, currentMethod, currentReq, currentRepl, currentConn, next, currentOpts...)
	}
}

func WithUnaryClientInterceptors(interceptors ...grpc.UnaryClientInterceptor) grpc.DialOption {
	return grpc.WithUnaryInterceptor(UnaryClientChain(interceptors...))
}

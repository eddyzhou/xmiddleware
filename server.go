package xmiddleware

import (
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"

	"xmiddleware/interceptor"
)

func NewServer(sos ...XServerOption) *grpc.Server {
	opt := &options{}
	for _, o := range sos {
		o(opt)
	}

	if opt.initMonitor && opt.initRateLimit {
		interceptor.InitMonitor(opt.application, opt.port, opt.sentryDSN)
		interceptor.InitRateLimiter(opt.rateLimiter)
		return grpc.NewServer(grpc.UnaryInterceptor(
			interceptor.UnaryInterceptorChain(interceptor.Recovery, interceptor.Monitoring, interceptor.Logging, interceptor.RateLimit)))
	} else if opt.initMonitor {
		interceptor.InitMonitor(opt.application, opt.port, opt.sentryDSN)
		return grpc.NewServer(grpc.UnaryInterceptor(
			interceptor.UnaryInterceptorChain(interceptor.Recovery, interceptor.Monitoring, interceptor.Logging)))
	} else if opt.initRateLimit {
		interceptor.InitRateLimiter(opt.rateLimiter)
		return grpc.NewServer(grpc.UnaryInterceptor(
			interceptor.UnaryInterceptorChain(interceptor.Recovery, interceptor.Logging, interceptor.RateLimit)))
	} else {
		return grpc.NewServer(grpc.UnaryInterceptor(
			interceptor.UnaryInterceptorChain(interceptor.Recovery, interceptor.Logging)))
	}
}

func StartMetricsServer(metricsPort int) {
	go func() {
		http.Handle("/metrics", prometheus.Handler())
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", metricsPort), nil))
	}()
}

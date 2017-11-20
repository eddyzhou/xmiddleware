package xmiddleware

import (
	"fmt"
	"net/http"

	"github.com/eddyzhou/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	"xlbj-gitlab.xunlei.cn/shoulei-service/xmiddleware/interceptor"
)

func NewServer(sos ...XServerOption) *grpc.Server {
	opt := &options{}
	for _, o := range sos {
		o(opt)
	}

	chain := interceptor.UnaryServerChain(interceptor.Logging)

	if opt.tc != nil {
		tc := opt.tc
		t := interceptor.NewThrottler(tc.limit, tc.backlogLimit, tc.backlogTimeout)
		chain = interceptor.UnaryServerChain(t.Throttle, chain)
	}
	if opt.rc != nil {
		rc := opt.rc
		rl := interceptor.NewRateLimiter(rc.fillInterval, rc.capacity, rc.quantum)
		chain = interceptor.UnaryServerChain(rl.RateLimit, chain)
	}
	if opt.mc != nil {
		mc := opt.mc
		m, err := interceptor.NewMonitor(mc.application, mc.port, mc.sentryDSN)
		if err != nil {
			panic(err)
		}
		chain = interceptor.UnaryServerChain(m.Recovery, m.Monitoring, chain)
	}

	return grpc.NewServer(grpc.UnaryInterceptor(chain))
}

func StartMetricsServer(metricsPort int) {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", metricsPort), nil))
	}()
}

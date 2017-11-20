package interceptor

import (
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/eddyzhou/log"
	"github.com/getsentry/raven-go"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const (
	MAXSTACKSIZE = 4096
)

var (
	defaultBuckets = []float64{10, 20, 30, 50, 80, 100, 200, 300, 500, 1000, 2000, 3000}
)

type Monitor struct {
	sentryClient *raven.Client
	reqCounter   *prometheus.CounterVec
	errCounter   *prometheus.CounterVec
	respLatency  *prometheus.HistogramVec
}

func NewMonitor(application string, port int, sentryDSN string, buckets ...float64) (*Monitor, error) {
	var m Monitor
	client, err := raven.New(sentryDSN)
	if err != nil {
		log.Error("Monitor: init failed: ", err.Error())
		return nil, err
	}
	m.sentryClient = client

	process := strconv.Itoa(port)
	m.reqCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   application,
			Name:        "requests_total",
			Help:        "Total request counts",
			ConstLabels: prometheus.Labels{"method": "rpc", "process": process},
		},
		[]string{
			"endpoint",
		},
	)
	prometheus.MustRegister(m.reqCounter)

	m.errCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   application,
			Name:        "error_total",
			Help:        "Total error counts",
			ConstLabels: prometheus.Labels{"method": "rpc", "process": process},
		},
		[]string{
			"endpoint",
		},
	)
	prometheus.MustRegister(m.errCounter)

	if len(buckets) == 0 {
		buckets = defaultBuckets
	}
	m.respLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:   application,
			Name:        "response_latency_millisecond",
			Help:        "Response latency (millisecond)",
			ConstLabels: prometheus.Labels{"method": "rpc", "process": process},
			Buckets:     buckets,
		},
		[]string{
			"endpoint",
		},
	)
	prometheus.MustRegister(m.respLatency)

	return &m, nil
}

func (m *Monitor) Observe(method string, latency float64) {
	labels := prometheus.Labels{"endpoint": method}
	m.reqCounter.With(labels).Inc()
	m.respLatency.With(labels).Observe(latency)
}

func (m *Monitor) ObserveError(method string, err error) {
	labels := prometheus.Labels{"endpoint": method}
	m.errCounter.With(labels).Inc()

	m.sentryClient.CaptureError(err, nil)
}

func (m *Monitor) Monitoring(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	start := time.Now()
	resp, err = handler(ctx, req)
	if err == nil {
		m.Observe(info.FullMethod, float64(time.Since(start).Nanoseconds())/1000000)
	} else {
		m.ObserveError(info.FullMethod, err)
	}

	return resp, err
}

// Recovery interceptor to handle grpc panic
func (m *Monitor) Recovery(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	// recovery func
	defer func() {
		if r := recover(); r != nil {
			// log stack
			stack := make([]byte, MAXSTACKSIZE)
			stack = stack[:runtime.Stack(stack, false)]
			errStack := string(stack)
			log.Errorf("panic grpc invoke: %s, err=%v, stack:\n%s", info.FullMethod, r, errStack)

			func() {
				defer func() {
					if r := recover(); err != nil {
						log.Errorf("report sentry failed: %s, trace:\n%s", r, debug.Stack())
						err = grpc.Errorf(codes.Internal, "panic error: %v", r)
					}
				}()
				switch rval := r.(type) {
				case error:
					m.ObserveError(info.FullMethod, rval)
				default:
					m.ObserveError(info.FullMethod, errors.New(fmt.Sprint(rval)))
				}
			}()

			// if panic, set custom error to 'err', in order that client and sense it.
			err = grpc.Errorf(codes.Internal, "panic error: %v", r)
		}
	}()

	return handler(ctx, req)
}

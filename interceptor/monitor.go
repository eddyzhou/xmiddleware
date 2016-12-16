package interceptor

import (
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/getsentry/raven-go"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	monitor         *Monitor
	initMonitorOnce sync.Once
	defaultBuckets  = []float64{10, 20, 30, 50, 80, 100, 200, 300, 500, 1000, 2000, 3000}
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
		log.Fatal(err)
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

// --------------------

func InitMonitor(application string, port int, sentryDSN string) {
	initMonitorOnce.Do(func() {
		if m, err := NewMonitor(application, port, sentryDSN); err == nil {
			monitor = m
		} else {
			panic(err)
		}
	})
}

func Monitoring(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	start := time.Now()
	resp, err = handler(ctx, req)
	if err == nil {
		monitor.Observe(info.FullMethod, float64(time.Since(start).Nanoseconds())/1000000)
	} else {
		monitor.ObserveError(info.FullMethod, err)
	}

	return resp, err
}

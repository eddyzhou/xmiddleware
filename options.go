package xmiddleware

import (
	"time"
)

type options struct {
	mc *monitorConf
	rc *rateLimitConf
	tc *throttlerConf
}

type monitorConf struct {
	application string
	port        int
	sentryDSN   string
}

type rateLimitConf struct {
	fillInterval time.Duration
	capacity     int64
	quantum      int64
}

type throttlerConf struct {
	limit          int
	backlogLimit   int
	backlogTimeout time.Duration
}

type XServerOption func(*options)

func Monitor(application string, port int, sentryDSN string) XServerOption {
	return func(o *options) {
		o.mc = &monitorConf{
			application: application,
			port:        port,
			sentryDSN:   sentryDSN,
		}
	}
}

// quantum tokens are added every fillInterval, up to the given maximum capacity
func RateLimit(fillInterval time.Duration, capacity int64, quantum int64) XServerOption {
	return func(o *options) {
		o.rc = &rateLimitConf{
			fillInterval: fillInterval,
			capacity:     capacity,
			quantum:      quantum,
		}
	}
}

func Throttler(limit int, backlogLimit int, backlogTimeout time.Duration) XServerOption {
	return func(o *options) {
		o.tc = &throttlerConf{
			limit:          limit,
			backlogLimit:   backlogLimit,
			backlogTimeout: backlogTimeout,
		}
	}
}

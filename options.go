package xmiddleware

import (
	"time"
	"xmiddleware/interceptor"
)

type options struct {
	application string
	port        int
	sentryDSN   string
	initMonitor bool

	rateLimiter   interceptor.RateLimiter
	initRateLimit bool
}

type XServerOption func(*options)

func Monitor(application string, port int, sentryDSN string) XServerOption {
	return func(o *options) {
		o.application = application
		o.port = port
		o.sentryDSN = sentryDSN
		o.initMonitor = true
	}
}

func RateLimit(interval time.Duration, maxCount uint) XServerOption {
	return func(o *options) {
		o.rateLimiter = interceptor.RateLimiter{
			Interval: interval,
			MaxCount: maxCount,
		}
		o.initRateLimit = true
	}
}

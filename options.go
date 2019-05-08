package schedule

import "github.com/boxgo/schedule/lock"

type (
	// Option option set func
	Option func(opts *Options)

	// Options items
	Options struct {
		locker        lock.Lock
		onceHandler   Handler
		timingHandler Handler
	}
)

// WithLocker set locker
func WithLocker(locker lock.Lock) Option {
	return func(op *Options) {
		op.locker = locker
	}
}

// WithOnceHandler set onceHandler
func WithOnceHandler(handler Handler) Option {
	return func(op *Options) {
		op.onceHandler = handler
	}
}

// WithTimingHandler set timingHandler
func WithTimingHandler(handler Handler) Option {
	return func(op *Options) {
		op.timingHandler = handler
	}
}

package retry

import (
	"errors"
	"time"
)

var (
	ErrStop = errors.New("stop retries")
)

type RetryableFunc func() error

type Options struct {
	Delayer Delayer
	Stopper Stopper
}

func Do(fn RetryableFunc, opts Options) error {
	if fn == nil {
		return nil
	}

	startTime := time.Now()
	attempts := 0
	for {
		err := fn()
		if err == nil || err == ErrStop || opts.Stopper == nil || opts.Delayer == nil {
			return err
		}

		attempts += 1
		if opts.Stopper.Stop(startTime, attempts, err) {
			return err
		}

		d := opts.Delayer.Delay(startTime, attempts, err)
		if d.Nanoseconds() > 0 {
			time.Sleep(d)
		}
	}
}

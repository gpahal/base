package retry

import (
	"errors"
	"time"
)

var (
	ErrStop = errors.New("stop retries")
)

type RetryableFunc func() error

type Retryable struct {
	Fn RetryableFunc
	Delayer Delayer
	Stopper Stopper
}

func Do(r *Retryable) []error {
	if r == nil || r.Fn == nil {
		return nil
	}

	startTime := time.Now()
	var errs []error
	attempts := 0
	for {
		err := r.Fn()
		if err == nil || err == ErrStop {
			return errs
		}
		errs = append(errs, err)

		attempts += 1
		if r.Stopper != nil && r.Stopper.Stop(startTime, attempts, err) {
			return errs
		}

		if r.Delayer == nil {
			continue
		}
		d := r.Delayer.Delay(startTime, attempts, err)
		if d.Nanoseconds() > 0 {
			time.Sleep(d)
		}
	}
}

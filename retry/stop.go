package retry

import (
	"time"
)

type Stopper interface {
	Stop(startTime time.Time, attempts int, err error) bool
}

type StopperFunc func(startTime time.Time, attempts int, err error) bool

func (sf StopperFunc) Stop(startTime time.Time, attempts int, err error) bool {
	return sf(startTime, attempts, err)
}

func MaxAttemptsStopper(maxAttempts int) Stopper {
	return StopperFunc(func(startTime time.Time, attempts int, err error) bool {
		return attempts >= maxAttempts
	})
}

func TimeoutStopper(d time.Duration) Stopper {
	return StopperFunc(func(startTime time.Time, attempts int, err error) bool {
		return time.Now().After(startTime.Add(d))
	})
}

func DeadlineStopper(deadline time.Time) Stopper {
	return StopperFunc(func(startTime time.Time, attempts int, err error) bool {
		return time.Now().After(deadline)
	})
}

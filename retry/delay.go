package retry

import (
	"math"
	"time"

	"github.com/gpahal/golib/random"
)

const (
	// 1 << 63 would overflow signed int64 (time.Duration), thus 62
	maxExp = 62
)

type Delayer interface {
	Delay(startTime time.Time, attempts int, err error) time.Duration
}

type DelayerFunc func(startTime time.Time, attempts int, err error) time.Duration

func (df DelayerFunc) Delay(startTime time.Time, attempts int, err error) time.Duration {
	return df(startTime, attempts, err)
}

func FixedDelayer(d time.Duration) Delayer {
	return DelayerFunc(func(startTime time.Time, attempts int, err error) time.Duration {
		return d
	})
}

func LinearDelayer(step time.Duration) Delayer {
	return DelayerFunc(func(startTime time.Time, attempts int, err error) time.Duration {
		return time.Duration(attempts) * step
	})
}

func ExponentialBackoffDelayer(coefficient int) Delayer {
	if coefficient <= 0 {
		return nil
	}

	currMaxExp := maxExp - int(math.Floor(math.Log2(float64(coefficient))))
	return DelayerFunc(func(startTime time.Time, attempts int, err error) time.Duration {
		if attempts > currMaxExp {
			attempts = currMaxExp
		}
		return time.Duration(coefficient * (1 << attempts))
	})
}

func RandomDelayer(minDelay time.Duration, maxJitter time.Duration) Delayer {
	rnd := random.New()
	return DelayerFunc(func(startTime time.Time, attempts int, err error) time.Duration {
		return max(minDelay, 0) + time.Duration(rnd.Int64n(int64(maxJitter)))
	})
}

func LimitDelayer(inner Delayer, limit time.Duration) Delayer {
	if inner == nil {
		return nil
	}

	return DelayerFunc(func(startTime time.Time, attempts int, err error) time.Duration {
		d := inner.Delay(startTime, attempts, err)
		if d > limit {
			d = limit
		}
		return d
	})
}

func MinDelayer(delayers ...Delayer) Delayer {
	return CombineDelayers(func(ds []time.Duration) time.Duration {
		var min time.Duration
		for _, d := range ds {
			if d < min {
				min = d
			}
		}
		return min
	}, delayers...)
}

func MaxDelayer(delayers ...Delayer) Delayer {
	return CombineDelayers(func(ds []time.Duration) time.Duration {
		var max time.Duration
		for _, d := range ds {
			if d > max {
				max = d
			}
		}
		return max
	}, delayers...)
}

func SumDelayer(delayers ...Delayer) Delayer {
	return CombineDelayers(func(ds []time.Duration) time.Duration {
		var sum time.Duration
		for _, d := range ds {
			sum += d
		}
		return sum
	}, delayers...)
}

func CombineDelayers(combine func(ds []time.Duration) time.Duration, delayers ...Delayer) Delayer {
	if len(delayers) == 0 {
		return nil
	}

	return DelayerFunc(func(startTime time.Time, attempts int, err error) time.Duration {
		if len(delayers) == 0 {
			return 0
		}

		var ds []time.Duration
		for _, delayer := range delayers {
			if delayer == nil {
				ds = append(ds, 0)
				continue
			}
			ds = append(ds, delayer.Delay(startTime, attempts, err))
		}
		return combine(ds)
	})
}

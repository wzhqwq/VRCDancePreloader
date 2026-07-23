package utils

import (
	"context"
	"math/rand"
	"time"
)

type RetryPolicy struct {
	MaxRetries int

	Delay time.Duration

	Jitter bool
}

func Retry[T any](ctx context.Context, policy *RetryPolicy, fn func(context.Context, int) (T, bool, error)) (T, error) {
	var (
		result   T
		canRetry bool
		err      error
	)

	if policy == nil {
		result, canRetry, err = fn(ctx, 1)
		return result, err
	}

	for attempt := 0; ; attempt++ {
		result, canRetry, err = fn(ctx, attempt)
		if err == nil {
			return result, nil
		}

		if attempt >= policy.MaxRetries || !canRetry {
			return result, err
		}

		wait := policy.Delay

		if policy.Jitter {
			wait = jitter(wait)
		}

		timer := time.NewTimer(wait)

		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return result, ctx.Err()
		}
	}
}

func jitter(d time.Duration) time.Duration {
	return time.Duration(
		float64(d) * (0.5 + rand.Float64()),
	)
}

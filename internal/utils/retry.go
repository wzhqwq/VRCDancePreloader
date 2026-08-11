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

type RetryFetchFn[T any] func(context.Context, int, time.Duration) (T, bool, error)

func Retry[T any](ctx context.Context, policy RetryPolicy, fn RetryFetchFn[T]) (T, error) {
	var (
		result   T
		canRetry bool
		err      error
	)

	for attempt := 0; ; attempt++ {
		wait := policy.Delay

		if policy.Jitter {
			wait = jitter(wait)
		}

		result, canRetry, err = fn(ctx, attempt, wait)
		if err == nil {
			return result, nil
		}

		if attempt >= policy.MaxRetries || !canRetry {
			return result, err
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

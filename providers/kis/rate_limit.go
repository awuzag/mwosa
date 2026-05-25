package kis

import (
	"context"
	"math"
	"strconv"
	"sync"
	"time"

	kisclient "github.com/ev3rlit/mwosa/clients/kis"
	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/samber/oops"
)

const defaultKISReadRPS = 15

type RateLimitPolicy struct {
	ReadRPS     float64
	Burst       int
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Jitter      time.Duration
}

type ReadRateLimiter interface {
	Wait(context.Context, kisclient.RateLimitRequest) error
}

type rateLimitSleeper func(context.Context, time.Duration) error

func defaultRateLimitPolicy() RateLimitPolicy {
	return RateLimitPolicy{
		ReadRPS:     defaultKISReadRPS,
		Burst:       1,
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    500 * time.Millisecond,
	}
}

func (p RateLimitPolicy) normalized() RateLimitPolicy {
	defaults := defaultRateLimitPolicy()
	if p.ReadRPS <= 0 {
		p.ReadRPS = defaults.ReadRPS
	}
	if p.Burst <= 0 {
		p.Burst = defaults.Burst
	}
	if p.MaxAttempts <= 0 {
		p.MaxAttempts = defaults.MaxAttempts
	}
	if p.BaseDelay <= 0 {
		p.BaseDelay = defaults.BaseDelay
	}
	if p.MaxDelay <= 0 {
		p.MaxDelay = defaults.MaxDelay
	}
	if p.MaxDelay < p.BaseDelay {
		p.MaxDelay = p.BaseDelay
	}
	if p.Jitter < 0 {
		p.Jitter = 0
	}
	return p
}

func (p RateLimitPolicy) isZero() bool {
	return p.ReadRPS == 0 &&
		p.Burst == 0 &&
		p.MaxAttempts == 0 &&
		p.BaseDelay == 0 &&
		p.MaxDelay == 0 &&
		p.Jitter == 0
}

func (p RateLimitPolicy) limitLabel() string {
	p = p.normalized()
	return strconv.FormatFloat(p.ReadRPS, 'f', -1, 64) + " rps"
}

type tokenBucketReadLimiter struct {
	mu       sync.Mutex
	policy   RateLimitPolicy
	now      func() time.Time
	sleep    rateLimitSleeper
	tokens   float64
	lastSeen time.Time
}

func newTokenBucketReadLimiter(policy RateLimitPolicy, now func() time.Time, sleep rateLimitSleeper) *tokenBucketReadLimiter {
	policy = policy.normalized()
	if now == nil {
		now = time.Now
	}
	if sleep == nil {
		sleep = sleepContext
	}
	return &tokenBucketReadLimiter{
		policy:   policy,
		now:      now,
		sleep:    sleep,
		tokens:   float64(policy.Burst),
		lastSeen: now(),
	}
}

func (l *tokenBucketReadLimiter) Wait(ctx context.Context, _ kisclient.RateLimitRequest) error {
	if l == nil {
		return nil
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		wait := l.reserveDelay()
		if wait <= 0 {
			return nil
		}
		if err := l.sleep(ctx, wait); err != nil {
			return err
		}
	}
}

func (l *tokenBucketReadLimiter) reserveDelay() time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	elapsed := now.Sub(l.lastSeen).Seconds()
	if elapsed > 0 {
		l.tokens = math.Min(float64(l.policy.Burst), l.tokens+elapsed*l.policy.ReadRPS)
		l.lastSeen = now
	}
	if l.tokens >= 1 {
		l.tokens--
		return 0
	}
	missing := 1 - l.tokens
	return time.Duration((missing / l.policy.ReadRPS) * float64(time.Second))
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func withReadRetry[T any](ctx context.Context, p *Provider, request kisclient.RateLimitRequest, call func(context.Context) (T, error)) (T, error) {
	var zero T
	if p == nil {
		return zero, oops.In("kis_adapter").New("kis provider is nil")
	}
	policy := p.rateLimitPolicy.normalized()
	request.Provider = kisclient.ProviderKIS
	if request.Limit == "" {
		request.Limit = policy.limitLabel()
	}
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		if err := p.waitRead(ctx, request); err != nil {
			return zero, err
		}
		result, err := call(ctx)
		if err == nil {
			return result, nil
		}
		if !kisclient.RateLimitExceeded(err) || attempt == policy.MaxAttempts {
			return zero, err
		}
		if err := p.sleep(ctx, retryDelay(policy, attempt)); err != nil {
			return zero, err
		}
	}
	return zero, oops.In("kis_adapter").With("provider", provider.ProviderKIS).New("kis read retry exhausted unexpectedly")
}

func (p *Provider) waitRead(ctx context.Context, request kisclient.RateLimitRequest) error {
	if p.readLimiter == nil {
		return ctx.Err()
	}
	return p.readLimiter.Wait(ctx, request)
}

func retryDelay(policy RateLimitPolicy, attempt int) time.Duration {
	delay := policy.BaseDelay
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= policy.MaxDelay {
			return policy.MaxDelay + policy.Jitter
		}
	}
	if delay > policy.MaxDelay {
		delay = policy.MaxDelay
	}
	return delay + policy.Jitter
}

func kisReadRequest(group provider.GroupID, operation provider.OperationID, endpoint string) kisclient.RateLimitRequest {
	return kisclient.RateLimitRequest{
		Provider:  kisclient.ProviderKIS,
		Group:     string(group),
		Operation: string(operation),
		Endpoint:  endpoint,
	}
}

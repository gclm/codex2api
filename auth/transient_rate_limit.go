package auth

import (
	"time"
)

const (
	// TransientRateLimitBackoffBase is the first account-wide freeze after a
	// Codex throttle that is not usage_limit / 5h / 7d exhaustion.
	TransientRateLimitBackoffBase = 15 * time.Second
	// TransientRateLimitBackoffMax caps Retry-After and the exponential ladder.
	TransientRateLimitBackoffMax = 5 * time.Minute
	// transientRateLimitStableReset is how long an account must stay free of
	// new upstream 429s after LastRateLimitedAt before the backoff ladder
	// resets. Clearing on the first success would recreate a 15s retry loop
	// while concurrent in-flight requests are still being rejected.
	transientRateLimitStableReset = 5 * time.Minute
)

// nextTransientRateLimitCooldown returns the freeze duration for the given
// backoff level, never shorter than Retry-After and never longer than the cap.
func nextTransientRateLimitCooldown(level int, retryAfter time.Duration) time.Duration {
	if level < 0 {
		level = 0
	}
	cooldown := TransientRateLimitBackoffBase
	for step := 0; step < level && cooldown < TransientRateLimitBackoffMax; step++ {
		cooldown *= 2
	}
	if retryAfter > cooldown {
		cooldown = retryAfter
	}
	if cooldown > TransientRateLimitBackoffMax {
		cooldown = TransientRateLimitBackoffMax
	}
	if cooldown < TransientRateLimitBackoffBase {
		cooldown = TransientRateLimitBackoffBase
	}
	return cooldown
}

// MarkTransientRateLimited applies an account-wide short freeze for a Codex
// throttle. Concurrent 429s that land while the current window is still open
// reuse that deadline instead of climbing the backoff ladder. An already
// longer quota cooldown is left untouched.
//
// Unlike a quota cooldown the freeze is seconds long, so it neither triggers
// a WHAM usage probe nor is written to the database: under a burst that would
// turn every throttled account into one probe plus one write per window.
// The scheduler and the cross-instance cooldown cache are still updated.
func (s *Store) MarkTransientRateLimited(acc *Account, retryAfter time.Duration) time.Duration {
	if s == nil || acc == nil {
		return nextTransientRateLimitCooldown(0, retryAfter)
	}

	now := time.Now()
	acc.mu.Lock()
	if acc.Status == StatusCooldown && acc.CooldownUtil.After(now) {
		remaining := acc.CooldownUtil.Sub(now)
		acc.mu.Unlock()
		return remaining
	}
	level := acc.transientRateLimitBackoff
	cooldown := nextTransientRateLimitCooldown(level, retryAfter)
	if cooldown < TransientRateLimitBackoffMax {
		acc.transientRateLimitBackoff = level + 1
	}
	acc.mu.Unlock()

	s.markCooldownWithPersist(acc, cooldown, ResponsesRateLimitedCooldownReason, "", true, false)

	acc.mu.Lock()
	if acc.Status == StatusCooldown && acc.CooldownReason == ResponsesRateLimitedCooldownReason {
		acc.transientRateLimitUntil = acc.CooldownUtil
	}
	acc.mu.Unlock()
	return cooldown
}

// transientRateLimitRemainingLocked reports how long the current freeze still
// lasts when, and only when, the active cooldown is the one created by
// MarkTransientRateLimited and no quota window blocks the account underneath.
func (a *Account) transientRateLimitRemainingLocked(now time.Time) (time.Duration, bool) {
	if a.Status != StatusCooldown || !a.CooldownUtil.After(now) {
		return 0, false
	}
	if a.CooldownReason != ResponsesRateLimitedCooldownReason {
		return 0, false
	}
	if a.transientRateLimitUntil.IsZero() || !a.CooldownUtil.Equal(a.transientRateLimitUntil) {
		return 0, false
	}
	if a.usageWindowBlocksFreshDispatchLocked(now) {
		return 0, false
	}
	return a.CooldownUtil.Sub(now), true
}

// TransientRateLimitRemaining is the exported, locking form of
// transientRateLimitRemainingLocked.
func (a *Account) TransientRateLimitRemaining(now time.Time) (time.Duration, bool) {
	if a == nil {
		return 0, false
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.transientRateLimitRemainingLocked(now)
}

func (a *Account) observeTransientRateLimitSuccessLocked(now time.Time) {
	if a == nil || a.transientRateLimitBackoff == 0 {
		return
	}
	if a.Status == StatusCooldown && a.CooldownUtil.After(now) {
		return
	}
	if a.LastRateLimitedAt.IsZero() || now.Sub(a.LastRateLimitedAt) < transientRateLimitStableReset {
		return
	}
	a.transientRateLimitBackoff = 0
}

// TransientRateLimitBackoff reports the in-memory throttle backoff exponent.
func (a *Account) TransientRateLimitBackoff() int {
	if a == nil {
		return 0
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.transientRateLimitBackoff
}

package auth

import (
	"testing"
	"time"

	"github.com/codex2api/database"
)

func newTransientRateLimitTestStore() *Store {
	return NewStore(nil, nil, &database.SystemSettings{
		MaxConcurrency:  4,
		TestConcurrency: 1,
		TestModel:       "gpt-5.4",
	})
}

func TestMarkTransientRateLimitedProgressiveBackoff(t *testing.T) {
	store := newTransientRateLimitTestStore()
	acc := &Account{DBID: 1, AccessToken: "token", Status: StatusReady}

	first := store.MarkTransientRateLimited(acc, 0)
	if first < 14*time.Second || first > 16*time.Second {
		t.Fatalf("first cooldown = %v, want about 15s", first)
	}
	if got := acc.TransientRateLimitBackoff(); got != 1 {
		t.Fatalf("backoff after first = %d, want 1", got)
	}

	second := store.MarkTransientRateLimited(acc, 0)
	if second > first {
		t.Fatalf("same-window second cooldown = %v, want reuse of first window %v", second, first)
	}
	if got := acc.TransientRateLimitBackoff(); got != 1 {
		t.Fatalf("backoff after same-window repeat = %d, want 1", got)
	}

	acc.mu.Lock()
	acc.Status = StatusReady
	acc.CooldownUtil = time.Time{}
	acc.CooldownReason = ""
	acc.mu.Unlock()

	third := store.MarkTransientRateLimited(acc, 0)
	if third < 28*time.Second || third > 32*time.Second {
		t.Fatalf("escalated cooldown = %v, want about 30s", third)
	}
	if got := acc.TransientRateLimitBackoff(); got != 2 {
		t.Fatalf("backoff after escalate = %d, want 2", got)
	}
}

func TestMarkTransientRateLimitedRespectsRetryAfter(t *testing.T) {
	store := newTransientRateLimitTestStore()
	acc := &Account{DBID: 2, AccessToken: "token", Status: StatusReady}

	got := store.MarkTransientRateLimited(acc, 45*time.Second)
	if got < 44*time.Second || got > 46*time.Second {
		t.Fatalf("cooldown = %v, want about 45s Retry-After", got)
	}
}

func TestMarkTransientRateLimitedDoesNotShortenQuotaCooldown(t *testing.T) {
	store := newTransientRateLimitTestStore()
	acc := &Account{DBID: 3, AccessToken: "token", Status: StatusReady}
	store.MarkCooldown(acc, 2*time.Hour, "usage_limit")

	got := store.MarkTransientRateLimited(acc, 0)
	if got < time.Hour {
		t.Fatalf("cooldown = %v, want existing quota window preserved", got)
	}
	if acc.GetCooldownReason() != "usage_limit" {
		t.Fatalf("reason = %q, want usage_limit", acc.GetCooldownReason())
	}
}

func TestReportRequestSuccessResetsTransientBackoffAfterStableWindow(t *testing.T) {
	store := newTransientRateLimitTestStore()
	acc := &Account{DBID: 4, AccessToken: "token", Status: StatusReady}
	store.MarkTransientRateLimited(acc, 0)

	acc.mu.Lock()
	acc.Status = StatusReady
	acc.CooldownUtil = time.Time{}
	acc.CooldownReason = ""
	acc.LastRateLimitedAt = time.Now().Add(-transientRateLimitStableReset - time.Second)
	acc.mu.Unlock()

	store.ReportRequestSuccess(acc, 20*time.Millisecond)
	if got := acc.TransientRateLimitBackoff(); got != 0 {
		t.Fatalf("backoff after stable success = %d, want 0", got)
	}
}

func TestReportRequestSuccessKeepsBackoffDuringStableWindow(t *testing.T) {
	store := newTransientRateLimitTestStore()
	acc := &Account{DBID: 5, AccessToken: "token", Status: StatusReady}
	store.MarkTransientRateLimited(acc, 0)

	acc.mu.Lock()
	acc.Status = StatusReady
	acc.CooldownUtil = time.Time{}
	acc.CooldownReason = ""
	acc.LastRateLimitedAt = time.Now().Add(-time.Minute)
	acc.mu.Unlock()

	store.ReportRequestSuccess(acc, 20*time.Millisecond)
	if got := acc.TransientRateLimitBackoff(); got != 1 {
		t.Fatalf("backoff after early success = %d, want 1", got)
	}
}

func TestClearCooldownResetsTransientBackoff(t *testing.T) {
	store := newTransientRateLimitTestStore()
	acc := &Account{DBID: 6, AccessToken: "token", Status: StatusReady}
	store.MarkTransientRateLimited(acc, 0)
	store.ClearCooldown(acc)
	if got := acc.TransientRateLimitBackoff(); got != 0 {
		t.Fatalf("backoff after ClearCooldown = %d, want 0", got)
	}
}

func TestUsageLimitedCandidateSummaryTransientOnly(t *testing.T) {
	store := newTransientRateLimitTestStore()
	fast := &Account{DBID: 11, AccessToken: "token", Status: StatusReady}
	slow := &Account{DBID: 12, AccessToken: "token", Status: StatusReady}
	store.AddAccount(fast)
	store.AddAccount(slow)

	store.MarkTransientRateLimited(fast, 0)
	store.MarkTransientRateLimited(slow, 90*time.Second)

	got := store.UsageLimitedCandidateSummary(0, nil, nil, DispatchPolicyStandard)
	if !got.Found || !got.TransientOnly {
		t.Fatalf("summary = %#v, want Found && TransientOnly", got)
	}
	if got.RetryAfter < 10*time.Second || got.RetryAfter > 16*time.Second {
		t.Fatalf("RetryAfter = %v, want the shortest freeze (about 15s)", got.RetryAfter)
	}
	if _, ok := fast.TransientRateLimitRemaining(time.Now()); !ok {
		t.Fatal("fast account should report an active transient freeze")
	}
}

func TestUsageLimitedCandidateSummaryQuotaWins(t *testing.T) {
	store := newTransientRateLimitTestStore()
	throttled := &Account{DBID: 21, AccessToken: "token", Status: StatusReady}
	exhausted := &Account{DBID: 22, AccessToken: "token", Status: StatusReady}
	store.AddAccount(throttled)
	store.AddAccount(exhausted)

	store.MarkTransientRateLimited(throttled, 0)
	store.MarkCooldown(exhausted, 2*time.Hour, "usage_limit")

	got := store.UsageLimitedCandidateSummary(0, nil, nil, DispatchPolicyStandard)
	if !got.Found || got.TransientOnly {
		t.Fatalf("summary = %#v, want Found && !TransientOnly", got)
	}
	if got.RetryAfter != 0 {
		t.Fatalf("RetryAfter = %v, want 0 when a quota window is exhausted", got.RetryAfter)
	}
}

func TestTransientRateLimitRemainingIgnoresQuotaCooldown(t *testing.T) {
	store := newTransientRateLimitTestStore()
	acc := &Account{DBID: 31, AccessToken: "token", Status: StatusReady}
	store.MarkTransientRateLimited(acc, 0)
	// A later, longer quota cooldown replaces the transient deadline.
	store.MarkCooldown(acc, time.Hour, "usage_limit")
	if _, ok := acc.TransientRateLimitRemaining(time.Now()); ok {
		t.Fatal("quota cooldown must not be reported as transient")
	}
	// The same reason with a different deadline (real usage_limit via
	// MarkResponsesRateLimited) is not transient either.
	acc2 := &Account{DBID: 32, AccessToken: "token", Status: StatusReady}
	store.MarkResponsesRateLimited(acc2, time.Hour)
	if _, ok := acc2.TransientRateLimitRemaining(time.Now()); ok {
		t.Fatal("MarkResponsesRateLimited cooldown must not be reported as transient")
	}
}

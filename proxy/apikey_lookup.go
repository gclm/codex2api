package proxy

import (
	"crypto/sha256"
	"slices"
	"strings"

	"github.com/codex2api/database"
)

// Only overlapping lookups share a result. Nothing is retained between calls:
// constrained keys still observe current database state, including quota usage,
// revocation and expiry. Infrastructure errors are never negative-cached.
func (h *Handler) resolveAPIKey(key string) (*database.APIKeyRow, bool, error) {
	key = strings.TrimSpace(key)
	if key == "" || h.configKeys[key] {
		return h.resolveAPIKeyUnshared(key)
	}
	digest := sha256.Sum256([]byte(key))
	value, err, shared := h.apiKeyLookups.Do(string(digest[:]), func() (any, error) {
		row, _, err := h.resolveAPIKeyUnshared(key)
		return row, err
	})
	if err != nil {
		return nil, false, err
	}
	row, _ := value.(*database.APIKeyRow)
	if row != nil && shared {
		row = cloneAPIKeyLookupRow(row)
	}
	return row, row != nil, nil
}

func cloneAPIKeyLookupRow(row *database.APIKeyRow) *database.APIKeyRow {
	copy := *row
	copy.AllowedGroupIDs = slices.Clone(row.AllowedGroupIDs)
	copy.Limits.ModelRequestLimits = slices.Clone(row.Limits.ModelRequestLimits)
	copy.Limits.ModelAllow = slices.Clone(row.Limits.ModelAllow)
	copy.Limits.ModelDeny = slices.Clone(row.Limits.ModelDeny)
	copy.Limits.PlanAllow = slices.Clone(row.Limits.PlanAllow)
	copy.Limits.NoAffinityGroupIDs = slices.Clone(row.Limits.NoAffinityGroupIDs)
	copy.Limits.ScopeLimits = slices.Clone(row.Limits.ScopeLimits)
	return &copy
}

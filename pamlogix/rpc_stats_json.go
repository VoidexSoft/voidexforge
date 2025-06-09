package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/heroiclabs/nakama-common/runtime"
)

func rpcStatsGet_Json(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		statsSystem := p.GetStatsSystem()
		if statsSystem == nil {
			return "", runtime.NewError("stats system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		// List only for the current user
		statsMap, err := statsSystem.List(ctx, logger, nk, userId, []string{userId})
		if err != nil {
			return "", err
		}
		stats, ok := statsMap[userId]
		if !ok || stats == nil {
			stats = &StatList{Public: map[string]*Stat{}, Private: map[string]*Stat{}}
		}
		data, err := json.Marshal(stats)
		if err != nil {
			logger.Error("Failed to marshal stats: %v", err)
			return "", runtime.NewError("failed to marshal stats", INTERNAL_ERROR_CODE) // INTERNAL
		}
		return string(data), nil
	}
}

func rpcStatsUpdate_Json(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		statsSystem := p.GetStatsSystem()
		if statsSystem == nil {
			return "", runtime.NewError("stats system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var req StatUpdateRequest
		if err := json.Unmarshal([]byte(payload), &req); err != nil {
			logger.Error("Failed to unmarshal StatUpdateRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal stat update request", INTERNAL_ERROR_CODE) // INTERNAL
		}

		stats, err := statsSystem.Update(ctx, logger, nk, userId, req.Public, req.Private)
		if err != nil {
			return "", err
		}
		data, err := json.Marshal(stats)
		if err != nil {
			logger.Error("Failed to marshal updated stats: %v", err)
			return "", runtime.NewError("failed to marshal updated stats", INTERNAL_ERROR_CODE) // INTERNAL
		}
		return string(data), nil
	}
}

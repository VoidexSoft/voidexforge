package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/heroiclabs/nakama-common/runtime"
)

// Streaks RPC functions

func rpcStreaksList(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		streaksSystem := p.GetStreaksSystem()
		if streaksSystem == nil {
			return "", runtime.NewError("streaks system not available", 12) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", 3) // INVALID_ARGUMENT
		}

		streaks, err := streaksSystem.List(ctx, logger, nk, userID)
		if err != nil {
			return "", err
		}

		// Create response in the format expected by the protobuf StreaksList
		response := &StreaksList{
			Streaks: streaks,
		}

		data, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal streaks: %v", err)
			return "", runtime.NewError("failed to marshal streaks", 13) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcStreaksUpdate(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		streaksSystem := p.GetStreaksSystem()
		if streaksSystem == nil {
			return "", runtime.NewError("streaks system not available", 12) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", 3) // INVALID_ARGUMENT
		}

		var request StreaksUpdateRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal StreaksUpdateRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal streaks update request", 3) // INVALID_ARGUMENT
		}

		if len(request.Updates) == 0 {
			return "", runtime.NewError("at least one streak update is required", 3) // INVALID_ARGUMENT
		}

		streaks, err := streaksSystem.Update(ctx, logger, nk, userID, request.Updates)
		if err != nil {
			return "", err
		}

		// Create response in the format expected by the protobuf StreaksList
		response := &StreaksList{
			Streaks: streaks,
		}

		data, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal streaks: %v", err)
			return "", runtime.NewError("failed to marshal streaks", 13) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcStreaksClaim(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		streaksSystem := p.GetStreaksSystem()
		if streaksSystem == nil {
			return "", runtime.NewError("streaks system not available", 12) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", 3) // INVALID_ARGUMENT
		}

		var request StreaksClaimRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal StreaksClaimRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal streaks claim request", 3) // INVALID_ARGUMENT
		}

		if len(request.Ids) == 0 {
			return "", runtime.NewError("at least one streak id is required", 3) // INVALID_ARGUMENT
		}

		streaks, err := streaksSystem.Claim(ctx, logger, nk, userID, request.Ids)
		if err != nil {
			return "", err
		}

		// Create response in the format expected by the protobuf StreaksList
		response := &StreaksList{
			Streaks: streaks,
		}

		data, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal streaks: %v", err)
			return "", runtime.NewError("failed to marshal streaks", 13) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcStreaksReset(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		streaksSystem := p.GetStreaksSystem()
		if streaksSystem == nil {
			return "", runtime.NewError("streaks system not available", 12) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", 3) // INVALID_ARGUMENT
		}

		var request StreaksResetRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal StreaksResetRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal streaks reset request", 3) // INVALID_ARGUMENT
		}

		if len(request.Ids) == 0 {
			return "", runtime.NewError("at least one streak id is required", 3) // INVALID_ARGUMENT
		}

		streaks, err := streaksSystem.Reset(ctx, logger, nk, userID, request.Ids)
		if err != nil {
			return "", err
		}

		// Create response in the format expected by the protobuf StreaksList
		response := &StreaksList{
			Streaks: streaks,
		}

		data, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal streaks: %v", err)
			return "", runtime.NewError("failed to marshal streaks", 13) // INTERNAL
		}

		return string(data), nil
	}
}

package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/heroiclabs/nakama-common/runtime"
)

// Unlockables RPC functions with JSON support

func rpcUnlockablesCreateJson(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		// Create a random unlockable (no specific ID or config provided)
		unlockables, err := unlockablesSystem.Create(ctx, logger, nk, userID, "", nil)
		if err != nil {
			return "", err
		}

		// Convert to JSON
		data, err := json.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables to JSON: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesGetJson(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		unlockables, err := unlockablesSystem.Get(ctx, logger, nk, userID)
		if err != nil {
			return "", err
		}

		// Convert to JSON
		data, err := json.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables to JSON: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesUnlockStartJson(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var request struct {
			InstanceId string `json:"instanceId"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal UnlockablesRequest from JSON: %v", err)
			return "", runtime.NewError("failed to unmarshal unlockables request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if request.InstanceId == "" {
			return "", runtime.NewError("instance id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		unlockables, err := unlockablesSystem.UnlockStart(ctx, logger, nk, userID, request.InstanceId)
		if err != nil {
			return "", err
		}

		// Convert to JSON
		data, err := json.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables to JSON: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesPurchaseUnlockJson(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var request struct {
			InstanceId string `json:"instanceId"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal UnlockablesRequest from JSON: %v", err)
			return "", runtime.NewError("failed to unmarshal unlockables request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if request.InstanceId == "" {
			return "", runtime.NewError("instance id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		unlockables, err := unlockablesSystem.PurchaseUnlock(ctx, logger, nk, userID, request.InstanceId)
		if err != nil {
			return "", err
		}

		// Convert to JSON
		data, err := json.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables to JSON: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesPurchaseSlotJson(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		unlockables, err := unlockablesSystem.PurchaseSlot(ctx, logger, nk, userID)
		if err != nil {
			return "", err
		}

		// Convert to JSON
		data, err := json.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables to JSON: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesClaimJson(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var request struct {
			InstanceId string `json:"instanceId"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal UnlockablesRequest from JSON: %v", err)
			return "", runtime.NewError("failed to unmarshal unlockables request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if request.InstanceId == "" {
			return "", runtime.NewError("instance id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		reward, err := unlockablesSystem.Claim(ctx, logger, nk, userID, request.InstanceId)
		if err != nil {
			return "", err
		}

		// Convert to JSON
		data, err := json.Marshal(reward)
		if err != nil {
			logger.Error("Failed to marshal unlockables reward to JSON: %v", err)
			return "", runtime.NewError("failed to marshal unlockables reward", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesQueueAddJson(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var request struct {
			InstanceIds []string `json:"instanceIds"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal UnlockablesQueueAddRequest from JSON: %v", err)
			return "", runtime.NewError("failed to unmarshal unlockables queue add request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if len(request.InstanceIds) == 0 {
			return "", runtime.NewError("at least one instance id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		unlockables, err := unlockablesSystem.QueueAdd(ctx, logger, nk, userID, request.InstanceIds)
		if err != nil {
			return "", err
		}

		// Convert to JSON
		data, err := json.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables to JSON: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesQueueRemoveJson(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var request struct {
			InstanceIds []string `json:"instanceIds"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal UnlockablesQueueRemoveRequest from JSON: %v", err)
			return "", runtime.NewError("failed to unmarshal unlockables queue remove request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if len(request.InstanceIds) == 0 {
			return "", runtime.NewError("at least one instance id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		unlockables, err := unlockablesSystem.QueueRemove(ctx, logger, nk, userID, request.InstanceIds)
		if err != nil {
			return "", err
		}

		// Convert to JSON
		data, err := json.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables to JSON: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesQueueSetJson(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var request UnlockablesQueueSetRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal UnlockablesQueueSetRequest from JSON: %v", err)
			return "", runtime.NewError("failed to unmarshal unlockables queue set request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		unlockables, err := unlockablesSystem.QueueSet(ctx, logger, nk, userID, request.InstanceIds)
		if err != nil {
			return "", err
		}

		// Convert to JSON
		data, err := json.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables to JSON: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

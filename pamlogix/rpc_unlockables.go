package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/heroiclabs/nakama-common/runtime"
)

// Unlockables RPC functions

func rpcUnlockablesCreate(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", 12) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", 3) // INVALID_ARGUMENT
		}

		// Create a random unlockable (no specific ID or config provided)
		unlockables, err := unlockablesSystem.Create(ctx, logger, nk, userID, "", nil)
		if err != nil {
			return "", err
		}

		data, err := json.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", 13) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesGet(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", 12) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", 3) // INVALID_ARGUMENT
		}

		unlockables, err := unlockablesSystem.Get(ctx, logger, nk, userID)
		if err != nil {
			return "", err
		}

		data, err := json.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", 13) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesUnlockStart(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", 12) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", 3) // INVALID_ARGUMENT
		}

		var request UnlockablesRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal UnlockablesRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal unlockables request", 3) // INVALID_ARGUMENT
		}

		if request.InstanceId == "" {
			return "", runtime.NewError("instance id is required", 3) // INVALID_ARGUMENT
		}

		unlockables, err := unlockablesSystem.UnlockStart(ctx, logger, nk, userID, request.InstanceId)
		if err != nil {
			return "", err
		}

		data, err := json.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", 13) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesPurchaseUnlock(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", 12) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", 3) // INVALID_ARGUMENT
		}

		var request UnlockablesRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal UnlockablesRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal unlockables request", 3) // INVALID_ARGUMENT
		}

		if request.InstanceId == "" {
			return "", runtime.NewError("instance id is required", 3) // INVALID_ARGUMENT
		}

		unlockables, err := unlockablesSystem.PurchaseUnlock(ctx, logger, nk, userID, request.InstanceId)
		if err != nil {
			return "", err
		}

		data, err := json.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", 13) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesPurchaseSlot(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", 12) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", 3) // INVALID_ARGUMENT
		}

		unlockables, err := unlockablesSystem.PurchaseSlot(ctx, logger, nk, userID)
		if err != nil {
			return "", err
		}

		data, err := json.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", 13) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesClaim(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", 12) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", 3) // INVALID_ARGUMENT
		}

		var request UnlockablesRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal UnlockablesRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal unlockables request", 3) // INVALID_ARGUMENT
		}

		if request.InstanceId == "" {
			return "", runtime.NewError("instance id is required", 3) // INVALID_ARGUMENT
		}

		reward, err := unlockablesSystem.Claim(ctx, logger, nk, userID, request.InstanceId)
		if err != nil {
			return "", err
		}

		// Create response in the format expected by the protobuf UnlockablesReward
		response := &UnlockablesReward{
			Unlockables: reward.Unlockables, // Use the updated unlockables list from the reward
			Reward:      reward.Reward,
		}

		data, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal unlockables reward: %v", err)
			return "", runtime.NewError("failed to marshal unlockables reward", 13) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesQueueAdd(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", 12) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", 3) // INVALID_ARGUMENT
		}

		var request UnlockablesQueueAddRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal UnlockablesQueueAddRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal unlockables queue add request", 3) // INVALID_ARGUMENT
		}

		if len(request.InstanceIds) == 0 {
			return "", runtime.NewError("at least one instance id is required", 3) // INVALID_ARGUMENT
		}

		unlockables, err := unlockablesSystem.QueueAdd(ctx, logger, nk, userID, request.InstanceIds)
		if err != nil {
			return "", err
		}

		data, err := json.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", 13) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesQueueRemove(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", 12) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", 3) // INVALID_ARGUMENT
		}

		var request UnlockablesQueueRemoveRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal UnlockablesQueueRemoveRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal unlockables queue remove request", 3) // INVALID_ARGUMENT
		}

		if len(request.InstanceIds) == 0 {
			return "", runtime.NewError("at least one instance id is required", 3) // INVALID_ARGUMENT
		}

		unlockables, err := unlockablesSystem.QueueRemove(ctx, logger, nk, userID, request.InstanceIds)
		if err != nil {
			return "", err
		}

		data, err := json.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", 13) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesQueueSet(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", 12) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", 3) // INVALID_ARGUMENT
		}

		var request UnlockablesQueueSetRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal UnlockablesQueueSetRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal unlockables queue set request", 3) // INVALID_ARGUMENT
		}

		unlockables, err := unlockablesSystem.QueueSet(ctx, logger, nk, userID, request.InstanceIds)
		if err != nil {
			return "", err
		}

		data, err := json.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", 13) // INTERNAL
		}

		return string(data), nil
	}
}

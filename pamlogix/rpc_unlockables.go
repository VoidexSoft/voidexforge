package pamlogix

import (
	"context"
	"database/sql"

	"github.com/heroiclabs/nakama-common/runtime"
	"google.golang.org/protobuf/proto"
)

// Unlockables RPC functions

func rpcUnlockablesCreate(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
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

		data, err := proto.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesGet(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
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

		data, err := proto.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesUnlockStart(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var request UnlockablesRequest
		if err := proto.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal UnlockablesRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal unlockables request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if request.InstanceId == "" {
			return "", runtime.NewError("instance id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		unlockables, err := unlockablesSystem.UnlockStart(ctx, logger, nk, userID, request.InstanceId)
		if err != nil {
			return "", err
		}

		data, err := proto.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesPurchaseUnlock(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var request UnlockablesRequest
		if err := proto.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal UnlockablesRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal unlockables request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if request.InstanceId == "" {
			return "", runtime.NewError("instance id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		unlockables, err := unlockablesSystem.PurchaseUnlock(ctx, logger, nk, userID, request.InstanceId)
		if err != nil {
			return "", err
		}

		data, err := proto.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesPurchaseSlot(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
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

		data, err := proto.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesClaim(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var request UnlockablesRequest
		if err := proto.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal UnlockablesRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal unlockables request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if request.InstanceId == "" {
			return "", runtime.NewError("instance id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		reward, err := unlockablesSystem.Claim(ctx, logger, nk, userID, request.InstanceId)
		if err != nil {
			return "", err
		}

		// Create response in the format expected by the protobuf UnlockablesReward
		response := &UnlockablesReward{
			Unlockables: &UnlockablesList{
				Unlockables:      reward.Unlockables.Unlockables,
				Overflow:         reward.Unlockables.Overflow,
				Slots:            reward.Unlockables.Slots,
				ActiveSlots:      reward.Unlockables.ActiveSlots,
				MaxActiveSlots:   reward.Unlockables.MaxActiveSlots,
				SlotCost:         reward.Unlockables.SlotCost,
				InstanceId:       reward.Unlockables.InstanceId,
				QueuedUnlocks:    reward.Unlockables.QueuedUnlocks,
				MaxQueuedUnlocks: reward.Unlockables.MaxQueuedUnlocks,
			},
			Reward: reward.Reward,
		}

		data, err := proto.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal unlockables reward: %v", err)
			return "", runtime.NewError("failed to marshal unlockables reward", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesQueueAdd(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var request UnlockablesQueueAddRequest
		if err := proto.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal UnlockablesQueueAddRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal unlockables queue add request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if len(request.InstanceIds) == 0 {
			return "", runtime.NewError("at least one instance id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		unlockables, err := unlockablesSystem.QueueAdd(ctx, logger, nk, userID, request.InstanceIds)
		if err != nil {
			return "", err
		}

		data, err := proto.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesQueueRemove(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		unlockablesSystem := p.GetUnlockablesSystem()
		if unlockablesSystem == nil {
			return "", runtime.NewError("unlockables system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var request UnlockablesQueueRemoveRequest
		if err := proto.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal UnlockablesQueueRemoveRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal unlockables queue remove request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if len(request.InstanceIds) == 0 {
			return "", runtime.NewError("at least one instance id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		unlockables, err := unlockablesSystem.QueueRemove(ctx, logger, nk, userID, request.InstanceIds)
		if err != nil {
			return "", err
		}

		data, err := proto.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcUnlockablesQueueSet(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
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
		if err := proto.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal UnlockablesQueueSetRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal unlockables queue set request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		unlockables, err := unlockablesSystem.QueueSet(ctx, logger, nk, userID, request.InstanceIds)
		if err != nil {
			return "", err
		}

		data, err := proto.Marshal(unlockables)
		if err != nil {
			logger.Error("Failed to marshal unlockables: %v", err)
			return "", runtime.NewError("failed to marshal unlockables", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/heroiclabs/nakama-common/runtime"
)

func rpcProgressionsGet(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		progressionSystem := p.GetProgressionSystem()
		if progressionSystem == nil {
			return "", runtime.NewError("progression system not available", 12) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", 3) // INVALID_ARGUMENT
		}

		var request ProgressionGetRequest
		if payload != "" {
			if err := json.Unmarshal([]byte(payload), &request); err != nil {
				logger.Error("Failed to unmarshal ProgressionGetRequest: %v", err)
				return "", runtime.NewError("failed to unmarshal progression get request", 3) // INVALID_ARGUMENT
			}
		}

		progressions, deltas, err := progressionSystem.Get(ctx, logger, nk, userId, request.Progressions)
		if err != nil {
			return "", err
		}

		response := &ProgressionList{
			Progressions: progressions,
			Deltas:       deltas,
		}

		data, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal progression response: %v", err)
			return "", runtime.NewError("failed to marshal progression response", 13) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcProgressionsPurchase(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		progressionSystem := p.GetProgressionSystem()
		if progressionSystem == nil {
			return "", runtime.NewError("progression system not available", 12) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", 3) // INVALID_ARGUMENT
		}

		var request ProgressionPurchaseRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal ProgressionPurchaseRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal progression purchase request", 3) // INVALID_ARGUMENT
		}

		if request.Id == "" {
			return "", runtime.NewError("progression id is required", 3) // INVALID_ARGUMENT
		}

		progressions, err := progressionSystem.Purchase(ctx, logger, nk, userId, request.Id)
		if err != nil {
			return "", err
		}

		response := &ProgressionList{
			Progressions: progressions,
		}

		data, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal progression response: %v", err)
			return "", runtime.NewError("failed to marshal progression response", 13) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcProgressionsUpdate(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		progressionSystem := p.GetProgressionSystem()
		if progressionSystem == nil {
			return "", runtime.NewError("progression system not available", 12) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", 3) // INVALID_ARGUMENT
		}

		var request ProgressionUpdateRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal ProgressionUpdateRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal progression update request", 3) // INVALID_ARGUMENT
		}

		if request.Id == "" {
			return "", runtime.NewError("progression id is required", 3) // INVALID_ARGUMENT
		}

		if len(request.Counts) == 0 {
			return "", runtime.NewError("counts are required for progression update", 3) // INVALID_ARGUMENT
		}

		progressions, err := progressionSystem.Update(ctx, logger, nk, userId, request.Id, request.Counts)
		if err != nil {
			return "", err
		}

		response := &ProgressionList{
			Progressions: progressions,
		}

		data, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal progression response: %v", err)
			return "", runtime.NewError("failed to marshal progression response", 13) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcProgressionsReset(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		progressionSystem := p.GetProgressionSystem()
		if progressionSystem == nil {
			return "", runtime.NewError("progression system not available", 12) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", 3) // INVALID_ARGUMENT
		}

		var request ProgressionResetRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal ProgressionResetRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal progression reset request", 3) // INVALID_ARGUMENT
		}

		if len(request.Ids) == 0 {
			return "", runtime.NewError("progression ids are required", 3) // INVALID_ARGUMENT
		}

		progressions, err := progressionSystem.Reset(ctx, logger, nk, userId, request.Ids)
		if err != nil {
			return "", err
		}

		response := &ProgressionList{
			Progressions: progressions,
		}

		data, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal progression response: %v", err)
			return "", runtime.NewError("failed to marshal progression response", 13) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcProgressionsComplete(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		progressionSystem := p.GetProgressionSystem()
		if progressionSystem == nil {
			return "", runtime.NewError("progression system not available", 12) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", 3) // INVALID_ARGUMENT
		}

		var request struct {
			Id string `json:"id"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal ProgressionCompleteRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal progression complete request", 3) // INVALID_ARGUMENT
		}

		if request.Id == "" {
			return "", runtime.NewError("progression id is required", 3) // INVALID_ARGUMENT
		}

		progressions, reward, err := progressionSystem.Complete(ctx, logger, nk, userId, request.Id)
		if err != nil {
			return "", err
		}

		response := struct {
			Progressions map[string]*Progression `json:"progressions"`
			Reward       *Reward                 `json:"reward,omitempty"`
		}{
			Progressions: progressions,
			Reward:       reward,
		}

		data, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal progression response: %v", err)
			return "", runtime.NewError("failed to marshal progression response", 13) // INTERNAL
		}

		return string(data), nil
	}
}

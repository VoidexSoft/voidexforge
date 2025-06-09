package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/heroiclabs/nakama-common/runtime"
)

func rpcProgressionsGet_Json(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		progressionSystem := p.GetProgressionSystem()
		if progressionSystem == nil {
			return "", runtime.NewError("progression system not available", UNIMPLEMENTED_ERROR_CODE)
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE)
		}

		var request ProgressionGetRequest
		if payload != "" {
			if err := json.Unmarshal([]byte(payload), &request); err != nil {
				logger.Error("Failed to unmarshal ProgressionGetRequestJson: %v", err)
				return "", runtime.NewError("failed to unmarshal progression get request", INVALID_ARGUMENT_ERROR_CODE)
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
			return "", runtime.NewError("failed to marshal progression response", INTERNAL_ERROR_CODE)
		}

		return string(data), nil
	}
}

func rpcProgressionsPurchase_Json(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		progressionSystem := p.GetProgressionSystem()
		if progressionSystem == nil {
			return "", runtime.NewError("progression system not available", UNIMPLEMENTED_ERROR_CODE)
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE)
		}

		var request ProgressionPurchaseRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal ProgressionPurchaseRequestJson: %v", err)
			return "", runtime.NewError("failed to unmarshal progression purchase request", INVALID_ARGUMENT_ERROR_CODE)
		}

		if request.Id == "" {
			return "", runtime.NewError("progression id is required", INVALID_ARGUMENT_ERROR_CODE)
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
			return "", runtime.NewError("failed to marshal progression response", INTERNAL_ERROR_CODE)
		}

		return string(data), nil
	}
}

func rpcProgressionsUpdate_Json(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		progressionSystem := p.GetProgressionSystem()
		if progressionSystem == nil {
			return "", runtime.NewError("progression system not available", UNIMPLEMENTED_ERROR_CODE)
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE)
		}

		var request ProgressionUpdateRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal ProgressionUpdateRequestJson: %v", err)
			return "", runtime.NewError("failed to unmarshal progression update request", INVALID_ARGUMENT_ERROR_CODE)
		}

		if request.Id == "" {
			return "", runtime.NewError("progression id is required", INVALID_ARGUMENT_ERROR_CODE)
		}

		if len(request.Counts) == 0 {
			return "", runtime.NewError("counts are required for progression update", INVALID_ARGUMENT_ERROR_CODE)
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
			return "", runtime.NewError("failed to marshal progression response", INTERNAL_ERROR_CODE)
		}

		return string(data), nil
	}
}

func rpcProgressionsReset_Json(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		progressionSystem := p.GetProgressionSystem()
		if progressionSystem == nil {
			return "", runtime.NewError("progression system not available", UNIMPLEMENTED_ERROR_CODE)
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE)
		}

		var request ProgressionResetRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal ProgressionResetRequestJson: %v", err)
			return "", runtime.NewError("failed to unmarshal progression reset request", INVALID_ARGUMENT_ERROR_CODE)
		}

		if len(request.Ids) == 0 {
			return "", runtime.NewError("progression ids are required", INVALID_ARGUMENT_ERROR_CODE)
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
			return "", runtime.NewError("failed to marshal progression response", INTERNAL_ERROR_CODE)
		}

		return string(data), nil
	}
}

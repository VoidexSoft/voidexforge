package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/heroiclabs/nakama-common/runtime"
)

func rpcTutorialsGetJson(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		tutorialsSystem := p.GetTutorialsSystem()
		if tutorialsSystem == nil {
			return "", runtime.NewError("tutorials system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		tutorials, err := tutorialsSystem.Get(ctx, logger, nk, userID)
		if err != nil {
			return "", err
		}

		// Create response in JSON format
		response := &TutorialList{
			Tutorials: tutorials,
		}

		data, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal tutorials: %v", err)
			return "", runtime.NewError("failed to marshal tutorials", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcTutorialsAcceptJson(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		tutorialsSystem := p.GetTutorialsSystem()
		if tutorialsSystem == nil {
			return "", runtime.NewError("tutorials system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var request TutorialAcceptRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal TutorialAcceptRequestJson: %v", err)
			return "", runtime.NewError("failed to unmarshal tutorial accept request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if request.Id == "" {
			return "", runtime.NewError("tutorial id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		tutorial, err := tutorialsSystem.Accept(ctx, logger, nk, request.Id, userID)
		if err != nil {
			return "", err
		}

		data, err := json.Marshal(tutorial)
		if err != nil {
			logger.Error("Failed to marshal tutorial: %v", err)
			return "", runtime.NewError("failed to marshal tutorial", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcTutorialsDeclineJson(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		tutorialsSystem := p.GetTutorialsSystem()
		if tutorialsSystem == nil {
			return "", runtime.NewError("tutorials system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var request TutorialDeclineRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal TutorialDeclineRequestJson: %v", err)
			return "", runtime.NewError("failed to unmarshal tutorial decline request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if request.Id == "" {
			return "", runtime.NewError("tutorial id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		tutorial, err := tutorialsSystem.Decline(ctx, logger, nk, request.Id, userID)
		if err != nil {
			return "", err
		}

		data, err := json.Marshal(tutorial)
		if err != nil {
			logger.Error("Failed to marshal tutorial: %v", err)
			return "", runtime.NewError("failed to marshal tutorial", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcTutorialsAbandonJson(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		tutorialsSystem := p.GetTutorialsSystem()
		if tutorialsSystem == nil {
			return "", runtime.NewError("tutorials system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var request TutorialAbandonRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal TutorialAbandonRequestJson: %v", err)
			return "", runtime.NewError("failed to unmarshal tutorial abandon request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if request.Id == "" {
			return "", runtime.NewError("tutorial id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		tutorial, err := tutorialsSystem.Abandon(ctx, logger, nk, request.Id, userID)
		if err != nil {
			return "", err
		}

		data, err := json.Marshal(tutorial)
		if err != nil {
			logger.Error("Failed to marshal tutorial: %v", err)
			return "", runtime.NewError("failed to marshal tutorial", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcTutorialsUpdateJson(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		tutorialsSystem := p.GetTutorialsSystem()
		if tutorialsSystem == nil {
			return "", runtime.NewError("tutorials system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var request TutorialUpdateRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal TutorialUpdateRequestJson: %v", err)
			return "", runtime.NewError("failed to unmarshal tutorial update request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if request.Id == "" {
			return "", runtime.NewError("tutorial id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		tutorials, err := tutorialsSystem.Update(ctx, logger, nk, userID, request.Id, int(request.Step))
		if err != nil {
			return "", err
		}

		// Create response in JSON format
		response := &TutorialList{
			Tutorials: tutorials,
		}

		data, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal tutorials: %v", err)
			return "", runtime.NewError("failed to marshal tutorials", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcTutorialsResetJson(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		tutorialsSystem := p.GetTutorialsSystem()
		if tutorialsSystem == nil {
			return "", runtime.NewError("tutorials system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var request TutorialResetRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal TutorialResetRequestJson: %v", err)
			return "", runtime.NewError("failed to unmarshal tutorial reset request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if len(request.Ids) == 0 {
			return "", runtime.NewError("at least one tutorial id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		tutorials, err := tutorialsSystem.Reset(ctx, logger, nk, userID, request.Ids)
		if err != nil {
			return "", err
		}

		// Create response in JSON format
		response := &TutorialList{
			Tutorials: tutorials,
		}

		data, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal tutorials: %v", err)
			return "", runtime.NewError("failed to marshal tutorials", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

package pamlogix

import (
	"context"
	"database/sql"

	"github.com/heroiclabs/nakama-common/runtime"
	"google.golang.org/protobuf/proto"
)

func rpcTutorialsGet(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
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

		// Create response in the format expected by the protobuf TutorialList
		response := &TutorialList{
			Tutorials: tutorials,
		}

		data, err := proto.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal tutorials: %v", err)
			return "", runtime.NewError("failed to marshal tutorials", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcTutorialsAccept(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
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
		if err := proto.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal TutorialAcceptRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal tutorial accept request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if request.Id == "" {
			return "", runtime.NewError("tutorial id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		tutorial, err := tutorialsSystem.Accept(ctx, logger, nk, request.Id, userID)
		if err != nil {
			return "", err
		}

		data, err := proto.Marshal(tutorial)
		if err != nil {
			logger.Error("Failed to marshal tutorial: %v", err)
			return "", runtime.NewError("failed to marshal tutorial", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcTutorialsDecline(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
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
		if err := proto.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal TutorialDeclineRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal tutorial decline request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if request.Id == "" {
			return "", runtime.NewError("tutorial id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		tutorial, err := tutorialsSystem.Decline(ctx, logger, nk, request.Id, userID)
		if err != nil {
			return "", err
		}

		data, err := proto.Marshal(tutorial)
		if err != nil {
			logger.Error("Failed to marshal tutorial: %v", err)
			return "", runtime.NewError("failed to marshal tutorial", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcTutorialsAbandon(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
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
		if err := proto.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal TutorialAbandonRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal tutorial abandon request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if request.Id == "" {
			return "", runtime.NewError("tutorial id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		tutorial, err := tutorialsSystem.Abandon(ctx, logger, nk, request.Id, userID)
		if err != nil {
			return "", err
		}

		data, err := proto.Marshal(tutorial)
		if err != nil {
			logger.Error("Failed to marshal tutorial: %v", err)
			return "", runtime.NewError("failed to marshal tutorial", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcTutorialsUpdate(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
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
		if err := proto.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal TutorialUpdateRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal tutorial update request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if request.Id == "" {
			return "", runtime.NewError("tutorial id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		tutorials, err := tutorialsSystem.Update(ctx, logger, nk, userID, request.Id, int(request.Step))
		if err != nil {
			return "", err
		}

		// Create response in the format expected by the protobuf TutorialList
		response := &TutorialList{
			Tutorials: tutorials,
		}

		data, err := proto.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal tutorials: %v", err)
			return "", runtime.NewError("failed to marshal tutorials", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcTutorialsReset(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
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
		if err := proto.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal TutorialResetRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal tutorial reset request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if len(request.Ids) == 0 {
			return "", runtime.NewError("at least one tutorial id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		tutorials, err := tutorialsSystem.Reset(ctx, logger, nk, userID, request.Ids)
		if err != nil {
			return "", err
		}

		// Create response in the format expected by the protobuf TutorialList
		response := &TutorialList{
			Tutorials: tutorials,
		}

		data, err := proto.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal tutorials: %v", err)
			return "", runtime.NewError("failed to marshal tutorials", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

package pamlogix

import (
	"context"
	"database/sql"

	"github.com/heroiclabs/nakama-common/runtime"
	"google.golang.org/protobuf/proto"
)

func rpcTeamsCreate(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		teamsSystem := p.GetTeamsSystem()
		if teamsSystem == nil {
			return "", runtime.NewError("teams system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		var request TeamCreateRequest
		if err := proto.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal TeamCreateRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal team create request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		team, err := teamsSystem.Create(ctx, logger, nk, &request)
		if err != nil {
			return "", err
		}

		data, err := proto.Marshal(team)
		if err != nil {
			logger.Error("Failed to marshal team: %v", err)
			return "", runtime.NewError("failed to marshal team", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcTeamsList(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		teamsSystem := p.GetTeamsSystem()
		if teamsSystem == nil {
			return "", runtime.NewError("teams system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		var request TeamListRequest
		if err := proto.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal TeamListRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal team list request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		teamList, err := teamsSystem.List(ctx, logger, nk, &request)
		if err != nil {
			return "", err
		}

		data, err := proto.Marshal(teamList)
		if err != nil {
			logger.Error("Failed to marshal team list: %v", err)
			return "", runtime.NewError("failed to marshal team list", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcTeamsSearch(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		teamsSystem := p.GetTeamsSystem()
		if teamsSystem == nil {
			return "", runtime.NewError("teams system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		var request TeamSearchRequest
		if err := proto.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal TeamSearchRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal team search request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		teamList, err := teamsSystem.Search(ctx, db, logger, nk, &request)
		if err != nil {
			return "", err
		}

		data, err := proto.Marshal(teamList)
		if err != nil {
			logger.Error("Failed to marshal team list: %v", err)
			return "", runtime.NewError("failed to marshal team list", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcTeamsWriteChatMessage(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		teamsSystem := p.GetTeamsSystem()
		if teamsSystem == nil {
			return "", runtime.NewError("teams system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var request TeamWriteChatMessageRequest
		if err := proto.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal TeamWriteChatMessageRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal team write chat message request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		ack, err := teamsSystem.WriteChatMessage(ctx, logger, nk, userId, &request)
		if err != nil {
			return "", err
		}

		data, err := proto.Marshal(ack)
		if err != nil {
			logger.Error("Failed to marshal channel message ack: %v", err)
			return "", runtime.NewError("failed to marshal channel message ack", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

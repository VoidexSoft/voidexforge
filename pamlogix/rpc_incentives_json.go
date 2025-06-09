package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/heroiclabs/nakama-common/runtime"
)

// rpcIncentivesSenderListJson returns all incentives created by the user in JSON format
func rpcIncentivesSenderListJson(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		incentivesSystem := p.GetIncentivesSystem()
		if incentivesSystem == nil {
			return "", runtime.NewError("incentives system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		incentives, err := incentivesSystem.SenderList(ctx, logger, nk, userId)
		if err != nil {
			return "", err
		}

		response := struct {
			Incentives []*Incentive `json:"incentives"`
		}{
			Incentives: incentives,
		}

		data, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal incentives: %v", err)
			return "", runtime.NewError("failed to marshal incentives", INTERNAL_ERROR_CODE) // INTERNAL
		}
		return string(data), nil
	}
}

// rpcIncentivesSenderCreateJson creates a new incentive for the user using JSON
func rpcIncentivesSenderCreateJson(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		incentivesSystem := p.GetIncentivesSystem()
		if incentivesSystem == nil {
			return "", runtime.NewError("incentives system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		request := &IncentiveSenderCreateRequest{}
		if err := json.Unmarshal([]byte(payload), request); err != nil {
			logger.Error("Failed to unmarshal incentive create request: %v", err)
			return "", runtime.NewError("failed to unmarshal incentive create request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if request.Id == "" {
			return "", runtime.NewError("id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		incentives, err := incentivesSystem.SenderCreate(ctx, logger, nk, userId, request.Id)
		if err != nil {
			return "", err
		}

		response := struct {
			Incentives []*Incentive `json:"incentives"`
		}{
			Incentives: incentives,
		}

		data, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal incentives: %v", err)
			return "", runtime.NewError("failed to marshal incentives", INTERNAL_ERROR_CODE) // INTERNAL
		}
		return string(data), nil
	}
}

// rpcIncentivesSenderDeleteJson deletes an incentive created by the user using JSON
func rpcIncentivesSenderDeleteJson(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		incentivesSystem := p.GetIncentivesSystem()
		if incentivesSystem == nil {
			return "", runtime.NewError("incentives system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		request := &IncentiveSenderDeleteRequest{}
		if err := json.Unmarshal([]byte(payload), request); err != nil {
			logger.Error("Failed to unmarshal incentive delete request: %v", err)
			return "", runtime.NewError("failed to unmarshal incentive delete request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if request.Code == "" {
			return "", runtime.NewError("code is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		incentives, err := incentivesSystem.SenderDelete(ctx, logger, nk, userId, request.Code)
		if err != nil {
			return "", err
		}

		response := struct {
			Incentives []*Incentive `json:"incentives"`
		}{
			Incentives: incentives,
		}

		data, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal incentives: %v", err)
			return "", runtime.NewError("failed to marshal incentives", INTERNAL_ERROR_CODE) // INTERNAL
		}
		return string(data), nil
	}
}

// rpcIncentivesSenderClaimJson allows the incentive creator to claim rewards for recipients using JSON
func rpcIncentivesSenderClaimJson(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		incentivesSystem := p.GetIncentivesSystem()
		if incentivesSystem == nil {
			return "", runtime.NewError("incentives system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		request := &IncentiveSenderClaimRequest{}
		if err := json.Unmarshal([]byte(payload), request); err != nil {
			logger.Error("Failed to unmarshal incentive claim request: %v", err)
			return "", runtime.NewError("failed to unmarshal incentive claim request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if request.Code == "" {
			return "", runtime.NewError("code is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		incentives, err := incentivesSystem.SenderClaim(ctx, logger, nk, userId, request.Code, request.RecipientIds)
		if err != nil {
			return "", err
		}

		response := struct {
			Incentives []*Incentive `json:"incentives"`
		}{
			Incentives: incentives,
		}

		data, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal incentives: %v", err)
			return "", runtime.NewError("failed to marshal incentives", INTERNAL_ERROR_CODE) // INTERNAL
		}
		return string(data), nil
	}
}

// rpcIncentivesRecipientGetJson allows a potential recipient to view information about an incentive using JSON
func rpcIncentivesRecipientGetJson(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		incentivesSystem := p.GetIncentivesSystem()
		if incentivesSystem == nil {
			return "", runtime.NewError("incentives system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		request := &IncentiveRecipientGetRequest{}
		if err := json.Unmarshal([]byte(payload), request); err != nil {
			logger.Error("Failed to unmarshal incentive get request: %v", err)
			return "", runtime.NewError("failed to unmarshal incentive get request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if request.Code == "" {
			return "", runtime.NewError("code is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		incentive, err := incentivesSystem.RecipientGet(ctx, logger, nk, userId, request.Code)
		if err != nil {
			return "", err
		}

		data, err := json.Marshal(incentive)
		if err != nil {
			logger.Error("Failed to marshal incentive info: %v", err)
			return "", runtime.NewError("failed to marshal incentive info", INTERNAL_ERROR_CODE) // INTERNAL
		}
		return string(data), nil
	}
}

// rpcIncentivesRecipientClaimJson allows a user to claim an incentive and receive rewards using JSON
func rpcIncentivesRecipientClaimJson(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		incentivesSystem := p.GetIncentivesSystem()
		if incentivesSystem == nil {
			return "", runtime.NewError("incentives system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		request := &IncentiveRecipientClaimRequest{}
		if err := json.Unmarshal([]byte(payload), request); err != nil {
			logger.Error("Failed to unmarshal incentive claim request: %v", err)
			return "", runtime.NewError("failed to unmarshal incentive claim request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if request.Code == "" {
			return "", runtime.NewError("code is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		incentive, err := incentivesSystem.RecipientClaim(ctx, logger, nk, userId, request.Code)
		if err != nil {
			return "", err
		}

		data, err := json.Marshal(incentive)
		if err != nil {
			logger.Error("Failed to marshal incentive info: %v", err)
			return "", runtime.NewError("failed to marshal incentive info", INTERNAL_ERROR_CODE) // INTERNAL
		}
		return string(data), nil
	}
}

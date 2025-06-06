package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/heroiclabs/nakama-common/runtime"
)

// rpcIncentivesSenderList returns all incentives created by the user
func rpcIncentivesSenderList(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
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

		data, err := json.Marshal(incentives)
		if err != nil {
			logger.Error("Failed to marshal incentives: %v", err)
			return "", runtime.NewError("failed to marshal incentives", INTERNAL_ERROR_CODE) // INTERNAL
		}
		return string(data), nil
	}
}

// rpcIncentivesSenderCreate creates a new incentive for the user
func rpcIncentivesSenderCreate(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		incentivesSystem := p.GetIncentivesSystem()
		if incentivesSystem == nil {
			return "", runtime.NewError("incentives system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var req struct {
			IncentiveID string `json:"incentive_id"`
		}
		if err := json.Unmarshal([]byte(payload), &req); err != nil {
			logger.Error("Failed to unmarshal incentive create request: %v", err)
			return "", runtime.NewError("failed to unmarshal incentive create request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if req.IncentiveID == "" {
			return "", runtime.NewError("incentive_id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		incentives, err := incentivesSystem.SenderCreate(ctx, logger, nk, userId, req.IncentiveID)
		if err != nil {
			return "", err
		}

		data, err := json.Marshal(incentives)
		if err != nil {
			logger.Error("Failed to marshal incentives: %v", err)
			return "", runtime.NewError("failed to marshal incentives", INTERNAL_ERROR_CODE) // INTERNAL
		}
		return string(data), nil
	}
}

// rpcIncentivesSenderDelete deletes an incentive created by the user
func rpcIncentivesSenderDelete(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		incentivesSystem := p.GetIncentivesSystem()
		if incentivesSystem == nil {
			return "", runtime.NewError("incentives system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var req struct {
			Code string `json:"code"`
		}
		if err := json.Unmarshal([]byte(payload), &req); err != nil {
			logger.Error("Failed to unmarshal incentive delete request: %v", err)
			return "", runtime.NewError("failed to unmarshal incentive delete request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if req.Code == "" {
			return "", runtime.NewError("code is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		incentives, err := incentivesSystem.SenderDelete(ctx, logger, nk, userId, req.Code)
		if err != nil {
			return "", err
		}

		data, err := json.Marshal(incentives)
		if err != nil {
			logger.Error("Failed to marshal incentives: %v", err)
			return "", runtime.NewError("failed to marshal incentives", INTERNAL_ERROR_CODE) // INTERNAL
		}
		return string(data), nil
	}
}

// rpcIncentivesSenderClaim allows the incentive creator to claim rewards for recipients
func rpcIncentivesSenderClaim(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		incentivesSystem := p.GetIncentivesSystem()
		if incentivesSystem == nil {
			return "", runtime.NewError("incentives system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var req struct {
			Code        string   `json:"code"`
			ClaimantIDs []string `json:"claimant_ids,omitempty"`
		}
		if err := json.Unmarshal([]byte(payload), &req); err != nil {
			logger.Error("Failed to unmarshal incentive claim request: %v", err)
			return "", runtime.NewError("failed to unmarshal incentive claim request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if req.Code == "" {
			return "", runtime.NewError("code is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		incentives, err := incentivesSystem.SenderClaim(ctx, logger, nk, userId, req.Code, req.ClaimantIDs)
		if err != nil {
			return "", err
		}

		data, err := json.Marshal(incentives)
		if err != nil {
			logger.Error("Failed to marshal incentives: %v", err)
			return "", runtime.NewError("failed to marshal incentives", INTERNAL_ERROR_CODE) // INTERNAL
		}
		return string(data), nil
	}
}

// rpcIncentivesRecipientGet allows a potential recipient to view information about an incentive
func rpcIncentivesRecipientGet(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		incentivesSystem := p.GetIncentivesSystem()
		if incentivesSystem == nil {
			return "", runtime.NewError("incentives system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var req struct {
			Code string `json:"code"`
		}
		if err := json.Unmarshal([]byte(payload), &req); err != nil {
			logger.Error("Failed to unmarshal incentive get request: %v", err)
			return "", runtime.NewError("failed to unmarshal incentive get request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if req.Code == "" {
			return "", runtime.NewError("code is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		incentive, err := incentivesSystem.RecipientGet(ctx, logger, nk, userId, req.Code)
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

// rpcIncentivesRecipientClaim allows a user to claim an incentive and receive rewards
func rpcIncentivesRecipientClaim(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		incentivesSystem := p.GetIncentivesSystem()
		if incentivesSystem == nil {
			return "", runtime.NewError("incentives system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var req struct {
			Code string `json:"code"`
		}
		if err := json.Unmarshal([]byte(payload), &req); err != nil {
			logger.Error("Failed to unmarshal incentive claim request: %v", err)
			return "", runtime.NewError("failed to unmarshal incentive claim request", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		if req.Code == "" {
			return "", runtime.NewError("code is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		incentive, err := incentivesSystem.RecipientClaim(ctx, logger, nk, userId, req.Code)
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

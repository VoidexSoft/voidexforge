package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/heroiclabs/nakama-common/runtime"
)

func rpcEnergyGet_Json(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		energySystem := p.GetEnergySystem()
		if energySystem == nil {
			return "", runtime.NewError("energy system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		energies, err := energySystem.Get(ctx, logger, nk, userId)
		if err != nil {
			return "", err
		}

		// Convert to JSON EnergyList
		response := &EnergyList{
			Energies: energies,
		}

		// Marshal response to JSON
		data, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal energies: %v", err)
			return "", runtime.NewError("failed to marshal energies", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcEnergySpend_Json(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		energySystem := p.GetEnergySystem()
		if energySystem == nil {
			return "", runtime.NewError("energy system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		// Parse the input request
		request := &EnergySpendRequest{}
		if err := json.Unmarshal([]byte(payload), request); err != nil {
			logger.Error("Failed to unmarshal EnergySpendRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal energy spend request", INTERNAL_ERROR_CODE) // INTERNAL
		}

		// Call the energy system to spend the energy
		energies, reward, err := energySystem.Spend(ctx, logger, nk, userId, request.Amounts)
		if err != nil {
			return "", err
		}

		// Create response with energies and reward
		response := &EnergySpendReward{
			Energies: &EnergyList{
				Energies: energies,
			},
			Reward: reward,
		}

		// Marshal response to JSON
		data, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal energy response: %v", err)
			return "", runtime.NewError("failed to marshal energy response", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcEnergyGrant_Json(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		energySystem := p.GetEnergySystem()
		if energySystem == nil {
			return "", runtime.NewError("energy system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		// Parse the input request
		request := &EnergyGrantRequest{}
		if err := json.Unmarshal([]byte(payload), request); err != nil {
			logger.Error("Failed to unmarshal EnergyGrantRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal energy grant request", INTERNAL_ERROR_CODE) // INTERNAL
		}

		// Call the energy system to grant the energy
		energies, err := energySystem.Grant(ctx, logger, nk, userId, request.Amounts, request.Modifiers)
		if err != nil {
			logger.Error("Failed to grant energy: %v", err)
			return "", err
		}

		// Create response with energies
		response := &EnergyList{
			Energies: energies,
		}

		// Marshal the response
		data, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal energies: %v", err)
			return "", runtime.NewError("failed to marshal energies", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

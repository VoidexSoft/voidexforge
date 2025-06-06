package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/heroiclabs/nakama-common/runtime"
)

func rpcEnergyGet(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
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

		data, err := json.Marshal(energies)
		if err != nil {
			logger.Error("Failed to marshal energies: %v", err)
			return "", runtime.NewError("failed to marshal energies", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcEnergySpend(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		energySystem := p.GetEnergySystem()
		if energySystem == nil {
			return "", runtime.NewError("energy system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var request struct {
			Energies []struct {
				EnergyId string `json:"energy_id"`
				Amount   int64  `json:"amount"`
			} `json:"energies"`
		}

		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal EnergySpendRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal energy spend request", INTERNAL_ERROR_CODE) // INTERNAL
		}

		amounts := make(map[string]int32)
		for _, energy := range request.Energies {
			amounts[energy.EnergyId] = int32(energy.Amount)
		}

		// Call the energy system to spend the energy
		energies, reward, err := energySystem.Spend(ctx, logger, nk, userId, amounts)
		if err != nil {
			return "", err
		}

		// Create response with energies and reward
		response := struct {
			Energies map[string]*Energy `json:"energies"`
			Reward   *Reward            `json:"reward,omitempty"`
		}{
			Energies: energies,
			Reward:   reward,
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

func rpcEnergyGrant(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		energySystem := p.GetEnergySystem()
		if energySystem == nil {
			return "", runtime.NewError("energy system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
		}

		var request struct {
			Energies []struct {
				EnergyId string `json:"energy_id"`
				Amount   int64  `json:"amount"`
			} `json:"energies"`
			Modifiers []*RewardEnergyModifier `json:"modifiers,omitempty"`
		}

		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal EnergyGrantRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal energy grant request", INTERNAL_ERROR_CODE) // INTERNAL
		}

		// Convert to the format expected by the Grant method
		amounts := make(map[string]int32)
		for _, energy := range request.Energies {
			amounts[energy.EnergyId] = int32(energy.Amount)
		}

		// Call the energy system to grant the energy
		energies, err := energySystem.Grant(ctx, logger, nk, userId, amounts, request.Modifiers)
		if err != nil {
			return "", err
		}

		// Marshal the response
		data, err := json.Marshal(energies)
		if err != nil {
			logger.Error("Failed to marshal energies: %v", err)
			return "", runtime.NewError("failed to marshal energies", INTERNAL_ERROR_CODE) // INTERNAL
		}

		return string(data), nil
	}
}

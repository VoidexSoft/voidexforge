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
			return "", runtime.NewError("energy system not available", 12) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", 3) // INVALID_ARGUMENT
		}

		energies, err := energySystem.Get(ctx, logger, nk, userId)
		if err != nil {
			return "", err
		}

		data, err := json.Marshal(energies)
		if err != nil {
			logger.Error("Failed to marshal energies: %v", err)
			return "", runtime.NewError("failed to marshal energies", 13) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcEnergySpend(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		energySystem := p.GetEnergySystem()
		if energySystem == nil {
			return "", runtime.NewError("energy system not available", 12) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", 3) // INVALID_ARGUMENT
		}

		var request struct {
			Energies []struct {
				EnergyId string `json:"energy_id"`
				Amount   int64  `json:"amount"`
			} `json:"energies"`
		}

		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal EnergySpendRequest: %v", err)
			return "", runtime.NewError("failed to unmarshal energy spend request", 13) // INTERNAL
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
			return "", runtime.NewError("failed to marshal energy response", 13) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcEnergyGrant(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		energySystem := p.GetEnergySystem()
		if energySystem == nil {
			return "", runtime.NewError("energy system not available", 12) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", 3) // INVALID_ARGUMENT
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
			return "", runtime.NewError("failed to unmarshal energy grant request", 13) // INTERNAL
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
			return "", runtime.NewError("failed to marshal energies", 13) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcInventoryList(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		inventorySystem := p.GetInventorySystem()
		if inventorySystem == nil {
			return "", runtime.NewError("inventory system not available", 12) // UNIMPLEMENTED
		}

		var request struct {
			Category string `json:"category,omitempty"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal InventoryListRequest: %v", err)
			return "", ErrPayloadDecode
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", ErrNoSessionUser
		}

		items, itemSets, err := inventorySystem.List(ctx, logger, nk, userID, request.Category)
		if err != nil {
			logger.Error("Error listing inventory items: %v", err)
			return "", err
		}

		// Convert maps to slices for response
		itemsList := make([]*InventoryConfigItem, 0, len(items))
		for _, item := range items {
			itemsList = append(itemsList, item)
		}

		response := struct {
			Items    []*InventoryConfigItem `json:"items"`
			ItemSets map[string][]string    `json:"item_sets"`
		}{
			Items:    itemsList,
			ItemSets: itemSets,
		}

		responseData, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

func rpcInventoryListInventory(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		inventorySystem := p.GetInventorySystem()
		if inventorySystem == nil {
			return "", runtime.NewError("inventory system not available", 12) // UNIMPLEMENTED
		}

		var request struct {
			Category string `json:"category,omitempty"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal InventoryListInventoryRequest: %v", err)
			return "", ErrPayloadDecode
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", ErrNoSessionUser
		}

		inventory, err := inventorySystem.ListInventoryItems(ctx, logger, nk, userID, request.Category)
		if err != nil {
			logger.Error("Error listing inventory items: %v", err)
			return "", err
		}

		responseData, err := json.Marshal(inventory)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

func rpcInventoryConsume(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		// Implementation to be added when needed
		return "", runtime.NewError("not implemented", 12) // UNIMPLEMENTED
	}
}

func rpcInventoryGrant(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		// Implementation to be added when needed
		return "", runtime.NewError("not implemented", 12) // UNIMPLEMENTED
	}
}

func rpcInventoryUpdate(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		// Implementation to be added when needed
		return "", runtime.NewError("not implemented", 12) // UNIMPLEMENTED
	}
}

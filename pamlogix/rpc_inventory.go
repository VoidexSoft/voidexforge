package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/heroiclabs/nakama-common/runtime"
)

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
		itemsList := make([]*inventory.InventoryConfigItem, 0, len(items))
		for _, item := range items {
			itemsList = append(itemsList, item)
		}

		response := struct {
			Items    []*inventory.InventoryConfigItem `json:"items"`
			ItemSets map[string][]string              `json:"item_sets"`
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

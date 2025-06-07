package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/heroiclabs/nakama-common/runtime"
)

func rpcJsonInventoryList(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		inventorySystem := p.GetInventorySystem()
		if inventorySystem == nil {
			return "", runtime.NewError("inventory system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
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

func rpcJsonInventoryListInventory(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		inventorySystem := p.GetInventorySystem()
		if inventorySystem == nil {
			return "", runtime.NewError("inventory system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
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

		inventoryItems, err := inventorySystem.ListInventoryItems(ctx, logger, nk, userID, request.Category)
		if err != nil {
			logger.Error("Error listing inventory items: %v", err)
			return "", err
		}

		responseData, err := json.Marshal(inventoryItems)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

func rpcJsonInventoryConsume(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		inventorySystem := p.GetInventorySystem()
		if inventorySystem == nil {
			return "", runtime.NewError("inventory system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		var request InventoryConsumeRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal InventoryConsumeRequest: %v", err)
			return "", ErrPayloadDecode
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", ErrNoSessionUser
		}

		// Convert request items and instances to the expected format
		itemIDs := make(map[string]int64)
		if request.Items != nil {
			for itemID, count := range request.Items {
				itemIDs[itemID] = count
			}
		}

		instanceIDs := make(map[string]int64)
		if request.Instances != nil {
			for instanceID, count := range request.Instances {
				instanceIDs[instanceID] = count
			}
		}

		// Consume the items
		updatedInventory, rewards, instanceRewards, err := inventorySystem.ConsumeItems(ctx, logger, nk, userID, itemIDs, instanceIDs, request.Overconsume)
		if err != nil {
			logger.Error("Error consuming inventory items: %v", err)
			return "", err
		}

		// Convert rewards to response format
		responseRewards := make(map[string]*RewardList)
		for itemID, rewardList := range rewards {
			responseRewards[itemID] = &RewardList{
				Rewards: rewardList,
			}
		}

		responseInstanceRewards := make(map[string]*RewardList)
		for instanceID, rewardList := range instanceRewards {
			responseInstanceRewards[instanceID] = &RewardList{
				Rewards: rewardList,
			}
		}

		response := &InventoryConsumeRewards{
			Inventory:       updatedInventory,
			Rewards:         responseRewards,
			InstanceRewards: responseInstanceRewards,
		}

		responseData, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

func rpcJsonInventoryGrant(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		inventorySystem := p.GetInventorySystem()
		if inventorySystem == nil {
			return "", runtime.NewError("inventory system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		var request InventoryGrantRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal InventoryGrantRequest: %v", err)
			return "", ErrPayloadDecode
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", ErrNoSessionUser
		}

		// Grant the items (ignoreLimits defaults to false for user-initiated requests)
		updatedInventory, _, _, _, err := inventorySystem.GrantItems(ctx, logger, nk, userID, request.Items, false)
		if err != nil {
			logger.Error("Error granting inventory items: %v", err)
			return "", err
		}
		//TODO: return all outcomes, not just updated inventory
		response := &InventoryUpdateAck{
			Inventory: updatedInventory,
		}

		responseData, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

func rpcJsonInventoryUpdate(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		inventorySystem := p.GetInventorySystem()
		if inventorySystem == nil {
			return "", runtime.NewError("inventory system not available", UNIMPLEMENTED_ERROR_CODE) // UNIMPLEMENTED
		}

		var request InventoryUpdateItemsRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal InventoryUpdateItemsRequest: %v", err)
			return "", ErrPayloadDecode
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", ErrNoSessionUser
		}

		// Convert request item updates to the expected format
		instanceIDs := make(map[string]*InventoryUpdateItemProperties)
		if request.ItemUpdates != nil {
			for instanceID, updateProps := range request.ItemUpdates {
				instanceIDs[instanceID] = updateProps
			}
		}

		// Update the items
		updatedInventory, err := inventorySystem.UpdateItems(ctx, logger, nk, userID, instanceIDs)
		if err != nil {
			logger.Error("Error updating inventory items: %v", err)
			return "", err
		}

		response := &InventoryUpdateAck{
			Inventory: updatedInventory,
		}

		responseData, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

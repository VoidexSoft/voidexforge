package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
)

// Economy system RPC handlers
func rpcEconomyDonationClaim(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		var request struct {
			DonationClaims map[string]*EconomyDonationClaimRequestDetails `json:"donation_claims"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal EconomyDonationClaimRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Call the economy system to claim donations
		donationsList, err := p.GetEconomySystem().DonationClaim(ctx, logger, nk, userID, request.DonationClaims)
		if err != nil {
			logger.Error("Error claiming donations: %v", err)
			return "", err
		}

		// Encode the response
		responseData, err := json.Marshal(donationsList)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

func rpcEconomyDonationGive(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		var request struct {
			UserId     string `json:"user_id"`
			DonationId string `json:"donation_id"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal EconomyDonationGiveRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session
		fromUserID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || fromUserID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Call the economy system to give a donation
		donation, updatedWallet, updatedInventory, rewardModifiers, contributorReward, timestamp, err := p.GetEconomySystem().DonationGive(ctx, logger, nk, request.UserId, request.DonationId, fromUserID)
		if err != nil {
			logger.Error("Error giving donation: %v", err)
			return "", err
		}

		// Convert inventory.Items to map for the response
		inventoryMap := make(map[string]*InventoryItem)
		if updatedInventory != nil && updatedInventory.Items != nil {
			inventoryMap = updatedInventory.Items
		}

		// Convert ActiveRewardModifier to map for the response
		rewardModifiersMap := make(map[string]*ActiveRewardModifier)
		for i, modifier := range rewardModifiers {
			rewardModifiersMap[modifier.Id] = rewardModifiers[i]
		}

		// Prepare the response
		response := struct {
			Donation        *EconomyDonation                 `json:"donation"`
			Wallet          map[string]int64                 `json:"wallet"`
			Inventory       map[string]*InventoryItem        `json:"inventory"`
			RewardModifiers map[string]*ActiveRewardModifier `json:"reward_modifiers"`
			Reward          *Reward                          `json:"reward"`
			Timestamp       int64                            `json:"timestamp"`
		}{
			Donation:        donation,
			Wallet:          updatedWallet,
			Inventory:       inventoryMap,
			RewardModifiers: rewardModifiersMap,
			Reward:          contributorReward,
			Timestamp:       timestamp,
		}

		// Encode the response
		responseData, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

func rpcEconomyDonationGet(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		var request struct {
			UserIds []string `json:"user_ids"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal EconomyDonationGetRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Call the economy system to get donations
		donationsList, err := p.GetEconomySystem().DonationGet(ctx, logger, nk, request.UserIds)
		if err != nil {
			logger.Error("Error getting donations: %v", err)
			return "", err
		}

		// Encode the response
		responseData, err := json.Marshal(donationsList)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

func rpcEconomyDonationRequest(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		var request struct {
			DonationId string `json:"donation_id"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal EconomyDonationRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Call the economy system to request a donation
		donation, success, err := p.GetEconomySystem().DonationRequest(ctx, logger, nk, userID, request.DonationId)
		if err != nil {
			logger.Error("Error requesting donation: %v", err)
			return "", err
		}

		// Prepare the response
		response := struct {
			Donation *EconomyDonation `json:"donation"`
			Success  bool             `json:"success"`
		}{
			Donation: donation,
			Success:  success,
		}

		// Encode the response
		responseData, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

func rpcEconomyStoreGet(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		var request struct {
			UserId string `json:"user_id,omitempty"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal EconomyListRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session if not provided in the request
		userID := request.UserId
		if userID == "" {
			var ok bool
			userID, ok = ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
			if !ok {
				logger.Warn("No user ID in context or request")
				// Allow continuing without a user ID
			}
		}

		// Call the economy system to get store items and placements
		storeItems, placements, rewardModifiers, timestamp, err := p.GetEconomySystem().List(ctx, logger, nk, userID)
		if err != nil {
			logger.Error("Error listing economy items: %v", err)
			return "", err
		}

		// Convert map to slice for response
		storeItemsSlice := make([]*EconomyConfigStoreItem, 0, len(storeItems))
		for _, item := range storeItems {
			storeItemsSlice = append(storeItemsSlice, item)
		}

		placementsSlice := make([]*EconomyConfigPlacement, 0, len(placements))
		for _, placement := range placements {
			placementsSlice = append(placementsSlice, placement)
		}

		// Convert ActiveRewardModifier slice to map for the response
		rewardModifiersMap := make(map[string]*ActiveRewardModifier)
		for i, modifier := range rewardModifiers {
			rewardModifiersMap[modifier.Id] = rewardModifiers[i]
		}

		// Prepare the response
		response := struct {
			StoreItems      []*EconomyConfigStoreItem        `json:"store_items"`
			Placements      []*EconomyConfigPlacement        `json:"placements"`
			RewardModifiers map[string]*ActiveRewardModifier `json:"reward_modifiers,omitempty"`
			Timestamp       int64                            `json:"timestamp"`
		}{
			StoreItems:      storeItemsSlice,
			Placements:      placementsSlice,
			RewardModifiers: rewardModifiersMap,
			Timestamp:       timestamp,
		}

		// Encode the response
		responseData, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

func rpcEconomyGrant(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		var request struct {
			UserId         string                 `json:"user_id,omitempty"`
			Currencies     map[string]int64       `json:"currencies"`
			Items          map[string]int32       `json:"items"`
			Modifiers      []*RewardModifier      `json:"modifiers,omitempty"`
			WalletMetadata map[string]interface{} `json:"wallet_metadata,omitempty"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal EconomyGrantRequest: %v", err)
			return "", runtime.NewError("Failed to unmarshal EconomyGrantRequest: "+err.Error(), 13)
		}

		// Extract user ID from session if not provided in the request
		userID := request.UserId
		if userID == "" {
			var ok bool
			userID, ok = ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
			if !ok || userID == "" {
				logger.Error("No user ID in context or request")
				return "", ErrNoSessionUser
			}
		}

		// Convert int32 to int64 for items
		itemsInt64 := make(map[string]int64)
		for k, v := range request.Items {
			itemsInt64[k] = int64(v)
		}

		// Call the economy system to grant currencies/items
		updatedWallet, rewardModifiers, timestamp, err := p.GetEconomySystem().Grant(ctx, logger, nk, userID, request.Currencies, itemsInt64, request.Modifiers, request.WalletMetadata)
		if err != nil {
			logger.Error("Error granting economy items: %v", err)
			return "", err
		}

		// Convert ActiveRewardModifier slice to map for the response
		rewardModifiersMap := make(map[string]*ActiveRewardModifier)
		for i, modifier := range rewardModifiers {
			rewardModifiersMap[modifier.Id] = rewardModifiers[i]
		}

		// Prepare the response
		response := struct {
			Wallet          map[string]int64                 `json:"wallet"`
			RewardModifiers map[string]*ActiveRewardModifier `json:"reward_modifiers,omitempty"`
			Timestamp       int64                            `json:"timestamp"`
		}{
			Wallet:          updatedWallet,
			RewardModifiers: rewardModifiersMap,
			Timestamp:       timestamp,
		}

		// Encode the response
		responseData, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

func rpcEconomyPurchaseIntent(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		var request struct {
			ItemId string `json:"item_id"`
			Store  string `json:"store,omitempty"`
			Sku    string `json:"sku,omitempty"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal EconomyPurchaseIntentRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Convert string store type to EconomyStoreType enum
		var storeType EconomyStoreType
		switch request.Store {
		case "apple", "APPLE_APPSTORE":
			storeType = EconomyStoreType_ECONOMY_STORE_TYPE_APPLE_APPSTORE
		case "google", "GOOGLE_PLAY":
			storeType = EconomyStoreType_ECONOMY_STORE_TYPE_GOOGLE_PLAY
		case "facebook", "FBINSTANT":
			storeType = EconomyStoreType_ECONOMY_STORE_TYPE_FBINSTANT
		case "discord", "DISCORD":
			storeType = EconomyStoreType_ECONOMY_STORE_TYPE_DISCORD
		default:
			storeType = EconomyStoreType_ECONOMY_STORE_TYPE_UNSPECIFIED
		}

		// Call the economy system to create a purchase intent
		err := p.GetEconomySystem().PurchaseIntent(ctx, logger, nk, userID, request.ItemId, storeType, request.Sku)
		if err != nil {
			logger.Error("Error creating purchase intent: %v", err)
			return "", err
		}

		// No response body for this endpoint
		return "", nil
	}
}

func rpcEconomyPurchaseItem(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		var request struct {
			ItemId  string `json:"item_id"`
			Store   string `json:"store,omitempty"`
			Receipt string `json:"receipt"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal EconomyPurchaseRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Convert string store type to EconomyStoreType enum
		var storeType EconomyStoreType
		switch request.Store {
		case "apple", "APPLE_APPSTORE":
			storeType = EconomyStoreType_ECONOMY_STORE_TYPE_APPLE_APPSTORE
		case "google", "GOOGLE_PLAY":
			storeType = EconomyStoreType_ECONOMY_STORE_TYPE_GOOGLE_PLAY
		case "facebook", "FBINSTANT":
			storeType = EconomyStoreType_ECONOMY_STORE_TYPE_FBINSTANT
		case "discord", "DISCORD":
			storeType = EconomyStoreType_ECONOMY_STORE_TYPE_DISCORD
		default:
			storeType = EconomyStoreType_ECONOMY_STORE_TYPE_UNSPECIFIED
		}

		// Call the economy system to process a purchase
		updatedWallet, updatedInventory, reward, isSandboxPurchase, err := p.GetEconomySystem().PurchaseItem(ctx, logger, db, nk, userID, request.ItemId, storeType, request.Receipt)
		if err != nil {
			logger.Error("Error processing purchase: %v", err)
			return "", err
		}

		// Convert inventory.Items to map for the response
		inventoryMap := make(map[string]*InventoryItem)
		if updatedInventory != nil && updatedInventory.Items != nil {
			inventoryMap = updatedInventory.Items
		}

		// Prepare the response
		response := struct {
			Wallet            map[string]int64          `json:"wallet"`
			Inventory         map[string]*InventoryItem `json:"inventory"`
			Reward            *Reward                   `json:"reward"`
			IsSandboxPurchase bool                      `json:"is_sandbox_purchase"`
			Timestamp         int64                     `json:"timestamp"`
		}{
			Wallet:            updatedWallet,
			Inventory:         inventoryMap,
			Reward:            reward,
			IsSandboxPurchase: isSandboxPurchase,
			Timestamp:         time.Now().Unix(),
		}

		// Encode the response
		responseData, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

func rpcEconomyPurchaseRestore(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		var request struct {
			Store    string   `json:"store,omitempty"`
			Receipts []string `json:"receipts"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal EconomyPurchaseRestoreRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Convert string store type to EconomyStoreType enum
		var storeType EconomyStoreType
		switch request.Store {
		case "apple", "APPLE_APPSTORE":
			storeType = EconomyStoreType_ECONOMY_STORE_TYPE_APPLE_APPSTORE
		case "google", "GOOGLE_PLAY":
			storeType = EconomyStoreType_ECONOMY_STORE_TYPE_GOOGLE_PLAY
		case "facebook", "FBINSTANT":
			storeType = EconomyStoreType_ECONOMY_STORE_TYPE_FBINSTANT
		case "discord", "DISCORD":
			storeType = EconomyStoreType_ECONOMY_STORE_TYPE_DISCORD
		default:
			storeType = EconomyStoreType_ECONOMY_STORE_TYPE_UNSPECIFIED
		}

		// Call the economy system to restore purchases
		err := p.GetEconomySystem().PurchaseRestore(ctx, logger, nk, userID, storeType, request.Receipts)
		if err != nil {
			logger.Error("Error restoring purchases: %v", err)
			return "", err
		}

		// No response body for this endpoint
		return "", nil
	}
}

func rpcEconomyPlacementStatus(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		var request struct {
			RewardId    string `json:"reward_id"`
			PlacementId string `json:"placement_id"`
			RetryCount  int32  `json:"retry_count,omitempty"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal EconomyPlacementStatusRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Call the economy system to get placement status
		status, err := p.GetEconomySystem().PlacementStatus(ctx, logger, nk, userID, request.RewardId, request.PlacementId, int(request.RetryCount))
		if err != nil {
			logger.Error("Error getting placement status: %v", err)
			return "", err
		}

		// Encode the response
		responseData, err := json.Marshal(status)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

func rpcEconomyPlacementStart(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		var request struct {
			PlacementId string                 `json:"placement_id"`
			Metadata    map[string]interface{} `json:"metadata,omitempty"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal EconomyPlacementStartRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Convert interface{} metadata to string metadata
		metadataStr := make(map[string]string)
		for k, v := range request.Metadata {
			// Convert the value to string
			switch val := v.(type) {
			case string:
				metadataStr[k] = val
			default:
				// For non-string values, try to JSON marshal them
				bytes, err := json.Marshal(val)
				if err != nil {
					logger.Warn("Failed to marshal metadata value for key %s: %v", k, err)
					continue
				}
				metadataStr[k] = string(bytes)
			}
		}

		// Call the economy system to start a placement
		status, err := p.GetEconomySystem().PlacementStart(ctx, logger, nk, userID, request.PlacementId, metadataStr)
		if err != nil {
			logger.Error("Error starting placement: %v", err)
			return "", err
		}

		// Encode the response
		responseData, err := json.Marshal(status)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

func rpcEconomyPlacementSuccess(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		var request struct {
			RewardId    string `json:"reward_id"`
			PlacementId string `json:"placement_id"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal EconomyPlacementSuccessRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Call the economy system to report a successful placement view
		reward, placementMetadata, err := p.GetEconomySystem().PlacementSuccess(ctx, logger, nk, userID, request.RewardId, request.PlacementId)
		if err != nil {
			logger.Error("Error reporting placement success: %v", err)
			return "", err
		}

		// Convert string metadata to interface{} metadata for JSON response
		metadataInterface := make(map[string]interface{})
		for k, v := range placementMetadata {
			metadataInterface[k] = v
		}

		// Prepare the response
		response := struct {
			Reward    *Reward                `json:"reward"`
			Metadata  map[string]interface{} `json:"metadata,omitempty"`
			Timestamp int64                  `json:"timestamp"`
		}{
			Reward:    reward,
			Metadata:  metadataInterface,
			Timestamp: time.Now().Unix(),
		}

		// Encode the response
		responseData, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

func rpcEconomyPlacementFail(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		var request struct {
			RewardId    string `json:"reward_id"`
			PlacementId string `json:"placement_id"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal EconomyPlacementFailRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Call the economy system to report a failed placement view
		placementMetadata, err := p.GetEconomySystem().PlacementFail(ctx, logger, nk, userID, request.RewardId, request.PlacementId)
		if err != nil {
			logger.Error("Error reporting placement failure: %v", err)
			return "", err
		}

		// Convert string metadata to interface{} metadata for JSON response
		metadataInterface := make(map[string]interface{})
		for k, v := range placementMetadata {
			metadataInterface[k] = v
		}

		// Prepare the response
		response := struct {
			Metadata  map[string]interface{} `json:"metadata,omitempty"`
			Timestamp int64                  `json:"timestamp"`
		}{
			Metadata:  metadataInterface,
			Timestamp: time.Now().Unix(),
		}

		// Encode the response
		responseData, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

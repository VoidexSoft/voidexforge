package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
)

// Economy system RPC handlers with JSON serialization
func rpcEconomyDonationClaim_Json(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		request := &EconomyDonationClaimRequest{}
		if err := json.Unmarshal([]byte(payload), request); err != nil {
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
		donationsList, err := p.GetEconomySystem().DonationClaim(ctx, logger, nk, userID, request.Donations)
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

func rpcEconomyDonationGive_Json(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		request := &EconomyDonationGiveRequest{}
		if err := json.Unmarshal([]byte(payload), request); err != nil {
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
		_, updatedWallet, updatedInventory, rewardModifiers, contributorReward, timestamp, err := p.GetEconomySystem().DonationGive(ctx, logger, nk, request.UserId, request.DonationId, fromUserID)
		if err != nil {
			logger.Error("Error giving donation: %v", err)
			return "", err
		}

		//use EconomyUpdateAck
		response := &EconomyUpdateAck{
			Wallet:                updatedWallet,
			Inventory:             updatedInventory,
			ActiveRewardModifiers: rewardModifiers,
			Reward:                contributorReward,
			CurrentTimeSec:        timestamp,
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

func rpcEconomyDonationGet_Json(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		request := &EconomyDonationGetRequest{}
		if err := json.Unmarshal([]byte(payload), request); err != nil {
			logger.Error("Failed to unmarshal EconomyDonationGetRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Call the economy system to get donations
		donationsList, err := p.GetEconomySystem().DonationGet(ctx, logger, nk, request.Ids)
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

func rpcEconomyDonationRequest_Json(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		request := &EconomyDonationRequest{}
		if err := json.Unmarshal([]byte(payload), request); err != nil {
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

		// Prepare the response using protobuf message
		response := &EconomyDonationAck{
			Created:  success,
			Donation: donation,
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

func rpcEconomyStoreGet_Json(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		request := &EconomyListRequest{}
		if err := json.Unmarshal([]byte(payload), request); err != nil {
			logger.Error("Failed to unmarshal EconomyListRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok {
			logger.Warn("No user ID in context")
			// Allow continuing without a user ID
		}

		// Call the economy system to get store items and placements
		storeItems, placements, rewardModifiers, timestamp, err := p.GetEconomySystem().List(ctx, logger, nk, userID)
		if err != nil {
			logger.Error("Error listing economy items: %v", err)
			return "", err
		}

		// Convert map to slice for response
		storeItemsSlice := make([]*EconomyListStoreItem, 0, len(storeItems))
		for itemId, item := range storeItems {
			var cost *EconomyListStoreItemCost
			if item.Cost != nil {
				cost = &EconomyListStoreItemCost{
					Currencies: item.Cost.Currencies,
					Sku:        item.Cost.Sku,
				}
			}
			storeItemsSlice = append(storeItemsSlice, &EconomyListStoreItem{
				Id:                   itemId,
				Name:                 item.Name,
				Description:          item.Description,
				Category:             item.Category,
				Cost:                 cost,
				AdditionalProperties: item.AdditionalProperties,
				Unavailable:          item.Unavailable,
			})
		}

		placementsSlice := make([]*EconomyListPlacement, 0, len(placements))
		for placementId, placement := range placements {
			placementsSlice = append(placementsSlice, &EconomyListPlacement{
				Id:                   placementId,
				AdditionalProperties: placement.AdditionalProperties,
			})
		}

		// Prepare the response using protobuf message
		response := &EconomyList{
			StoreItems:            storeItemsSlice,
			Placements:            placementsSlice,
			ActiveRewardModifiers: rewardModifiers,
			CurrentTimeSec:        timestamp,
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

func rpcEconomyGrant_Json(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		request := &EconomyGrantRequest{}
		if err := json.Unmarshal([]byte(payload), request); err != nil {
			logger.Error("Failed to unmarshal EconomyGrantRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Call the economy system to grant currencies and reward modifiers
		updatedWallet, rewardModifiers, timestamp, err := p.GetEconomySystem().Grant(ctx, logger, nk, userID, request.Currencies, request.Items, request.RewardModifiers, nil)
		if err != nil {
			logger.Error("Error granting economy items: %v", err)
			return "", err
		}

		// use EconomyUpdateAck proto message instead
		response := &EconomyUpdateAck{
			Wallet:                updatedWallet,
			ActiveRewardModifiers: rewardModifiers,
			CurrentTimeSec:        timestamp,
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

func rpcEconomyPurchaseIntent_Json(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		request := &EconomyPurchaseIntentRequest{}
		if err := json.Unmarshal([]byte(payload), request); err != nil {
			logger.Error("Failed to unmarshal EconomyPurchaseIntentRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Call the economy system to create purchase intent
		err := p.GetEconomySystem().PurchaseIntent(ctx, logger, nk, userID, request.ItemId, request.StoreType, request.Sku)
		if err != nil {
			logger.Error("Error creating purchase intent: %v", err)
			return "", err
		}

		// No response body for this endpoint
		return "", nil
		//TODO: Create a protobuf message for this
	}
}

func rpcEconomyPurchaseItem_Json(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		request := &EconomyPurchaseRequest{}
		if err := json.Unmarshal([]byte(payload), request); err != nil {
			logger.Error("Failed to unmarshal EconomyPurchaseRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Call the economy system to purchase an item
		updatedWallet, updatedInventory, reward, isSandboxPurchase, err := p.GetEconomySystem().PurchaseItem(ctx, logger, db, nk, userID, request.ItemId, request.StoreType, request.Receipt)
		if err != nil {
			logger.Error("Error processing purchase: %v", err)
			return "", err
		}

		response := &EconomyPurchaseAck{
			Wallet:            updatedWallet,
			Inventory:         updatedInventory,
			Reward:            reward,
			IsSandboxPurchase: isSandboxPurchase,
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

func rpcEconomyPurchaseRestore_Json(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		request := &EconomyPurchaseRestoreRequest{}
		if err := json.Unmarshal([]byte(payload), request); err != nil {
			logger.Error("Failed to unmarshal EconomyPurchaseRestoreRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Call the economy system to restore purchases
		err := p.GetEconomySystem().PurchaseRestore(ctx, logger, nk, userID, request.StoreType, request.Receipts)
		if err != nil {
			logger.Error("Error restoring purchases: %v", err)
			return "", err
		}

		// No response body for this endpoint
		return "", nil
		//TODO: Create a protobuf message for this
	}
}

func rpcEconomyPlacementStatus_Json(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		request := &EconomyPlacementStatusRequest{}
		if err := json.Unmarshal([]byte(payload), request); err != nil {
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
		status, err := p.GetEconomySystem().PlacementStatus(ctx, logger, nk, userID, request.RewardId, request.PlacementId, int(request.Count))
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

func rpcEconomyPlacementStart_Json(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		request := &EconomyPlacementStartRequest{}
		if err := json.Unmarshal([]byte(payload), request); err != nil {
			logger.Error("Failed to unmarshal EconomyPlacementStartRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Call the economy system to start placement
		status, err := p.GetEconomySystem().PlacementStart(ctx, logger, nk, userID, request.PlacementId, request.Metadata)
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

func rpcEconomyPlacementSuccess_Json(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		//use EconomyPlacementStatusRequest
		request := &EconomyPlacementStatusRequest{}
		if err := json.Unmarshal([]byte(payload), request); err != nil {
			logger.Error("Failed to unmarshal EconomyPlacementStatusRequest: %v", err)
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
			logger.Error("Error handling placement success: %v", err)
			return "", err
		}

		// Prepare the response
		response := &EconomyPlacementStatus{
			RewardId:        request.RewardId,
			PlacementId:     request.PlacementId,
			CompleteTimeSec: time.Now().Unix(),
			Reward:          reward,
			Success:         true,
			Metadata:        placementMetadata,
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

func rpcEconomyPlacementFail_Json(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		if p.GetEconomySystem() == nil {
			return "", ErrSystemNotFound
		}

		request := &EconomyPlacementStatusRequest{}
		if err := json.Unmarshal([]byte(payload), request); err != nil {
			logger.Error("Failed to unmarshal EconomyPlacementStatusRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Call the economy system to handle placement failure
		_, err := p.GetEconomySystem().PlacementFail(ctx, logger, nk, userID, request.RewardId, request.PlacementId)
		if err != nil {
			logger.Error("Error handling placement failure: %v", err)
			return "", err
		}

		// Prepare the response
		response := &EconomyPlacementStatus{
			RewardId:        request.RewardId,
			PlacementId:     request.PlacementId,
			CompleteTimeSec: time.Now().Unix(),
			Success:         false,
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

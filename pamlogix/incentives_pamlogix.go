package pamlogix

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/heroiclabs/nakama-common/runtime"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	incentivesStorageCollection = "incentives"
	userIncentivesStorageKey    = "user_incentives"
	incentiveCodePrefix         = "INC"
)

// NakamaIncentivesSystem implements the IncentivesSystem interface using Nakama as the backend.
type NakamaIncentivesSystem struct {
	config            *IncentivesConfig
	onSenderReward    OnReward[*IncentivesConfigIncentive]
	onRecipientReward OnReward[*IncentivesConfigIncentive]
	pamlogix          Pamlogix
}

// NewNakamaIncentivesSystem creates a new instance of the incentives system with the given configuration.
func NewNakamaIncentivesSystem(config *IncentivesConfig) *NakamaIncentivesSystem {
	return &NakamaIncentivesSystem{
		config: config,
	}
}

// SetPamlogix sets the Pamlogix instance for this incentives system
func (i *NakamaIncentivesSystem) SetPamlogix(pl Pamlogix) {
	i.pamlogix = pl
}

// GetType returns the system type for the incentives system.
func (i *NakamaIncentivesSystem) GetType() SystemType {
	return SystemTypeIncentives
}

// GetConfig returns the configuration for the incentives system.
func (i *NakamaIncentivesSystem) GetConfig() any {
	return i.config
}

// SenderList returns all incentives created by the user.
func (i *NakamaIncentivesSystem) SenderList(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (incentives []*Incentive, err error) {
	// Get user's incentives from storage
	userIncentives, err := i.getUserIncentives(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user incentives: %v", err)
		return nil, runtime.NewError("failed to get user incentives", 13) // INTERNAL
	}

	// Convert to response format
	incentives = make([]*Incentive, 0, len(userIncentives))
	for _, incentive := range userIncentives {
		incentives = append(incentives, incentive)
	}

	return incentives, nil
}

// SenderCreate creates a new incentive for the user.
func (i *NakamaIncentivesSystem) SenderCreate(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, incentiveID string) (incentives []*Incentive, err error) {
	// Validate incentive ID exists in config
	if i.config == nil || i.config.Incentives == nil {
		return nil, runtime.NewError("incentives system not configured", 9) // FAILED_PRECONDITION
	}

	incentiveConfig, exists := i.config.Incentives[incentiveID]
	if !exists {
		return nil, runtime.NewError("incentive configuration not found", 5) // NOT_FOUND
	}

	// Get user's current incentives
	userIncentives, err := i.getUserIncentives(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user incentives: %v", err)
		return nil, runtime.NewError("failed to get user incentives", 13) // INTERNAL
	}

	// Check max concurrent limit
	if incentiveConfig.MaxConcurrent > 0 {
		activeCount := 0
		for _, incentive := range userIncentives {
			if incentive.Id == incentiveID && (incentive.ExpiryTimeSec == 0 || incentive.ExpiryTimeSec > time.Now().Unix()) {
				activeCount++
			}
		}
		if activeCount >= incentiveConfig.MaxConcurrent {
			return nil, runtime.NewError("maximum concurrent incentives reached", 9) // FAILED_PRECONDITION
		}
	}

	// Generate unique code
	code := i.generateIncentiveCode()

	// Create new incentive
	now := time.Now().Unix()
	var expiryTime int64
	if incentiveConfig.ExpiryDurationSec > 0 {
		expiryTime = now + incentiveConfig.ExpiryDurationSec
	}

	// Convert rewards to AvailableRewards format
	var recipientRewards, senderRewards *AvailableRewards
	if incentiveConfig.RecipientReward != nil {
		recipientRewards = i.convertRewardConfigToAvailableRewards(incentiveConfig.RecipientReward)
	}
	if incentiveConfig.SenderReward != nil {
		senderRewards = i.convertRewardConfigToAvailableRewards(incentiveConfig.SenderReward)
	}

	// Convert additional properties to protobuf Struct
	var additionalProps *structpb.Struct
	if len(incentiveConfig.AdditionalProperties) > 0 {
		additionalProps, _ = structpb.NewStruct(incentiveConfig.AdditionalProperties)
	}

	newIncentive := &Incentive{
		Id:                   incentiveID,
		Name:                 incentiveConfig.Name,
		Description:          incentiveConfig.Description,
		Code:                 code,
		Type:                 incentiveConfig.Type,
		CreateTimeSec:        now,
		UpdateTimeSec:        now,
		ExpiryTimeSec:        expiryTime,
		RecipientRewards:     recipientRewards,
		SenderRewards:        senderRewards,
		UnclaimedRecipients:  make([]string, 0),
		Rewards:              make([]*Reward, 0),
		MaxClaims:            int64(incentiveConfig.MaxClaims),
		Claims:               make(map[string]*IncentiveClaim),
		AdditionalProperties: additionalProps,
	}

	// Add to user's incentives
	userIncentives[code] = newIncentive

	// Save to storage
	err = i.saveUserIncentives(ctx, logger, nk, userID, userIncentives)
	if err != nil {
		logger.Error("Failed to save user incentives: %v", err)
		return nil, runtime.NewError("failed to save incentives", 13) // INTERNAL
	}

	// Return updated list
	return i.SenderList(ctx, logger, nk, userID)
}

// SenderDelete deletes an incentive created by the user.
func (i *NakamaIncentivesSystem) SenderDelete(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, code string) (incentives []*Incentive, err error) {
	// Get user's incentives
	userIncentives, err := i.getUserIncentives(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user incentives: %v", err)
		return nil, runtime.NewError("failed to get user incentives", 13) // INTERNAL
	}

	// Check if incentive exists and belongs to user
	incentive, exists := userIncentives[code]
	if !exists {
		return nil, runtime.NewError("incentive not found", 5) // NOT_FOUND
	}

	// Check if incentive has been claimed (prevent deletion if so)
	if len(incentive.Claims) > 0 {
		return nil, runtime.NewError("cannot delete claimed incentive", 9) // FAILED_PRECONDITION
	}

	// Remove from user's incentives
	delete(userIncentives, code)

	// Save to storage
	err = i.saveUserIncentives(ctx, logger, nk, userID, userIncentives)
	if err != nil {
		logger.Error("Failed to save user incentives: %v", err)
		return nil, runtime.NewError("failed to save incentives", 13) // INTERNAL
	}

	// Return updated list
	return i.SenderList(ctx, logger, nk, userID)
}

// SenderClaim allows the incentive creator to claim rewards for recipients who have used their incentive.
func (i *NakamaIncentivesSystem) SenderClaim(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, code string, claimantIDs []string) (incentives []*Incentive, err error) {
	// Get user's incentives
	userIncentives, err := i.getUserIncentives(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user incentives: %v", err)
		return nil, runtime.NewError("failed to get user incentives", 13) // INTERNAL
	}

	// Check if incentive exists and belongs to user
	incentive, exists := userIncentives[code]
	if !exists {
		return nil, runtime.NewError("incentive not found", 5) // NOT_FOUND
	}

	// Get incentive config
	incentiveConfig, configExists := i.config.Incentives[incentive.Id]
	if !configExists {
		return nil, runtime.NewError("incentive configuration not found", 5) // NOT_FOUND
	}

	// Determine which recipients to claim for
	var recipientsToClaim []string
	if len(claimantIDs) == 0 {
		// Claim for all unclaimed recipients
		recipientsToClaim = incentive.UnclaimedRecipients
	} else {
		// Claim for specified recipients only
		recipientsToClaim = claimantIDs
	}

	// Process sender rewards for each recipient
	economySystem := i.pamlogix.GetEconomySystem()
	if economySystem == nil {
		return nil, runtime.NewError("economy system not available", 9) // FAILED_PRECONDITION
	}

	var totalReward *Reward
	claimedRecipients := make([]string, 0)

	for _, recipientID := range recipientsToClaim {
		// Check if this recipient is in the unclaimed list
		found := false
		for _, unclaimedID := range incentive.UnclaimedRecipients {
			if unclaimedID == recipientID {
				found = true
				break
			}
		}

		if !found {
			logger.Warn("Recipient %s not found in unclaimed list for incentive %s", recipientID, code)
			continue
		}

		// Roll sender reward
		if incentiveConfig.SenderReward != nil {
			reward, err := economySystem.RewardRoll(ctx, logger, nk, userID, incentiveConfig.SenderReward)
			if err != nil {
				logger.Error("Failed to roll sender reward: %v", err)
				continue
			}

			// Apply custom reward function if set
			if i.onSenderReward != nil {
				reward, err = i.onSenderReward(ctx, logger, nk, userID, incentive.Id, incentiveConfig, incentiveConfig.SenderReward, reward)
				if err != nil {
					logger.Error("Error in sender reward callback: %v", err)
					continue
				}
			}

			// Grant reward to sender
			if reward != nil {
				_, _, _, err = economySystem.RewardGrant(ctx, logger, nk, userID, reward, map[string]interface{}{
					"incentive_code": code,
					"recipient_id":   recipientID,
				}, false)
				if err != nil {
					logger.Error("Failed to grant sender reward: %v", err)
					continue
				}

				// Add to incentive rewards list
				incentive.Rewards = append(incentive.Rewards, reward)

				// Merge with total reward for response
				if totalReward == nil {
					totalReward = reward
				} else {
					totalReward = i.mergeRewards(totalReward, reward)
				}
			}
		}

		claimedRecipients = append(claimedRecipients, recipientID)
	}

	// Remove claimed recipients from unclaimed list
	if len(claimedRecipients) > 0 {
		newUnclaimedRecipients := make([]string, 0)
		for _, unclaimedID := range incentive.UnclaimedRecipients {
			shouldKeep := true
			for _, claimedID := range claimedRecipients {
				if unclaimedID == claimedID {
					shouldKeep = false
					break
				}
			}
			if shouldKeep {
				newUnclaimedRecipients = append(newUnclaimedRecipients, unclaimedID)
			}
		}
		incentive.UnclaimedRecipients = newUnclaimedRecipients
	}

	// Update incentive
	incentive.UpdateTimeSec = time.Now().Unix()

	// Save to storage
	err = i.saveUserIncentives(ctx, logger, nk, userID, userIncentives)
	if err != nil {
		logger.Error("Failed to save user incentives: %v", err)
		return nil, runtime.NewError("failed to save incentives", 13) // INTERNAL
	}

	// Return updated list
	return i.SenderList(ctx, logger, nk, userID)
}

// RecipientGet allows a potential recipient to view information about an incentive.
func (i *NakamaIncentivesSystem) RecipientGet(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, code string) (incentive *IncentiveInfo, err error) {
	// Find the incentive by code across all users
	incentiveData, senderID, err := i.findIncentiveByCode(ctx, logger, nk, code)
	if err != nil {
		return nil, err
	}
	if incentiveData == nil {
		return nil, runtime.NewError("incentive not found", 5) // NOT_FOUND
	}

	// Check if incentive has expired
	if incentiveData.ExpiryTimeSec > 0 && incentiveData.ExpiryTimeSec <= time.Now().Unix() {
		return nil, runtime.NewError("incentive has expired", 9) // FAILED_PRECONDITION
	}

	// Get incentive config
	incentiveConfig, configExists := i.config.Incentives[incentiveData.Id]
	if !configExists {
		return nil, runtime.NewError("incentive configuration not found", 5) // NOT_FOUND
	}

	// Check if user has already claimed this incentive
	claim, alreadyClaimed := incentiveData.Claims[userID]
	var claimTime int64
	var reward *Reward
	if alreadyClaimed {
		claimTime = claim.ClaimTimeSec
		reward = claim.Reward
	}

	// Check if user can claim (age restriction, max claims, etc.)
	canClaim := false
	if !alreadyClaimed {
		canClaim = i.canUserClaimIncentive(ctx, logger, nk, userID, incentiveData, incentiveConfig)
	}

	// Build response
	incentiveInfo := &IncentiveInfo{
		Id:               incentiveData.Id,
		Name:             incentiveData.Name,
		Description:      incentiveData.Description,
		Code:             incentiveData.Code,
		Type:             incentiveData.Type,
		Sender:           senderID,
		AvailableRewards: incentiveData.RecipientRewards,
		CanClaim:         canClaim,
		Reward:           reward,
		CreateTimeSec:    incentiveData.CreateTimeSec,
		UpdateTimeSec:    incentiveData.UpdateTimeSec,
		ExpiryTimeSec:    incentiveData.ExpiryTimeSec,
		ClaimTimeSec:     claimTime,
	}

	return incentiveInfo, nil
}

// RecipientClaim allows a user to claim an incentive and receive rewards.
func (i *NakamaIncentivesSystem) RecipientClaim(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, code string) (incentive *IncentiveInfo, err error) {
	// Find the incentive by code
	incentiveData, senderID, err := i.findIncentiveByCode(ctx, logger, nk, code)
	if err != nil {
		return nil, err
	}
	if incentiveData == nil {
		return nil, runtime.NewError("incentive not found", 5) // NOT_FOUND
	}

	// Check if incentive has expired
	if incentiveData.ExpiryTimeSec > 0 && incentiveData.ExpiryTimeSec <= time.Now().Unix() {
		return nil, runtime.NewError("incentive has expired", 9) // FAILED_PRECONDITION
	}

	// Get incentive config
	incentiveConfig, configExists := i.config.Incentives[incentiveData.Id]
	if !configExists {
		return nil, runtime.NewError("incentive configuration not found", 5) // NOT_FOUND
	}

	// Check if user has already claimed this incentive
	if _, alreadyClaimed := incentiveData.Claims[userID]; alreadyClaimed {
		return nil, runtime.NewError("incentive already claimed", 9) // FAILED_PRECONDITION
	}

	// Check if user can claim this incentive
	if !i.canUserClaimIncentive(ctx, logger, nk, userID, incentiveData, incentiveConfig) {
		return nil, runtime.NewError("user cannot claim this incentive", 9) // FAILED_PRECONDITION
	}

	// Check max claims limit
	if incentiveConfig.MaxClaims > 0 && len(incentiveData.Claims) >= incentiveConfig.MaxClaims {
		return nil, runtime.NewError("incentive claim limit reached", 9) // FAILED_PRECONDITION
	}

	// Roll recipient reward
	if i.pamlogix == nil {
		return nil, runtime.NewError("economy system not available", 9) // FAILED_PRECONDITION
	}
	economySystem := i.pamlogix.GetEconomySystem()
	if economySystem == nil {
		return nil, runtime.NewError("economy system not available", 9) // FAILED_PRECONDITION
	}

	var reward *Reward
	if incentiveConfig.RecipientReward != nil {
		reward, err = economySystem.RewardRoll(ctx, logger, nk, userID, incentiveConfig.RecipientReward)
		if err != nil {
			logger.Error("Failed to roll recipient reward: %v", err)
			return nil, runtime.NewError("failed to generate reward", 13) // INTERNAL
		}

		// Apply custom reward function if set
		if i.onRecipientReward != nil {
			reward, err = i.onRecipientReward(ctx, logger, nk, userID, incentiveData.Id, incentiveConfig, incentiveConfig.RecipientReward, reward)
			if err != nil {
				logger.Error("Error in recipient reward callback: %v", err)
				return nil, runtime.NewError("failed to process reward", 13) // INTERNAL
			}
		}

		// Grant reward to recipient
		if reward != nil {
			_, _, _, err = economySystem.RewardGrant(ctx, logger, nk, userID, reward, map[string]interface{}{
				"incentive_code": code,
				"sender_id":      senderID,
			}, false)
			if err != nil {
				logger.Error("Failed to grant recipient reward: %v", err)
				return nil, runtime.NewError("failed to grant reward", 13) // INTERNAL
			}
		}
	}

	// Record the claim
	now := time.Now().Unix()
	claim := &IncentiveClaim{
		Reward:       reward,
		ClaimTimeSec: now,
	}

	// Initialize Claims map if it's nil (can happen after JSON unmarshaling)
	if incentiveData.Claims == nil {
		incentiveData.Claims = make(map[string]*IncentiveClaim)
	}
	incentiveData.Claims[userID] = claim

	// Add user to unclaimed recipients list for sender
	incentiveData.UnclaimedRecipients = append(incentiveData.UnclaimedRecipients, userID)
	incentiveData.UpdateTimeSec = now

	// Save updated incentive data back to sender's storage
	err = i.saveIncentiveForSender(ctx, logger, nk, senderID, code, incentiveData)
	if err != nil {
		logger.Error("Failed to save updated incentive: %v", err)
		return nil, runtime.NewError("failed to save incentive", 13) // INTERNAL
	}

	// Return incentive info
	return &IncentiveInfo{
		Id:               incentiveData.Id,
		Name:             incentiveData.Name,
		Description:      incentiveData.Description,
		Code:             incentiveData.Code,
		Type:             incentiveData.Type,
		Sender:           senderID,
		AvailableRewards: incentiveData.RecipientRewards,
		CanClaim:         false, // Just claimed, so can't claim again
		Reward:           reward,
		CreateTimeSec:    incentiveData.CreateTimeSec,
		UpdateTimeSec:    incentiveData.UpdateTimeSec,
		ExpiryTimeSec:    incentiveData.ExpiryTimeSec,
		ClaimTimeSec:     now,
	}, nil
}

// SetOnSenderReward sets a custom reward function for sender rewards.
func (i *NakamaIncentivesSystem) SetOnSenderReward(fn OnReward[*IncentivesConfigIncentive]) {
	i.onSenderReward = fn
}

// SetOnRecipientReward sets a custom reward function for recipient rewards.
func (i *NakamaIncentivesSystem) SetOnRecipientReward(fn OnReward[*IncentivesConfigIncentive]) {
	i.onRecipientReward = fn
}

// Helper functions

// generateIncentiveCode generates a unique incentive code.
func (i *NakamaIncentivesSystem) generateIncentiveCode() string {
	// Generate a UUID and use part of it for the code
	id := uuid.New()
	// Use first 8 characters of UUID (without hyphens) and make it uppercase
	codeStr := strings.ToUpper(strings.ReplaceAll(id.String(), "-", ""))[:8]
	return fmt.Sprintf("%s%s", incentiveCodePrefix, codeStr)
}

// getUserIncentives retrieves a user's incentives from storage.
func (i *NakamaIncentivesSystem) getUserIncentives(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (map[string]*Incentive, error) {
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{{
		Collection: incentivesStorageCollection,
		Key:        userIncentivesStorageKey,
		UserID:     userID,
	}})
	if err != nil {
		return nil, err
	}

	incentives := make(map[string]*Incentive)
	if len(objects) > 0 && objects[0].Value != "" {
		err = json.Unmarshal([]byte(objects[0].Value), &incentives)
		if err != nil {
			return nil, err
		}
	}

	return incentives, nil
}

// saveUserIncentives saves a user's incentives to storage.
func (i *NakamaIncentivesSystem) saveUserIncentives(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, incentives map[string]*Incentive) error {
	data, err := json.Marshal(incentives)
	if err != nil {
		return err
	}

	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{{
		Collection:      incentivesStorageCollection,
		Key:             userIncentivesStorageKey,
		UserID:          userID,
		Value:           string(data),
		PermissionRead:  runtime.STORAGE_PERMISSION_OWNER_READ,
		PermissionWrite: runtime.STORAGE_PERMISSION_OWNER_WRITE,
	}})

	return err
}

// findIncentiveByCode searches for an incentive by its code across all users.
func (i *NakamaIncentivesSystem) findIncentiveByCode(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, code string) (*Incentive, string, error) {
	// This is a simplified implementation. In a production system, you might want to:
	// 1. Use a separate index collection for codes
	// 2. Use database queries for better performance
	// 3. Implement caching

	// For now, we'll search through storage objects
	// This is not efficient for large numbers of users, but works for demonstration
	objects, _, err := nk.StorageList(ctx, incentivesStorageCollection, "", userIncentivesStorageKey, 1000, "")
	if err != nil {
		return nil, "", err
	}

	for _, obj := range objects {
		var userIncentives map[string]*Incentive
		err = json.Unmarshal([]byte(obj.Value), &userIncentives)
		if err != nil {
			continue
		}

		if incentive, exists := userIncentives[code]; exists {
			return incentive, obj.UserId, nil
		}
	}

	return nil, "", nil
}

// saveIncentiveForSender saves a specific incentive back to the sender's storage.
func (i *NakamaIncentivesSystem) saveIncentiveForSender(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, senderID, code string, incentive *Incentive) error {
	// Get sender's current incentives
	userIncentives, err := i.getUserIncentives(ctx, logger, nk, senderID)
	if err != nil {
		return err
	}

	// Update the specific incentive
	userIncentives[code] = incentive

	// Save back to storage
	return i.saveUserIncentives(ctx, logger, nk, senderID, userIncentives)
}

// canUserClaimIncentive checks if a user is eligible to claim an incentive.
func (i *NakamaIncentivesSystem) canUserClaimIncentive(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, incentive *Incentive, config *IncentivesConfigIncentive) bool {
	// Check max recipient age
	if config.MaxRecipientAgeSec > 0 {
		account, err := nk.AccountGetId(ctx, userID)
		if err != nil {
			logger.Error("Failed to get account for age check: %v", err)
			return false
		}

		accountAge := time.Now().Unix() - account.User.CreateTime.AsTime().Unix()
		if accountAge > config.MaxRecipientAgeSec {
			return false
		}
	}

	// Check global claims limit
	if config.MaxGlobalClaims > 0 {
		// Count how many times this user has claimed this type of incentive
		claimCount := i.getUserGlobalClaimCount(ctx, logger, nk, userID, incentive.Id)
		if claimCount >= config.MaxGlobalClaims {
			return false
		}
	}

	return true
}

// getUserGlobalClaimCount counts how many times a user has claimed a specific incentive type.
func (i *NakamaIncentivesSystem) getUserGlobalClaimCount(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, incentiveID string) int {
	// This would require tracking claims across all incentives
	// For simplicity, we'll return 0 for now
	// In a production system, you'd want to maintain a separate claims index
	return 0
}

// convertRewardConfigToAvailableRewards converts EconomyConfigReward to AvailableRewards format.
func (i *NakamaIncentivesSystem) convertRewardConfigToAvailableRewards(rewardConfig *EconomyConfigReward) *AvailableRewards {
	if rewardConfig == nil {
		return nil
	}

	// Use the economy system to do the conversion if available
	if i.pamlogix != nil {
		if economySystem := i.pamlogix.GetEconomySystem(); economySystem != nil {
			// This is a reverse operation - we need to build AvailableRewards from EconomyConfigReward
			// For now, we'll create a basic structure
			return &AvailableRewards{
				MaxRolls:       rewardConfig.MaxRolls,
				TotalWeight:    rewardConfig.TotalWeight,
				MaxRepeatRolls: rewardConfig.MaxRepeatRolls,
			}
		}
	}

	return &AvailableRewards{
		MaxRolls:       rewardConfig.MaxRolls,
		TotalWeight:    rewardConfig.TotalWeight,
		MaxRepeatRolls: rewardConfig.MaxRepeatRolls,
	}
}

// mergeRewards combines two rewards into one.
func (i *NakamaIncentivesSystem) mergeRewards(reward1, reward2 *Reward) *Reward {
	if reward1 == nil {
		return reward2
	}
	if reward2 == nil {
		return reward1
	}

	merged := &Reward{
		Items:           make(map[string]int64),
		Currencies:      make(map[string]int64),
		Energies:        make(map[string]int32),
		EnergyModifiers: make([]*RewardEnergyModifier, 0),
		RewardModifiers: make([]*RewardModifier, 0),
		ItemInstances:   make(map[string]*RewardInventoryItem),
		GrantTimeSec:    reward1.GrantTimeSec,
	}

	// Merge items
	for k, v := range reward1.Items {
		merged.Items[k] = v
	}
	for k, v := range reward2.Items {
		merged.Items[k] += v
	}

	// Merge currencies
	for k, v := range reward1.Currencies {
		merged.Currencies[k] = v
	}
	for k, v := range reward2.Currencies {
		merged.Currencies[k] += v
	}

	// Merge energies
	for k, v := range reward1.Energies {
		merged.Energies[k] = v
	}
	for k, v := range reward2.Energies {
		merged.Energies[k] += v
	}

	// Merge modifiers
	merged.EnergyModifiers = append(merged.EnergyModifiers, reward1.EnergyModifiers...)
	merged.EnergyModifiers = append(merged.EnergyModifiers, reward2.EnergyModifiers...)
	merged.RewardModifiers = append(merged.RewardModifiers, reward1.RewardModifiers...)
	merged.RewardModifiers = append(merged.RewardModifiers, reward2.RewardModifiers...)

	// Merge item instances (reward2 takes precedence for conflicts)
	for k, v := range reward1.ItemInstances {
		merged.ItemInstances[k] = v
	}
	for k, v := range reward2.ItemInstances {
		merged.ItemInstances[k] = v
	}

	return merged
}

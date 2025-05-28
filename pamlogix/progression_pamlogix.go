package pamlogix

import (
	"context"
	"encoding/json"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/robfig/cron/v3"
)

const (
	progressionStorageCollection = "progression"
	userProgressionStorageKey    = "user_progressions"
)

// NakamaProgressionSystem implements the ProgressionSystem interface using Nakama as the backend.
type NakamaProgressionSystem struct {
	config     *ProgressionConfig
	pamlogix   Pamlogix
	cronParser cron.Parser
}

// NewNakamaProgressionSystem creates a new instance of the progression system with the given configuration.
func NewNakamaProgressionSystem(config *ProgressionConfig) *NakamaProgressionSystem {
	return &NakamaProgressionSystem{
		config:     config,
		cronParser: cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
	}
}

// SetPamlogix sets the Pamlogix instance for this progression system
func (p *NakamaProgressionSystem) SetPamlogix(pl Pamlogix) {
	p.pamlogix = pl
}

// GetType returns the system type for the progression system.
func (p *NakamaProgressionSystem) GetType() SystemType {
	return SystemTypeProgression
}

// GetConfig returns the configuration for the progression system.
func (p *NakamaProgressionSystem) GetConfig() any {
	return p.config
}

// Get returns all or an optionally-filtered set of progressions for the given user.
func (p *NakamaProgressionSystem) Get(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, lastKnownProgressions map[string]*Progression) (progressions map[string]*Progression, deltas map[string]*ProgressionDelta, err error) {
	if p.config == nil {
		return nil, nil, runtime.NewError("progression config not loaded", 13)
	}

	// Get user's progression data from storage
	userProgressions, err := p.getUserProgressions(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user progressions: %v", err)
		return nil, nil, err
	}

	now := time.Now().Unix()
	progressions = make(map[string]*Progression)
	deltas = make(map[string]*ProgressionDelta)

	// Process each progression from config
	for progressionID, progressionConfig := range p.config.Progressions {
		// Create progression from config
		progression := &Progression{
			Id:                   progressionID,
			Name:                 progressionConfig.Name,
			Description:          progressionConfig.Description,
			Category:             progressionConfig.Category,
			Counts:               make(map[string]int64),
			AdditionalProperties: progressionConfig.AdditionalProperties,
			Unlocked:             false,
			Preconditions:        progressionConfig.Preconditions,
		}

		// Merge with user data if it exists
		if userProgression, exists := userProgressions[progressionID]; exists {
			progression.Counts = userProgression.Counts

			// Apply scheduled resets if needed
			if progressionConfig.ResetSchedule != "" {
				progression = p.applyScheduledResets(logger, progressionConfig, progression, userProgression.UpdateTimeSec, now)
			}
		}

		// Check preconditions and determine unlock status
		progression.UnmetPreconditions = p.checkPreconditions(ctx, logger, nk, userID, progressionConfig.Preconditions, progressions)

		// Progression is unlocked if all preconditions are met
		progression.Unlocked = progression.UnmetPreconditions == nil

		// Calculate deltas if lastKnownProgressions is provided
		if lastKnownProgressions != nil {
			if lastKnown, exists := lastKnownProgressions[progressionID]; exists {
				delta := p.calculateDelta(lastKnown, progression)
				if delta != nil {
					deltas[progressionID] = delta
				}
			} else {
				// New progression
				deltas[progressionID] = &ProgressionDelta{
					Id:    progressionID,
					State: ProgressionDeltaState_PROGRESSION_DELTA_STATE_UNLOCKED,
				}
			}
		}

		progressions[progressionID] = progression
	}

	return progressions, deltas, nil
}

// Purchase permanently unlocks a specified progression, if that progression supports this operation.
func (p *NakamaProgressionSystem) Purchase(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, progressionID string) (progressions map[string]*Progression, err error) {
	if p.config == nil {
		return nil, runtime.NewError("progression config not loaded", 13)
	}

	progressionConfig, exists := p.config.Progressions[progressionID]
	if !exists {
		return nil, ErrProgressionNotFound
	}

	// Check if progression has a cost (required for purchase)
	if progressionConfig.Preconditions == nil || progressionConfig.Preconditions.Direct == nil || progressionConfig.Preconditions.Direct.Cost == nil {
		return nil, ErrProgressionNoCost
	}

	// Get user's progression data
	userProgressions, err := p.getUserProgressions(ctx, logger, nk, userID)
	if err != nil {
		return nil, err
	}

	// Check if already unlocked (progression is unlocked if cost has been paid)
	if userProgression, exists := userProgressions[progressionID]; exists && userProgression.Cost != nil {
		return nil, ErrProgressionAlreadyUnlocked
	}

	// Check if purchase is available (preconditions met except cost)
	if !p.canPurchase(ctx, logger, nk, userID, progressionConfig) {
		return nil, ErrProgressionNotAvailablePurchase
	}

	// Process the purchase cost through economy system
	economySystem := p.pamlogix.GetEconomySystem()
	if economySystem == nil {
		return nil, runtime.NewError("economy system not available", 13)
	}

	// Create a reward config for the cost (negative amounts)
	costReward := &EconomyConfigReward{
		Guaranteed: &EconomyConfigRewardContents{
			Currencies: make(map[string]*EconomyConfigRewardCurrency),
			Items:      make(map[string]*EconomyConfigRewardItem),
		},
	}

	if progressionConfig.Preconditions.Direct.Cost.Currencies != nil {
		for currencyID, amount := range progressionConfig.Preconditions.Direct.Cost.Currencies {
			costReward.Guaranteed.Currencies[currencyID] = &EconomyConfigRewardCurrency{
				EconomyConfigRewardRangeInt64: EconomyConfigRewardRangeInt64{
					Min: -amount, // Negative to deduct
					Max: -amount,
				},
			}
		}
	}
	if progressionConfig.Preconditions.Direct.Cost.Items != nil {
		for itemID, amount := range progressionConfig.Preconditions.Direct.Cost.Items {
			costReward.Guaranteed.Items[itemID] = &EconomyConfigRewardItem{
				EconomyConfigRewardRangeInt64: EconomyConfigRewardRangeInt64{
					Min: -amount, // Negative to deduct
					Max: -amount,
				},
			}
		}
	}

	// Roll and grant the cost (deduction)
	rolledCost, err := economySystem.RewardRoll(ctx, logger, nk, userID, costReward)
	if err != nil {
		return nil, err
	}

	_, _, _, err = economySystem.RewardGrant(ctx, logger, nk, userID, rolledCost, map[string]interface{}{
		"progression_id": progressionID,
		"type":           "progression_purchase",
	}, false)
	if err != nil {
		return nil, err
	}

	// Mark progression as unlocked
	now := time.Now().Unix()
	userProgression := &SyncProgressionUpdate{
		Counts:        make(map[string]int64),
		CreateTimeSec: now,
		UpdateTimeSec: now,
		Cost:          progressionConfig.Preconditions.Direct.Cost,
	}

	userProgressions[progressionID] = userProgression

	// Save to storage
	err = p.saveUserProgressions(ctx, logger, nk, userID, userProgressions)
	if err != nil {
		return nil, err
	}

	// Return updated progressions
	progressions, _, err = p.Get(ctx, logger, nk, userID, nil)
	return progressions, err
}

// Update a specified progression, if that progression supports this operation.
func (p *NakamaProgressionSystem) Update(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, progressionID string, counts map[string]int64) (progressions map[string]*Progression, err error) {
	if p.config == nil {
		return nil, runtime.NewError("progression config not loaded", 13)
	}

	progressionConfig, exists := p.config.Progressions[progressionID]
	if !exists {
		return nil, ErrProgressionNotFound
	}

	// Check if progression has counts (required for update)
	if progressionConfig.Preconditions == nil || progressionConfig.Preconditions.Direct == nil || progressionConfig.Preconditions.Direct.Counts == nil {
		return nil, ErrProgressionNoCount
	}

	// Get user's progression data
	userProgressions, err := p.getUserProgressions(ctx, logger, nk, userID)
	if err != nil {
		return nil, err
	}

	// Check if update is available (progression unlocked or can be unlocked)
	if !p.canUpdate(ctx, logger, nk, userID, progressionConfig, userProgressions[progressionID]) {
		return nil, ErrProgressionNotAvailableUpdate
	}

	// Get or create user progression
	userProgression, exists := userProgressions[progressionID]
	if !exists {
		now := time.Now().Unix()
		userProgression = &SyncProgressionUpdate{
			Counts:        make(map[string]int64),
			CreateTimeSec: now,
			UpdateTimeSec: now,
		}
	}

	// Update counts
	now := time.Now().Unix()
	for countID, amount := range counts {
		if userProgression.Counts == nil {
			userProgression.Counts = make(map[string]int64)
		}
		userProgression.Counts[countID] += amount
		if userProgression.Counts[countID] < 0 {
			userProgression.Counts[countID] = 0
		}
	}
	userProgression.UpdateTimeSec = now

	userProgressions[progressionID] = userProgression

	// Save to storage
	err = p.saveUserProgressions(ctx, logger, nk, userID, userProgressions)
	if err != nil {
		return nil, err
	}

	// Return updated progressions
	progressions, _, err = p.Get(ctx, logger, nk, userID, nil)
	return progressions, err
}

// Reset one or more progressions to clear their progress. Only applies to progression counts and unlock costs.
func (p *NakamaProgressionSystem) Reset(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, progressionIDs []string) (progressions map[string]*Progression, err error) {
	if p.config == nil {
		return nil, runtime.NewError("progression config not loaded", 13)
	}

	// Get user's progression data
	userProgressions, err := p.getUserProgressions(ctx, logger, nk, userID)
	if err != nil {
		return nil, err
	}

	needsSave := false
	for _, progressionID := range progressionIDs {
		if _, exists := p.config.Progressions[progressionID]; !exists {
			logger.Warn("Progression config not found for reset: %s", progressionID)
			continue
		}

		if userProgression, exists := userProgressions[progressionID]; exists {
			// Reset counts and unlock status
			userProgression.Counts = make(map[string]int64)
			userProgression.Cost = nil
			userProgression.UpdateTimeSec = time.Now().Unix()
			needsSave = true
		}
	}

	if needsSave {
		err = p.saveUserProgressions(ctx, logger, nk, userID, userProgressions)
		if err != nil {
			return nil, err
		}
	}

	// Return updated progressions
	progressions, _, err = p.Get(ctx, logger, nk, userID, nil)
	return progressions, err
}

// Complete marks a progression as completed and grants any associated rewards.
func (p *NakamaProgressionSystem) Complete(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, progressionID string) (progressions map[string]*Progression, reward *Reward, err error) {
	if p.config == nil {
		return nil, nil, runtime.NewError("progression config not loaded", 13)
	}

	progressionConfig, exists := p.config.Progressions[progressionID]
	if !exists {
		return nil, nil, ErrProgressionNotFound
	}

	// Check if progression is unlocked and can be completed
	currentProgressions, _, err := p.Get(ctx, logger, nk, userID, nil)
	if err != nil {
		return nil, nil, err
	}

	progression, exists := currentProgressions[progressionID]
	if !exists || !progression.Unlocked {
		return nil, nil, runtime.NewError("progression not unlocked", 3)
	}

	// Grant rewards if configured
	if progressionConfig.Rewards != nil {
		economySystem := p.pamlogix.GetEconomySystem()
		if economySystem != nil {
			// Roll the reward
			reward, err = economySystem.RewardRoll(ctx, logger, nk, userID, progressionConfig.Rewards)
			if err != nil {
				logger.Error("Failed to roll progression reward: %v", err)
				return nil, nil, err
			}

			// Grant the reward
			_, _, _, err = economySystem.RewardGrant(ctx, logger, nk, userID, reward, map[string]interface{}{
				"progression_id": progressionID,
				"type":           "progression_completion",
			}, false)
			if err != nil {
				logger.Error("Failed to grant progression reward: %v", err)
				return nil, nil, err
			}
		}
	}

	// Mark progression as completed in user data
	userProgressions, err := p.getUserProgressions(ctx, logger, nk, userID)
	if err != nil {
		return nil, nil, err
	}

	now := time.Now().Unix()
	if userProgression, exists := userProgressions[progressionID]; exists {
		userProgression.UpdateTimeSec = now
		// Add a completion timestamp to additional properties if needed
		// This could be extended to track completion history
	} else {
		userProgressions[progressionID] = &SyncProgressionUpdate{
			Counts:        make(map[string]int64),
			CreateTimeSec: now,
			UpdateTimeSec: now,
		}
	}

	// Save to storage
	err = p.saveUserProgressions(ctx, logger, nk, userID, userProgressions)
	if err != nil {
		return nil, nil, err
	}

	// Return updated progressions
	progressions, _, err = p.Get(ctx, logger, nk, userID, nil)
	return progressions, reward, err
}

// Helper methods

// getUserProgressions retrieves user progression data from storage
func (p *NakamaProgressionSystem) getUserProgressions(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (map[string]*SyncProgressionUpdate, error) {
	collection := progressionStorageCollection

	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{{
		Collection: collection,
		Key:        userProgressionStorageKey,
		UserID:     userID,
	}})
	if err != nil {
		logger.Error("Failed to read user progressions from storage: %v", err)
		return nil, err
	}

	userProgressions := make(map[string]*SyncProgressionUpdate)
	if len(objects) > 0 {
		if err := json.Unmarshal([]byte(objects[0].Value), &userProgressions); err != nil {
			logger.Error("Failed to unmarshal user progressions: %v", err)
			return nil, err
		}
	}

	return userProgressions, nil
}

// saveUserProgressions saves user progression data to storage
func (p *NakamaProgressionSystem) saveUserProgressions(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, userProgressions map[string]*SyncProgressionUpdate) error {
	collection := progressionStorageCollection

	data, err := json.Marshal(userProgressions)
	if err != nil {
		logger.Error("Failed to marshal user progressions: %v", err)
		return err
	}

	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{{
		Collection:      collection,
		Key:             userProgressionStorageKey,
		UserID:          userID,
		Value:           string(data),
		PermissionRead:  runtime.STORAGE_PERMISSION_OWNER_READ,
		PermissionWrite: runtime.STORAGE_PERMISSION_OWNER_WRITE,
	}})
	if err != nil {
		logger.Error("Failed to write user progressions to storage: %v", err)
		return err
	}

	return nil
}

// checkPreconditions evaluates progression preconditions and returns unmet ones
func (p *NakamaProgressionSystem) checkPreconditions(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, preconditions *ProgressionPreconditionsBlock, currentProgressions map[string]*Progression) *ProgressionPreconditionsBlock {
	if preconditions == nil {
		return nil
	}

	// Evaluate direct preconditions
	directMet := true
	var unmetDirect *ProgressionPreconditions
	if preconditions.Direct != nil {
		unmetDirect = p.checkDirectPreconditions(ctx, logger, nk, userID, preconditions.Direct, currentProgressions)
		directMet = unmetDirect == nil
	}

	// Evaluate nested preconditions
	nestedMet := true
	var unmetNested *ProgressionPreconditionsBlock
	if preconditions.Nested != nil {
		unmetNested = p.checkPreconditions(ctx, logger, nk, userID, preconditions.Nested, currentProgressions)
		nestedMet = unmetNested == nil
	}

	// Apply logical operator to determine overall result
	operator := preconditions.Operator
	if operator == ProgressionPreconditionsOperator_PROGRESSION_PRECONDITIONS_OPERATOR_UNSPECIFIED {
		operator = ProgressionPreconditionsOperator_PROGRESSION_PRECONDITIONS_OPERATOR_AND // Default to AND
	}

	var overallMet bool
	switch operator {
	case ProgressionPreconditionsOperator_PROGRESSION_PRECONDITIONS_OPERATOR_AND:
		overallMet = directMet && nestedMet
	case ProgressionPreconditionsOperator_PROGRESSION_PRECONDITIONS_OPERATOR_OR:
		overallMet = directMet || nestedMet
	case ProgressionPreconditionsOperator_PROGRESSION_PRECONDITIONS_OPERATOR_XOR:
		overallMet = directMet != nestedMet // XOR: exactly one must be true
	case ProgressionPreconditionsOperator_PROGRESSION_PRECONDITIONS_OPERATOR_NOT:
		overallMet = directMet && !nestedMet // Direct must be true, nested must be false
	default:
		overallMet = directMet && nestedMet // Default to AND
	}

	// If overall conditions are met, return nil (no unmet conditions)
	if overallMet {
		return nil
	}

	// Build unmet preconditions response based on what failed
	unmetPreconditions := &ProgressionPreconditionsBlock{
		Operator: operator,
	}

	// Include unmet conditions based on the operator logic
	switch operator {
	case ProgressionPreconditionsOperator_PROGRESSION_PRECONDITIONS_OPERATOR_AND:
		// For AND, include all unmet conditions
		if !directMet {
			unmetPreconditions.Direct = unmetDirect
		}
		if !nestedMet {
			unmetPreconditions.Nested = unmetNested
		}
	case ProgressionPreconditionsOperator_PROGRESSION_PRECONDITIONS_OPERATOR_OR:
		// For OR, include all conditions since none were met
		if !directMet {
			unmetPreconditions.Direct = unmetDirect
		}
		if !nestedMet {
			unmetPreconditions.Nested = unmetNested
		}
	case ProgressionPreconditionsOperator_PROGRESSION_PRECONDITIONS_OPERATOR_XOR:
		// For XOR, include appropriate unmet conditions
		if directMet && nestedMet {
			// Both are met, but XOR requires exactly one - this is an edge case
			// Include both to indicate the conflict
			unmetPreconditions.Direct = unmetDirect
			unmetPreconditions.Nested = unmetNested
		} else if !directMet && !nestedMet {
			// Neither is met, include both
			unmetPreconditions.Direct = unmetDirect
			unmetPreconditions.Nested = unmetNested
		}
		// If exactly one is met, XOR is satisfied, so we shouldn't reach here
	case ProgressionPreconditionsOperator_PROGRESSION_PRECONDITIONS_OPERATOR_NOT:
		// For NOT, direct must be true and nested must be false
		if !directMet {
			unmetPreconditions.Direct = unmetDirect
		}
		if nestedMet {
			// Nested should not be met, but it is - indicate this violation
			unmetPreconditions.Nested = unmetNested
		}
	}

	return unmetPreconditions
}

// checkDirectPreconditions evaluates direct preconditions
func (p *NakamaProgressionSystem) checkDirectPreconditions(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, preconditions *ProgressionPreconditions, currentProgressions map[string]*Progression) *ProgressionPreconditions {
	unmet := &ProgressionPreconditions{}
	hasUnmet := false

	// Check progression dependencies
	if len(preconditions.Progressions) > 0 {
		for _, requiredProgressionID := range preconditions.Progressions {
			if progression, exists := currentProgressions[requiredProgressionID]; !exists || !progression.Unlocked {
				if unmet.Progressions == nil {
					unmet.Progressions = make([]string, 0)
				}
				unmet.Progressions = append(unmet.Progressions, requiredProgressionID)
				hasUnmet = true
			}
		}
	}

	// Check count requirements
	if len(preconditions.Counts) > 0 {
		for countID, requiredAmount := range preconditions.Counts {
			// Find the progression that contains this count requirement
			// This is typically checked against the current progression being evaluated
			// We need to get the current progression's counts from storage
			userProgressions, err := p.getUserProgressions(ctx, logger, nk, userID)
			if err != nil {
				logger.Error("Failed to get user progressions for count check: %v", err)
				// Treat as unmet if we can't check
				if unmet.Counts == nil {
					unmet.Counts = make(map[string]int64)
				}
				unmet.Counts[countID] = requiredAmount
				hasUnmet = true
				continue
			}

			// Check if any progression has sufficient count for this requirement
			countMet := false
			for _, userProgression := range userProgressions {
				if userProgression.Counts != nil {
					if currentCount, exists := userProgression.Counts[countID]; exists && currentCount >= requiredAmount {
						countMet = true
						break
					}
				}
			}

			if !countMet {
				if unmet.Counts == nil {
					unmet.Counts = make(map[string]int64)
				}
				unmet.Counts[countID] = requiredAmount
				hasUnmet = true
			}
		}
	}

	// Check achievement dependencies
	if len(preconditions.Achievements) > 0 {
		achievementsSystem := p.pamlogix.GetAchievementsSystem()
		if achievementsSystem != nil {
			achievements, _, err := achievementsSystem.GetAchievements(ctx, logger, nk, userID)
			if err == nil {
				for _, requiredAchievementID := range preconditions.Achievements {
					if achievement, exists := achievements[requiredAchievementID]; !exists || achievement.ClaimTimeSec == 0 {
						if unmet.Achievements == nil {
							unmet.Achievements = make([]string, 0)
						}
						unmet.Achievements = append(unmet.Achievements, requiredAchievementID)
						hasUnmet = true
					}
				}
			}
		}
	}

	// Check currency requirements
	if len(preconditions.CurrencyMin) > 0 {
		economySystem := p.pamlogix.GetEconomySystem()
		if economySystem != nil {
			// Get user's account to access wallet
			account, err := nk.AccountGetId(ctx, userID)
			if err != nil {
				logger.Error("Failed to get user account for currency check: %v", err)
				// Treat all currency requirements as unmet if we can't check wallet
				for currencyID, minAmount := range preconditions.CurrencyMin {
					if unmet.CurrencyMin == nil {
						unmet.CurrencyMin = make(map[string]int64)
					}
					unmet.CurrencyMin[currencyID] = minAmount
					hasUnmet = true
				}
			} else {
				// Unmarshal wallet from account
				wallet, err := economySystem.UnmarshalWallet(account)
				if err != nil {
					logger.Error("Failed to unmarshal wallet for currency check: %v", err)
					// Treat all currency requirements as unmet if we can't unmarshal wallet
					for currencyID, minAmount := range preconditions.CurrencyMin {
						if unmet.CurrencyMin == nil {
							unmet.CurrencyMin = make(map[string]int64)
						}
						unmet.CurrencyMin[currencyID] = minAmount
						hasUnmet = true
					}
				} else {
					for currencyID, minAmount := range preconditions.CurrencyMin {
						currentAmount := int64(0)
						if wallet != nil {
							if amount, exists := wallet[currencyID]; exists {
								currentAmount = amount
							}
						}

						if currentAmount < minAmount {
							if unmet.CurrencyMin == nil {
								unmet.CurrencyMin = make(map[string]int64)
							}
							unmet.CurrencyMin[currencyID] = minAmount
							hasUnmet = true
						}
					}
				}
			}
		} else {
			// Economy system not available, treat all currency requirements as unmet
			for currencyID, minAmount := range preconditions.CurrencyMin {
				if unmet.CurrencyMin == nil {
					unmet.CurrencyMin = make(map[string]int64)
				}
				unmet.CurrencyMin[currencyID] = minAmount
				hasUnmet = true
			}
		}
	}

	// Check item requirements
	if len(preconditions.ItemsMin) > 0 {
		inventorySystem := p.pamlogix.GetInventorySystem()
		if inventorySystem != nil {
			inventory, err := inventorySystem.ListInventoryItems(ctx, logger, nk, userID, "")
			if err == nil && inventory != nil {
				for itemID, minAmount := range preconditions.ItemsMin {
					if item, exists := inventory.Items[itemID]; !exists || item.Count < minAmount {
						if unmet.ItemsMin == nil {
							unmet.ItemsMin = make(map[string]int64)
						}
						unmet.ItemsMin[itemID] = minAmount
						hasUnmet = true
					}
				}
			}
		}
	}

	// Check stats requirements
	if len(preconditions.StatsMin) > 0 {
		statsSystem := p.pamlogix.GetStatsSystem()
		if statsSystem != nil {
			stats, err := statsSystem.List(ctx, logger, nk, userID, []string{userID})
			if err == nil && stats != nil {
				userStats := stats[userID]
				if userStats != nil {
					for statID, minValue := range preconditions.StatsMin {
						if stat, exists := userStats.Private[statID]; !exists || stat.Value < minValue {
							if unmet.StatsMin == nil {
								unmet.StatsMin = make(map[string]int64)
							}
							unmet.StatsMin[statID] = minValue
							hasUnmet = true
						}
					}
				}
			}
		}
	}

	// Check energy requirements
	if len(preconditions.EnergyMin) > 0 {
		energySystem := p.pamlogix.GetEnergySystem()
		if energySystem != nil {
			energies, err := energySystem.Get(ctx, logger, nk, userID)
			if err == nil {
				for energyID, minAmount := range preconditions.EnergyMin {
					if energy, exists := energies[energyID]; !exists || energy.Current < int32(minAmount) {
						if unmet.EnergyMin == nil {
							unmet.EnergyMin = make(map[string]int64)
						}
						unmet.EnergyMin[energyID] = minAmount
						hasUnmet = true
					}
				}
			}
		}
	}

	// Check maximum energy requirements
	if len(preconditions.EnergyMax) > 0 {
		energySystem := p.pamlogix.GetEnergySystem()
		if energySystem != nil {
			energies, err := energySystem.Get(ctx, logger, nk, userID)
			if err == nil {
				for energyID, maxAmount := range preconditions.EnergyMax {
					if energy, exists := energies[energyID]; exists && energy.Current > int32(maxAmount) {
						if unmet.EnergyMax == nil {
							unmet.EnergyMax = make(map[string]int64)
						}
						unmet.EnergyMax[energyID] = maxAmount
						hasUnmet = true
					}
				}
			}
		}
	}

	// Check maximum currency requirements
	if len(preconditions.CurrencyMax) > 0 {
		economySystem := p.pamlogix.GetEconomySystem()
		if economySystem != nil {
			// Get user's account to access wallet
			account, err := nk.AccountGetId(ctx, userID)
			if err == nil {
				// Unmarshal wallet from account
				wallet, err := economySystem.UnmarshalWallet(account)
				if err == nil {
					for currencyID, maxAmount := range preconditions.CurrencyMax {
						currentAmount := int64(0)
						if wallet != nil {
							if amount, exists := wallet[currencyID]; exists {
								currentAmount = amount
							}
						}

						if currentAmount > maxAmount {
							if unmet.CurrencyMax == nil {
								unmet.CurrencyMax = make(map[string]int64)
							}
							unmet.CurrencyMax[currencyID] = maxAmount
							hasUnmet = true
						}
					}
				}
			}
		}
	}

	// Check maximum item requirements
	if len(preconditions.ItemsMax) > 0 {
		inventorySystem := p.pamlogix.GetInventorySystem()
		if inventorySystem != nil {
			inventory, err := inventorySystem.ListInventoryItems(ctx, logger, nk, userID, "")
			if err == nil && inventory != nil {
				for itemID, maxAmount := range preconditions.ItemsMax {
					if item, exists := inventory.Items[itemID]; exists && item.Count > maxAmount {
						if unmet.ItemsMax == nil {
							unmet.ItemsMax = make(map[string]int64)
						}
						unmet.ItemsMax[itemID] = maxAmount
						hasUnmet = true
					}
				}
			}
		}
	}

	// Check maximum stats requirements
	if len(preconditions.StatsMax) > 0 {
		statsSystem := p.pamlogix.GetStatsSystem()
		if statsSystem != nil {
			stats, err := statsSystem.List(ctx, logger, nk, userID, []string{userID})
			if err == nil && stats != nil {
				userStats := stats[userID]
				if userStats != nil {
					for statID, maxValue := range preconditions.StatsMax {
						if stat, exists := userStats.Private[statID]; exists && stat.Value > maxValue {
							if unmet.StatsMax == nil {
								unmet.StatsMax = make(map[string]int64)
							}
							unmet.StatsMax[statID] = maxValue
							hasUnmet = true
						}
					}
				}
			}
		}
	}

	if !hasUnmet {
		return nil
	}

	return unmet
}

// canPurchase checks if a progression can be purchased
func (p *NakamaProgressionSystem) canPurchase(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, progressionConfig *ProgressionConfigProgression) bool {
	// Check all preconditions except cost
	if progressionConfig.Preconditions == nil || progressionConfig.Preconditions.Direct == nil {
		return true
	}

	// Create a temporary preconditions block without cost
	tempPreconditions := &ProgressionPreconditions{
		Counts:       progressionConfig.Preconditions.Direct.Counts,
		Progressions: progressionConfig.Preconditions.Direct.Progressions,
		Achievements: progressionConfig.Preconditions.Direct.Achievements,
		ItemsMin:     progressionConfig.Preconditions.Direct.ItemsMin,
		ItemsMax:     progressionConfig.Preconditions.Direct.ItemsMax,
		StatsMin:     progressionConfig.Preconditions.Direct.StatsMin,
		StatsMax:     progressionConfig.Preconditions.Direct.StatsMax,
		EnergyMin:    progressionConfig.Preconditions.Direct.EnergyMin,
		EnergyMax:    progressionConfig.Preconditions.Direct.EnergyMax,
		CurrencyMin:  progressionConfig.Preconditions.Direct.CurrencyMin,
		CurrencyMax:  progressionConfig.Preconditions.Direct.CurrencyMax,
		// Cost is intentionally omitted
	}

	unmet := p.checkDirectPreconditions(ctx, logger, nk, userID, tempPreconditions, nil)
	return unmet == nil
}

// canUpdate checks if a progression can be updated
func (p *NakamaProgressionSystem) canUpdate(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, progressionConfig *ProgressionConfigProgression, userProgression *SyncProgressionUpdate) bool {
	// If progression is already unlocked (cost has been paid), it can be updated
	if userProgression != nil && userProgression.Cost != nil {
		return true
	}

	// Otherwise, check if all preconditions are met
	unmet := p.checkPreconditions(ctx, logger, nk, userID, progressionConfig.Preconditions, nil)
	return unmet == nil
}

// calculateDelta computes the difference between two progression states
func (p *NakamaProgressionSystem) calculateDelta(lastKnown, current *Progression) *ProgressionDelta {
	delta := &ProgressionDelta{
		Id: current.Id,
	}

	hasChanges := false

	// Check unlock state change
	if lastKnown.Unlocked != current.Unlocked {
		if current.Unlocked {
			delta.State = ProgressionDeltaState_PROGRESSION_DELTA_STATE_UNLOCKED
		} else {
			delta.State = ProgressionDeltaState_PROGRESSION_DELTA_STATE_LOCKED
		}
		hasChanges = true
	}

	// Check count changes
	if len(current.Counts) > 0 || len(lastKnown.Counts) > 0 {
		delta.Counts = make(map[string]int64)

		// Check for changed or new counts
		for countID, currentValue := range current.Counts {
			if lastValue, exists := lastKnown.Counts[countID]; !exists || lastValue != currentValue {
				delta.Counts[countID] = currentValue - lastValue
				hasChanges = true
			}
		}

		// Check for removed counts
		for countID, lastValue := range lastKnown.Counts {
			if _, exists := current.Counts[countID]; !exists {
				delta.Counts[countID] = -lastValue
				hasChanges = true
			}
		}
	}

	if !hasChanges {
		return nil
	}

	return delta
}

// applyScheduledResets applies any scheduled resets based on CRON expressions
func (p *NakamaProgressionSystem) applyScheduledResets(logger runtime.Logger, config *ProgressionConfigProgression, progression *Progression, lastUpdateTime, now int64) *Progression {
	if config.ResetSchedule == "" {
		return progression
	}

	// Calculate when the next reset should occur
	nextResetTime, err := p.calculateNextResetTime(config.ResetSchedule, time.Unix(lastUpdateTime, 0))
	if err != nil {
		logger.Error("Failed to parse CRON expression for progression: %v", err)
		return progression
	}

	// If we've passed the reset time, apply reset logic
	if nextResetTime > 0 && now >= nextResetTime {
		// Reset counts but keep unlock status
		progression.Counts = make(map[string]int64)
	}

	return progression
}

// calculateNextResetTime calculates the next reset time based on CRON expression
func (p *NakamaProgressionSystem) calculateNextResetTime(cronExpr string, now time.Time) (int64, error) {
	if cronExpr == "" {
		return 0, nil // No reset scheduled
	}

	sched, err := p.cronParser.Parse(cronExpr)
	if err != nil {
		return 0, err
	}

	nextReset := sched.Next(now)
	return nextReset.Unix(), nil
}

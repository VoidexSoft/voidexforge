package pamlogix

import (
	"context"
	"encoding/json"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/heroiclabs/nakama-common/runtime"
)

// Constants for storage
const (
	unlockablesStorageCollection = "unlockables"
	userUnlockablesStorageKey    = "user_unlockables"
)

// Unlockable status constants
const (
	UnlockableStatusLocked    = 0
	UnlockableStatusUnlocking = 1
	UnlockableStatusUnlocked  = 2
)

// UnlockablesPamlogix is an implementation of the UnlockablesSystem interface
type UnlockablesPamlogix struct {
	config        *UnlockablesConfig
	onClaimReward OnReward[*UnlockablesConfigUnlockable]
	pamlogix      Pamlogix
}

// NewUnlockablesSystem creates a new instance of UnlockablesPamlogix
func NewUnlockablesSystem(config *UnlockablesConfig) UnlockablesSystem {
	// Initialize probability distribution for random unlockable selection
	if config != nil && config.Unlockables != nil {
		config.UnlockableProbabilities = make([]string, 0, len(config.Unlockables))
		for id, unlockable := range config.Unlockables {
			// Add the ID to the list based on its probability
			for i := 0; i < unlockable.Probability; i++ {
				config.UnlockableProbabilities = append(config.UnlockableProbabilities, id)
			}
		}
	}

	return &UnlockablesPamlogix{
		config: config,
	}
}

// SetPamlogix sets the Pamlogix instance for this unlockables system
func (u *UnlockablesPamlogix) SetPamlogix(pl Pamlogix) {
	u.pamlogix = pl
}

// GetType returns the type of the system
func (u *UnlockablesPamlogix) GetType() SystemType {
	return SystemTypeUnlockables
}

// GetConfig returns the configuration of the system
func (u *UnlockablesPamlogix) GetConfig() any {
	return u.config
}

// getUserUnlockables fetches the stored unlockables data for a user from Nakama storage.
func (u *UnlockablesPamlogix) getUserUnlockables(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (*UnlockablesList, error) {
	// Read from storage
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: unlockablesStorageCollection,
			Key:        userUnlockablesStorageKey,
			UserID:     userID,
		},
	})

	if err != nil {
		logger.Error("Failed to read user unlockables: %v", err)
		return nil, err
	}

	// If no data found, return new empty unlockables list
	if len(objects) == 0 || objects[0].Value == "" {
		return &UnlockablesList{
			Slots:            int32(u.config.Slots),
			ActiveSlots:      int32(u.config.ActiveSlots),
			MaxActiveSlots:   int32(u.config.MaxActiveSlots),
			Unlockables:      make([]*Unlockable, 0),
			QueuedUnlocks:    make([]string, 0),
			MaxQueuedUnlocks: int32(u.config.MaxQueuedUnlocks),
		}, nil
	}

	// Unmarshal the stored unlockables data
	unlockables := &UnlockablesList{}
	if err := json.Unmarshal([]byte(objects[0].Value), unlockables); err != nil {
		logger.Error("Failed to unmarshal user unlockables: %v", err)
		return nil, err
	}

	// If no unlockables or queue initialized, create them
	if unlockables.Unlockables == nil {
		unlockables.Unlockables = make([]*Unlockable, 0)
	}
	if unlockables.QueuedUnlocks == nil {
		unlockables.QueuedUnlocks = make([]string, 0)
	}

	return unlockables, nil
}

// saveUserUnlockables stores the updated unlockables data for a user in Nakama storage.
func (u *UnlockablesPamlogix) saveUserUnlockables(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, unlockables *UnlockablesList) error {
	// Marshal the unlockables data
	data, err := json.Marshal(unlockables)
	if err != nil {
		logger.Error("Failed to marshal user unlockables: %v", err)
		return err
	}

	// Write to storage
	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{
		{
			Collection:      unlockablesStorageCollection,
			Key:             userUnlockablesStorageKey,
			UserID:          userID,
			Value:           string(data),
			PermissionRead:  runtime.STORAGE_PERMISSION_OWNER_READ,
			PermissionWrite: runtime.STORAGE_PERMISSION_OWNER_WRITE,
		},
	})

	if err != nil {
		logger.Error("Failed to write user unlockables: %v", err)
		return err
	}

	return nil
}

// createUnlockable creates a new unlockable for the given unlockable ID or config
func (u *UnlockablesPamlogix) createUnlockable(unlockableID string, unlockableConfig *UnlockablesConfigUnlockable) *Unlockable {
	// If no custom config is provided, use the one from the system config
	config := unlockableConfig
	if config == nil && unlockableID != "" {
		if cfg, exists := u.config.Unlockables[unlockableID]; exists {
			config = cfg
		}
	}

	// If we still don't have a config, return nil
	if config == nil {
		return nil
	}

	// Generate a unique instance ID
	instanceID := uuid.New().String()

	// Create a new unlockable
	now := time.Now().Unix()
	unlockable := &Unlockable{
		Id:                   unlockableID,
		InstanceId:           instanceID,
		Category:             config.Category,
		Name:                 config.Name,
		Description:          config.Description,
		WaitTimeSec:          int32(config.WaitTimeSec),
		CreateTimeSec:        now,
		AdditionalProperties: make(map[string]string),
	}

	// Copy additional properties if they exist
	if config.AdditionalProperties != nil {
		for k, v := range config.AdditionalProperties {
			unlockable.AdditionalProperties[k] = v
		}
	}

	// Set costs if they exist
	if config.StartCost != nil {
		unlockable.StartCost = &UnlockableCost{
			Items:      config.StartCost.Items,
			Currencies: config.StartCost.Currencies,
		}
	}

	if config.Cost != nil {
		unlockable.Cost = &UnlockableCost{
			Items:      config.Cost.Items,
			Currencies: config.Cost.Currencies,
		}
	}

	return unlockable
}

// findUnlockableByID finds an unlockable with the given instance ID
func (u *UnlockablesPamlogix) findUnlockableByID(unlockables []*Unlockable, instanceID string) (int, *Unlockable) {
	for i, unlockable := range unlockables {
		if unlockable.InstanceId == instanceID {
			return i, unlockable
		}
	}
	return -1, nil
}

// countActiveUnlocks counts how many unlockables are currently being unlocked
func (u *UnlockablesPamlogix) countActiveUnlocks(unlockables []*Unlockable) int {
	count := 0
	for _, unlockable := range unlockables {
		if unlockable.UnlockStartTimeSec > 0 && !unlockable.CanClaim {
			count++
		}
	}
	return count
}

// selectRandomUnlockable selects a random unlockable ID based on probability distribution
func (u *UnlockablesPamlogix) selectRandomUnlockable() string {
	if len(u.config.UnlockableProbabilities) == 0 {
		return ""
	}

	idx := rand.Intn(len(u.config.UnlockableProbabilities))
	return u.config.UnlockableProbabilities[idx]
}

// getUnlockableConfig returns the configuration for a given unlockable ID
func (u *UnlockablesPamlogix) getUnlockableConfig(unlockableID string) *UnlockablesConfigUnlockable {
	if u.config == nil || u.config.Unlockables == nil {
		return nil
	}

	config, exists := u.config.Unlockables[unlockableID]
	if !exists {
		return nil
	}

	return config
}

// processQueue moves items from the queue to active slots if possible
func (u *UnlockablesPamlogix) processQueue(unlockables *UnlockablesList) bool {
	if len(unlockables.QueuedUnlocks) == 0 {
		return false // No items in queue
	}

	// Count active unlocks
	activeCount := u.countActiveUnlocks(unlockables.Unlockables)

	// If we already have the maximum number of active unlocks, can't process queue
	if activeCount >= int(unlockables.ActiveSlots) {
		return false
	}

	// Move items from queue to active slots
	for i := 0; i < len(unlockables.QueuedUnlocks) && activeCount < int(unlockables.ActiveSlots); i++ {
		instanceID := unlockables.QueuedUnlocks[0]
		_, unlockable := u.findUnlockableByID(unlockables.Unlockables, instanceID)

		if unlockable != nil && unlockable.UnlockStartTimeSec == 0 && !unlockable.CanClaim {
			// Start unlocking
			now := time.Now().Unix()
			unlockable.UnlockStartTimeSec = now
			unlockable.UnlockCompleteTimeSec = now + int64(unlockable.WaitTimeSec)

			// Remove from queue
			unlockables.QueuedUnlocks = unlockables.QueuedUnlocks[1:]

			// Increment active count
			activeCount++

			return true // Successfully moved an item from queue
		} else {
			// Invalid unlockable or already started/completed, remove from queue
			unlockables.QueuedUnlocks = unlockables.QueuedUnlocks[1:]
		}
	}

	return false
}

// updateUnlockProgress updates the progress of all unlocking unlockables
func (u *UnlockablesPamlogix) updateUnlockProgress(unlockables *UnlockablesList) bool {
	now := time.Now().Unix()
	updated := false

	for _, unlockable := range unlockables.Unlockables {
		// Check if unlockable is in the unlocking state
		if unlockable.UnlockStartTimeSec > 0 && !unlockable.CanClaim {
			// If unlock complete time has been reached, mark as can claim
			if now >= unlockable.UnlockCompleteTimeSec {
				unlockable.CanClaim = true
				updated = true
			}
		}
	}

	return updated
}

// checkUserHasResources checks if a user has sufficient resources to pay a cost
func (u *UnlockablesPamlogix) checkUserHasResources(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string,
	items map[string]int64, currencies map[string]int64) (bool, error) {

	// Skip check if no costs
	if (items == nil || len(items) == 0) && (currencies == nil || len(currencies) == 0) {
		return true, nil
	}

	// Check if we have access to the necessary systems
	if u.pamlogix == nil {
		logger.Warn("Cannot check resources: no Pamlogix instance available")
		return false, ErrSystemNotAvailable
	}

	// Check item costs if any
	if items != nil && len(items) > 0 {
		inventorySystem := u.pamlogix.GetInventorySystem()
		if inventorySystem == nil {
			logger.Warn("Cannot check item resources: no InventorySystem available")
			return false, ErrSystemNotAvailable
		}

		// Get user's inventory
		inventory, err := inventorySystem.ListInventoryItems(ctx, logger, nk, userID, "")
		if err != nil {
			logger.Error("Failed to get user inventory: %v", err)
			return false, err
		}

		// Check if user has enough of each required item
		for itemID, amount := range items {
			found := false
			for _, item := range inventory.Items {
				if item.Id == itemID && item.Count >= amount {
					found = true
					break
				}
			}

			if !found {
				logger.Debug("User %s does not have enough of item %s", userID, itemID)
				return false, nil
			}
		}
	}

	// Check currency costs if any
	if currencies != nil && len(currencies) > 0 {
		economySystem := u.pamlogix.GetEconomySystem()
		if economySystem == nil {
			logger.Warn("Cannot check currency resources: no EconomySystem available")
			return false, ErrSystemNotAvailable
		}

		// Get user's account to check wallet
		account, err := nk.AccountGetId(ctx, userID)
		if err != nil {
			logger.Error("Failed to get user account: %v", err)
			return false, err
		}

		// Get wallet from account
		wallet, err := economySystem.UnmarshalWallet(account)
		if err != nil {
			logger.Error("Failed to unmarshal wallet: %v", err)
			return false, err
		}

		// Check if user has enough of each required currency
		for currencyID, amount := range currencies {
			if wallet[currencyID] < amount {
				logger.Debug("User %s does not have enough of currency %s", userID, currencyID)
				return false, nil
			}
		}
	}

	return true, nil
}

// deductResources deducts the specified resources from a user
func (u *UnlockablesPamlogix) deductResources(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string,
	items map[string]int64, currencies map[string]int64) error {

	// Skip if no costs
	if (items == nil || len(items) == 0) && (currencies == nil || len(currencies) == 0) {
		return nil
	}

	// Check if we have access to the necessary systems
	if u.pamlogix == nil {
		logger.Warn("Cannot deduct resources: no Pamlogix instance available")
		return ErrSystemNotAvailable
	}

	// Deduct items if any
	if items != nil && len(items) > 0 {
		inventorySystem := u.pamlogix.GetInventorySystem()
		if inventorySystem == nil {
			logger.Warn("Cannot deduct item resources: no InventorySystem available")
			return ErrSystemNotAvailable
		}

		// Convert to negative values for subtraction
		deductItems := make(map[string]int64)
		for id, amount := range items {
			deductItems[id] = -amount
		}

		// Deduct items
		_, _, _, _, err := inventorySystem.GrantItems(ctx, logger, nk, userID, deductItems, false)
		if err != nil {
			logger.Error("Failed to deduct items: %v", err)
			return err
		}
	}

	// Deduct currencies if any
	if currencies != nil && len(currencies) > 0 {
		economySystem := u.pamlogix.GetEconomySystem()
		if economySystem == nil {
			logger.Warn("Cannot deduct currency resources: no EconomySystem available")
			return ErrSystemNotAvailable
		}

		// Convert to negative values for subtraction
		deductCurrencies := make(map[string]int64)
		for id, amount := range currencies {
			deductCurrencies[id] = -amount
		}

		// Deduct currencies
		_, _, _, err := economySystem.Grant(ctx, logger, nk, userID, deductCurrencies, nil, nil, nil)
		if err != nil {
			logger.Error("Failed to deduct currencies: %v", err)
			return err
		}
	}

	return nil
}

// Create will place a new unlockable into a slot either randomly, by ID, or optionally using a custom configuration.
func (u *UnlockablesPamlogix) Create(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, unlockableID string, unlockableConfig *UnlockablesConfigUnlockable) (unlockables *UnlockablesList, err error) {
	logger.Info("Creating unlockable for user: %s, unlockableID: %s", userID, unlockableID)

	// Check if the configuration is valid
	if u.config == nil {
		logger.Error("No unlockables configuration available")
		return nil, ErrSystemNotAvailable
	}

	// Retrieve user's current unlockables
	unlockables, err = u.getUserUnlockables(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user's unlockables: %v", err)
		return nil, err
	}

	// Update progress of existing unlockables
	u.updateUnlockProgress(unlockables)

	// Check if user has available slots (total number of unlockables < slots)
	if len(unlockables.Unlockables) >= int(unlockables.Slots) {
		logger.Info("User %s has no available slots", userID)
		return unlockables, ErrBadInput
	}

	// Determine which unlockable to create
	finalUnlockableID := unlockableID
	finalUnlockableConfig := unlockableConfig

	// If no unlockable ID is provided, select one randomly
	if finalUnlockableID == "" && finalUnlockableConfig == nil {
		finalUnlockableID = u.selectRandomUnlockable()
		if finalUnlockableID == "" {
			logger.Error("Failed to select a random unlockable")
			return unlockables, ErrSystemNotAvailable
		}
	}

	// Create the unlockable instance
	unlockable := u.createUnlockable(finalUnlockableID, finalUnlockableConfig)
	if unlockable == nil {
		logger.Error("Failed to create unlockable")
		return unlockables, ErrBadInput
	}

	// Add the new unlockable to the user's unlockables
	unlockables.Unlockables = append(unlockables.Unlockables, unlockable)

	// Store the newly created instance ID
	unlockables.InstanceId = unlockable.InstanceId

	// Check if we can immediately start unlocking from queue
	u.processQueue(unlockables)

	// Save the updated unlockables to storage
	if err := u.saveUserUnlockables(ctx, logger, nk, userID, unlockables); err != nil {
		logger.Error("Failed to save user unlockables: %v", err)
		return nil, err
	}

	return unlockables, nil
}

// Get returns all unlockables active for a user by ID.
func (u *UnlockablesPamlogix) Get(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (unlockables *UnlockablesList, err error) {
	logger.Info("Getting unlockables for user: %s", userID)

	// Retrieve user's unlockables from storage
	unlockables, err = u.getUserUnlockables(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user's unlockables: %v", err)
		return nil, err
	}

	// Update the progress of all unlocking unlockables
	updated := u.updateUnlockProgress(unlockables)

	// If any unlockables were completed, try to start new ones from the queue
	if updated {
		u.processQueue(unlockables)

		// Save the updated unlockables to storage
		if err := u.saveUserUnlockables(ctx, logger, nk, userID, unlockables); err != nil {
			logger.Error("Failed to save user unlockables: %v", err)
			return nil, err
		}
	}

	return unlockables, nil
}

// UnlockAdvance will add the given amount of time towards the completion of an unlockable that has been started.
func (u *UnlockablesPamlogix) UnlockAdvance(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, instanceID string, seconds int64) (unlockables *UnlockablesList, err error) {
	logger.Info("Advancing unlock for user: %s, instanceID: %s, seconds: %d", userID, instanceID, seconds)

	// Retrieve user's unlockables
	unlockables, err = u.getUserUnlockables(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user's unlockables: %v", err)
		return nil, err
	}

	// Find the unlockable with the given instance ID
	idx, unlockable := u.findUnlockableByID(unlockables.Unlockables, instanceID)
	if idx == -1 || unlockable == nil {
		logger.Error("Could not find unlockable with instance ID %s for user %s", instanceID, userID)
		return unlockables, ErrBadInput
	}

	// Check if the unlockable has been started
	if unlockable.UnlockStartTimeSec == 0 {
		logger.Error("Unlockable %s has not been started for user %s", instanceID, userID)
		return unlockables, ErrBadInput
	}

	// Check if the unlockable is already completed
	if unlockable.CanClaim {
		logger.Info("Unlockable %s is already completed for user %s", instanceID, userID)
		return unlockables, nil
	}

	// Add the time advance to the unlockable
	unlockable.AdvanceTimeSec += seconds

	// Check if the advance completes the unlock
	now := time.Now().Unix()
	currentProgress := now - unlockable.UnlockStartTimeSec + unlockable.AdvanceTimeSec

	if currentProgress >= int64(unlockable.WaitTimeSec) {
		// Mark as completed
		unlockable.CanClaim = true
		unlockable.UnlockCompleteTimeSec = now // Mark as completed now
	} else {
		// Update the unlock complete time
		unlockable.UnlockCompleteTimeSec = unlockable.UnlockStartTimeSec + int64(unlockable.WaitTimeSec) - unlockable.AdvanceTimeSec
	}

	// If an unlockable was completed, try to start new ones from the queue
	if unlockable.CanClaim {
		u.processQueue(unlockables)
	}

	// Save the updated unlockables
	if err := u.saveUserUnlockables(ctx, logger, nk, userID, unlockables); err != nil {
		logger.Error("Failed to save user unlockables: %v", err)
		return nil, err
	}

	return unlockables, nil
}

// UnlockStart will begin an unlock of an unlockable by instance ID for a user.
func (u *UnlockablesPamlogix) UnlockStart(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, instanceID string) (unlockables *UnlockablesList, err error) {
	logger.Info("Starting unlock for user: %s, instanceID: %s", userID, instanceID)

	// Retrieve user's unlockables
	unlockables, err = u.getUserUnlockables(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user's unlockables: %v", err)
		return nil, err
	}

	// Find the unlockable with the given instance ID
	idx, unlockable := u.findUnlockableByID(unlockables.Unlockables, instanceID)
	if idx == -1 || unlockable == nil {
		logger.Error("Could not find unlockable with instance ID %s for user %s", instanceID, userID)
		return unlockables, ErrBadInput
	}

	// Check if the unlockable is already started or completed
	if unlockable.UnlockStartTimeSec > 0 {
		logger.Error("Unlockable %s is already started for user %s", instanceID, userID)
		return unlockables, ErrBadInput
	}

	// Count active unlocks
	activeCount := u.countActiveUnlocks(unlockables.Unlockables)

	// Check if user has available active slots
	if activeCount >= int(unlockables.ActiveSlots) {
		logger.Error("User %s has no available active slots", userID)
		return unlockables, ErrBadInput
	}

	// Check if the user has sufficient resources to pay the start cost
	if unlockable.StartCost != nil {
		// Check if user has enough resources
		hasResources, err := u.checkUserHasResources(ctx, logger, nk, userID, unlockable.StartCost.Items, unlockable.StartCost.Currencies)
		if err != nil {
			logger.Error("Failed to check user resources: %v", err)
			return nil, err
		}

		if !hasResources {
			logger.Error("User %s does not have enough resources to start unlocking %s", userID, instanceID)
			return unlockables, ErrEconomyNotEnoughCurrency
		}

		// Deduct the start cost
		if err := u.deductResources(ctx, logger, nk, userID, unlockable.StartCost.Items, unlockable.StartCost.Currencies); err != nil {
			logger.Error("Failed to deduct resources: %v", err)
			return nil, err
		}
	}

	// Start the unlock
	now := time.Now().Unix()
	unlockable.UnlockStartTimeSec = now
	unlockable.UnlockCompleteTimeSec = now + int64(unlockable.WaitTimeSec)

	// Save the updated unlockables
	if err := u.saveUserUnlockables(ctx, logger, nk, userID, unlockables); err != nil {
		logger.Error("Failed to save user unlockables: %v", err)
		return nil, err
	}

	return unlockables, nil
}

// PurchaseUnlock will immediately unlock an unlockable with the specified instance ID for a user.
func (u *UnlockablesPamlogix) PurchaseUnlock(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, instanceID string) (unlockables *UnlockablesList, err error) {
	logger.Info("Purchasing unlock for user: %s, instanceID: %s", userID, instanceID)

	// Retrieve user's unlockables
	unlockables, err = u.getUserUnlockables(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user's unlockables: %v", err)
		return nil, err
	}

	// Find the unlockable with the given instance ID
	idx, unlockable := u.findUnlockableByID(unlockables.Unlockables, instanceID)
	if idx == -1 || unlockable == nil {
		logger.Error("Could not find unlockable with instance ID %s for user %s", instanceID, userID)
		return unlockables, ErrBadInput
	}

	// Check if the unlockable is already completed
	if unlockable.CanClaim {
		logger.Info("Unlockable %s is already completed for user %s", instanceID, userID)
		return unlockables, nil
	}

	// Determine the cost to complete the unlock
	var costItems map[string]int64
	var costCurrencies map[string]int64

	if unlockable.Cost != nil {
		costItems = unlockable.Cost.Items
		costCurrencies = unlockable.Cost.Currencies
	}

	// If we have started the unlock but not completed it, pro-rate the cost based on remaining time
	if unlockable.UnlockStartTimeSec > 0 {
		now := time.Now().Unix()
		totalTime := int64(unlockable.WaitTimeSec)
		elapsedTime := now - unlockable.UnlockStartTimeSec + unlockable.AdvanceTimeSec

		if elapsedTime < totalTime && totalTime > 0 {
			// Calculate proportion of time remaining
			remainingProportion := float64(totalTime-elapsedTime) / float64(totalTime)

			// Pro-rate the cost
			if costItems != nil {
				proRatedItems := make(map[string]int64)
				for id, amount := range costItems {
					proRatedItems[id] = int64(float64(amount) * remainingProportion)
					if proRatedItems[id] <= 0 {
						proRatedItems[id] = 1 // Minimum cost of 1
					}
				}
				costItems = proRatedItems
			}

			if costCurrencies != nil {
				proRatedCurrencies := make(map[string]int64)
				for id, amount := range costCurrencies {
					proRatedCurrencies[id] = int64(float64(amount) * remainingProportion)
					if proRatedCurrencies[id] <= 0 {
						proRatedCurrencies[id] = 1 // Minimum cost of 1
					}
				}
				costCurrencies = proRatedCurrencies
			}
		}
	}

	// Check if the user has sufficient resources to pay the unlock cost
	if costItems != nil || costCurrencies != nil {
		// Check if user has enough resources
		hasResources, err := u.checkUserHasResources(ctx, logger, nk, userID, costItems, costCurrencies)
		if err != nil {
			logger.Error("Failed to check user resources: %v", err)
			return nil, err
		}

		if !hasResources {
			logger.Error("User %s does not have enough resources to purchase unlock %s", userID, instanceID)
			return unlockables, ErrEconomyNotEnoughCurrency
		}

		// Deduct the cost
		if err := u.deductResources(ctx, logger, nk, userID, costItems, costCurrencies); err != nil {
			logger.Error("Failed to deduct resources: %v", err)
			return nil, err
		}
	}

	// Complete the unlock immediately
	now := time.Now().Unix()
	unlockable.CanClaim = true

	// If not already started, set the start time to now
	if unlockable.UnlockStartTimeSec == 0 {
		unlockable.UnlockStartTimeSec = now
	}

	unlockable.UnlockCompleteTimeSec = now

	// Try to process the queue if we freed up a slot
	u.processQueue(unlockables)

	// Save the updated unlockables
	if err := u.saveUserUnlockables(ctx, logger, nk, userID, unlockables); err != nil {
		logger.Error("Failed to save user unlockables: %v", err)
		return nil, err
	}

	return unlockables, nil
}

// PurchaseSlot will create a new slot for a user by ID.
func (u *UnlockablesPamlogix) PurchaseSlot(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (unlockables *UnlockablesList, err error) {
	logger.Info("Purchasing slot for user: %s", userID)

	// Retrieve user's unlockables
	unlockables, err = u.getUserUnlockables(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user's unlockables: %v", err)
		return nil, err
	}

	// Check if the user has reached the maximum number of active slots
	if unlockables.ActiveSlots >= unlockables.MaxActiveSlots {
		logger.Error("User %s has reached the maximum number of active slots", userID)
		return unlockables, ErrBadInput
	}

	// Check if the user has sufficient resources to pay the slot cost
	if unlockables.SlotCost != nil {
		// Check if user has enough resources
		hasResources, err := u.checkUserHasResources(ctx, logger, nk, userID, unlockables.SlotCost.Items, unlockables.SlotCost.Currencies)
		if err != nil {
			logger.Error("Failed to check user resources: %v", err)
			return nil, err
		}

		if !hasResources {
			logger.Error("User %s does not have enough resources to purchase a new slot", userID)
			return unlockables, ErrEconomyNotEnoughCurrency
		}

		// Deduct the cost
		if err := u.deductResources(ctx, logger, nk, userID, unlockables.SlotCost.Items, unlockables.SlotCost.Currencies); err != nil {
			logger.Error("Failed to deduct resources: %v", err)
			return nil, err
		}
	}

	// Increase the user's number of active slots
	unlockables.ActiveSlots++

	// Check if we can now start unlocking items from the queue
	u.processQueue(unlockables)

	// Save the updated unlockables
	if err := u.saveUserUnlockables(ctx, logger, nk, userID, unlockables); err != nil {
		logger.Error("Failed to save user unlockables: %v", err)
		return nil, err
	}

	return unlockables, nil
}

// Claim an unlockable which has been unlocked by instance ID for the user.
func (u *UnlockablesPamlogix) Claim(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, instanceID string) (reward *UnlockablesReward, err error) {
	logger.Info("Claiming unlockable for user: %s, instanceID: %s", userID, instanceID)

	// Initialize reward
	reward = &UnlockablesReward{
		Reward: &Reward{
			Items:           make(map[string]int64),
			Currencies:      make(map[string]int64),
			Energies:        make(map[string]int32),
			EnergyModifiers: make([]*RewardEnergyModifier, 0),
			RewardModifiers: make([]*RewardModifier, 0),
			GrantTimeSec:    time.Now().Unix(),
			ItemInstances:   make(map[string]*RewardInventoryItem),
		},
	}

	// Retrieve user's unlockables
	unlockables, err := u.getUserUnlockables(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user's unlockables: %v", err)
		return nil, err
	}

	// Find the unlockable with the given instance ID
	idx, unlockable := u.findUnlockableByID(unlockables.Unlockables, instanceID)
	if idx == -1 || unlockable == nil {
		logger.Error("Could not find unlockable with instance ID %s for user %s", instanceID, userID)
		return nil, ErrBadInput
	}

	// Check if the unlockable is in the unlocked state
	if !unlockable.CanClaim {
		logger.Error("Unlockable %s is not ready to be claimed for user %s", instanceID, userID)
		return nil, ErrBadInput
	}

	// Get the unlockable configuration to generate rewards
	unlockableID := unlockable.Id
	unlockableConfig := u.getUnlockableConfig(unlockableID)

	// Generate rewards based on the unlockable's configuration
	if unlockableConfig != nil && unlockableConfig.Reward != nil {
		// Get the economy system from the Pamlogix instance
		var economySystem EconomySystem

		if u.pamlogix != nil {
			economySystem = u.pamlogix.GetEconomySystem()
		}

		// If we couldn't get the economy system from Pamlogix, return an error
		if economySystem == nil {
			logger.Error("No economy system available through Pamlogix")
			return nil, ErrSystemNotAvailable
		}

		// Roll the reward using the reward configuration
		rolledReward, err := economySystem.RewardRoll(ctx, logger, nk, userID, unlockableConfig.Reward)
		if err != nil {
			logger.Error("Failed to roll reward for unlockable: %v", err)
			// Continue with empty reward
		} else if rolledReward != nil {
			// Use the rolled reward
			reward.Reward = rolledReward
		}

		// If a custom reward function is set, let it modify the reward
		if u.onClaimReward != nil {
			customReward, err := u.onClaimReward(ctx, logger, nk, userID, instanceID, unlockableConfig, unlockableConfig.Reward, reward.Reward)
			if err != nil {
				logger.Error("Custom reward function failed: %v", err)
				return nil, err
			}
			if customReward != nil {
				reward.Reward = customReward
			}
		}

		// Apply the reward to the user's account
		if reward.Reward != nil {
			newItems, updatedItems, notGrantedItemIDs, err := economySystem.RewardGrant(ctx, logger, nk, userID, reward.Reward, nil, false)
			if err != nil {
				logger.Error("Failed to grant reward: %v", err)
				return nil, err
			}

			logger.Debug("Granted %d new items, updated %d items, failed to grant %d items", len(newItems), len(updatedItems), len(notGrantedItemIDs))
		}
	}

	// Remove the unlockable from the user's unlockables
	unlockables.Unlockables = append(unlockables.Unlockables[:idx], unlockables.Unlockables[idx+1:]...)

	// Move an unlockable from the queue to an active slot if available
	u.processQueue(unlockables)

	// Save the updated unlockables
	if err := u.saveUserUnlockables(ctx, logger, nk, userID, unlockables); err != nil {
		logger.Error("Failed to save user unlockables: %v", err)
		return nil, err
	}

	// Set the updated unlockables list in the response
	reward.Unlockables = unlockables

	return reward, nil
}

// QueueAdd adds one or more unlockable instance IDs to the queue to be unlocked as soon as an active slot is available.
func (u *UnlockablesPamlogix) QueueAdd(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, instanceIDs []string) (unlockables *UnlockablesList, err error) {
	logger.Info("Adding to queue for user: %s, instanceIDs: %v", userID, instanceIDs)

	// Retrieve user's unlockables
	unlockables, err = u.getUserUnlockables(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user's unlockables: %v", err)
		return nil, err
	}

	// Check if each instance ID exists and is not already queued or unlocking
	for _, instanceID := range instanceIDs {
		// Check if the unlockable exists
		idx, unlockable := u.findUnlockableByID(unlockables.Unlockables, instanceID)
		if idx == -1 || unlockable == nil {
			logger.Error("Could not find unlockable with instance ID %s for user %s", instanceID, userID)
			continue
		}

		// Check if the unlockable is already started or completed
		if unlockable.UnlockStartTimeSec > 0 || unlockable.CanClaim {
			logger.Error("Unlockable %s is already started or completed for user %s", instanceID, userID)
			continue
		}

		// Check if the unlockable is already in the queue
		alreadyQueued := false
		for _, queuedID := range unlockables.QueuedUnlocks {
			if queuedID == instanceID {
				alreadyQueued = true
				break
			}
		}

		if alreadyQueued {
			logger.Error("Unlockable %s is already in the queue for user %s", instanceID, userID)
			continue
		}

		// Check if we've reached the max queue size
		if int32(len(unlockables.QueuedUnlocks)) >= unlockables.MaxQueuedUnlocks {
			logger.Error("Queue is full for user %s", userID)
			break
		}

		// Add to queue
		unlockables.QueuedUnlocks = append(unlockables.QueuedUnlocks, instanceID)
	}

	// Try to start unlocking items from the queue
	u.processQueue(unlockables)

	// Save the updated unlockables
	if err := u.saveUserUnlockables(ctx, logger, nk, userID, unlockables); err != nil {
		logger.Error("Failed to save user unlockables: %v", err)
		return nil, err
	}

	return unlockables, nil
}

// QueueRemove removes one or more unlockable instance IDs from the unlock queue, unless they have started unlocking already.
func (u *UnlockablesPamlogix) QueueRemove(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, instanceIDs []string) (unlockables *UnlockablesList, err error) {
	logger.Info("Removing from queue for user: %s, instanceIDs: %v", userID, instanceIDs)

	// Retrieve user's unlockables
	unlockables, err = u.getUserUnlockables(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user's unlockables: %v", err)
		return nil, err
	}

	// If no instance IDs to remove, return early
	if len(instanceIDs) == 0 {
		return unlockables, nil
	}

	// Create a map for quick lookup of IDs to remove
	removeIDs := make(map[string]bool)
	for _, id := range instanceIDs {
		removeIDs[id] = true
	}

	// Filter the queue to remove the specified IDs
	newQueue := make([]string, 0, len(unlockables.QueuedUnlocks))
	for _, queuedID := range unlockables.QueuedUnlocks {
		if !removeIDs[queuedID] {
			newQueue = append(newQueue, queuedID)
		}
	}

	// Update the queue
	unlockables.QueuedUnlocks = newQueue

	// Save the updated unlockables
	if err := u.saveUserUnlockables(ctx, logger, nk, userID, unlockables); err != nil {
		logger.Error("Failed to save user unlockables: %v", err)
		return nil, err
	}

	return unlockables, nil
}

// QueueSet replaces the entirety of the queue with the specified instance IDs, or wipes the queue if no instance IDs are given.
func (u *UnlockablesPamlogix) QueueSet(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, instanceIDs []string) (unlockables *UnlockablesList, err error) {
	logger.Info("Setting queue for user: %s, instanceIDs: %v", userID, instanceIDs)

	// Retrieve user's unlockables
	unlockables, err = u.getUserUnlockables(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user's unlockables: %v", err)
		return nil, err
	}

	// Clear the existing queue
	unlockables.QueuedUnlocks = make([]string, 0)

	// If no instance IDs provided, just clear the queue
	if len(instanceIDs) == 0 {
		if err := u.saveUserUnlockables(ctx, logger, nk, userID, unlockables); err != nil {
			logger.Error("Failed to save user unlockables: %v", err)
			return nil, err
		}
		return unlockables, nil
	}

	// Filter valid instance IDs (must exist, not be unlocking, not be unlocked)
	validInstanceIDs := make([]string, 0, len(instanceIDs))

	for _, instanceID := range instanceIDs {
		// Check if the unlockable exists
		idx, unlockable := u.findUnlockableByID(unlockables.Unlockables, instanceID)
		if idx == -1 || unlockable == nil {
			logger.Error("Could not find unlockable with instance ID %s for user %s", instanceID, userID)
			continue
		}

		// Check if the unlockable is already started or completed
		if unlockable.UnlockStartTimeSec > 0 || unlockable.CanClaim {
			logger.Error("Unlockable %s is already started or completed for user %s", instanceID, userID)
			continue
		}

		// Check if we've reached the max queue size
		if int32(len(validInstanceIDs)) >= unlockables.MaxQueuedUnlocks {
			logger.Error("Queue is full for user %s", userID)
			break
		}

		// Add to valid IDs
		validInstanceIDs = append(validInstanceIDs, instanceID)
	}

	// Set the queue to the valid IDs
	unlockables.QueuedUnlocks = validInstanceIDs

	// Try to start unlocking items from the queue
	u.processQueue(unlockables)

	// Save the updated unlockables
	if err := u.saveUserUnlockables(ctx, logger, nk, userID, unlockables); err != nil {
		logger.Error("Failed to save user unlockables: %v", err)
		return nil, err
	}

	return unlockables, nil
}

// SetOnClaimReward sets a custom reward function which will run after an unlockable's reward is rolled.
func (u *UnlockablesPamlogix) SetOnClaimReward(fn OnReward[*UnlockablesConfigUnlockable]) {
	u.onClaimReward = fn
}

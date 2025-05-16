package pamlogix

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
)

// NakamaEnergySystem implements the EnergySystem interface using Nakama as the backend.
type NakamaEnergySystem struct {
	config        *EnergyConfig
	onSpendReward OnReward[*EnergyConfigEnergy]
	pamlogix      Pamlogix
}

// NewNakamaEnergySystem creates a new instance of the energy system with the given configuration.
func NewNakamaEnergySystem(config *EnergyConfig) *NakamaEnergySystem {
	return &NakamaEnergySystem{
		config: config,
	}
}

// SetPamlogix sets the Pamlogix instance for this energy system
func (e *NakamaEnergySystem) SetPamlogix(pl Pamlogix) {
	e.pamlogix = pl
}

// GetType returns the system type for the energy system.
func (e *NakamaEnergySystem) GetType() SystemType {
	return SystemTypeEnergy
}

// GetConfig returns the configuration for the energy system.
func (e *NakamaEnergySystem) GetConfig() any {
	return e.config
}

// Get returns all energies defined and the values a user currently owns by ID.
func (e *NakamaEnergySystem) Get(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (map[string]*Energy, error) {
	if e.config == nil || len(e.config.Energies) == 0 {
		// No energies are configured
		return make(map[string]*Energy), nil
	}

	// Fetch existing energy state from storage
	energies, err := e.getUserEnergies(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user energies: %v", err)
		return nil, ErrInternal
	}

	// Current timestamp
	now := time.Now().Unix()

	// Ensure all configured energies are represented and calculate refill values
	for id, energyConfig := range e.config.Energies {
		if _, exists := energies[id]; !exists {
			// Create a new energy entry with default values
			energies[id] = &Energy{
				Id:                   id,
				Current:              energyConfig.StartCount,
				Max:                  energyConfig.MaxCount,
				Refill:               energyConfig.RefillCount,
				RefillSec:            energyConfig.RefillTimeSec,
				NextRefillTimeSec:    0,
				MaxRefillTimeSec:     0,
				StartRefillTimeSec:   now,
				CurrentTimeSec:       now,
				AdditionalProperties: energyConfig.AdditionalProperties,
			}
		}

		// Update current time
		energies[id].CurrentTimeSec = now

		// Apply any active modifiers to max energy and refill rate
		e.applyActiveModifiers(energies[id], now)

		// Apply any refills that should have occurred
		e.applyRefills(energies[id], now)

		// Calculate next refill time
		e.calculateRefillTimes(energies[id], now)
	}

	return energies, nil
}

// Spend will deduct the amounts from each energy for a user by ID.
func (e *NakamaEnergySystem) Spend(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, amounts map[string]int32) (map[string]*Energy, *Reward, error) {
	if e.config == nil || len(e.config.Energies) == 0 {
		// No energies are configured
		return make(map[string]*Energy), nil, ErrSystemNotAvailable
	}

	// Fetch current energy values
	energies, err := e.Get(ctx, logger, nk, userID)
	if err != nil {
		return nil, nil, err
	}

	// Validate and apply the spend amounts
	for id, amount := range amounts {
		_, exists := e.config.Energies[id]
		if !exists {
			logger.Warn("Attempted to spend non-existent energy: %s", id)
			return nil, nil, ErrBadInput
		}

		energy, exists := energies[id]
		if !exists {
			logger.Warn("Energy exists in config but not for user: %s", id)
			return nil, nil, ErrBadInput
		}

		// Check if user has enough energy to spend
		if energy.Current < amount {
			logger.Warn("Insufficient energy to spend: %s (have: %d, need: %d)", id, energy.Current, amount)
			return nil, nil, ErrBadInput
		}

		// Deduct the energy
		energy.Current -= amount

		// If we reach 0 and there's a refill timer, set the start refill time
		if energy.Current == 0 && energy.RefillSec > 0 {
			now := time.Now().Unix()
			energy.StartRefillTimeSec = now
			energy.NextRefillTimeSec = now + energy.RefillSec
			energy.MaxRefillTimeSec = now + (energy.RefillSec * int64((energy.Max / energy.Refill)))
			if energy.Max%energy.Refill > 0 {
				energy.MaxRefillTimeSec += energy.RefillSec
			}
		}
	}

	// Save the updated energy values
	if err := e.saveUserEnergies(ctx, logger, nk, userID, energies); err != nil {
		logger.Error("Failed to save user energies: %v", err)
		return nil, nil, ErrInternal
	}

	// Process reward if applicable
	var reward *Reward = nil

	// Check if any of the spent energies has a reward configured
	for id, amount := range amounts {
		energyConfig, exists := e.config.Energies[id]
		if !exists || energyConfig.Reward == nil {
			continue
		}

		// Process potential reward
		r, err := e.processEnergyReward(ctx, logger, nk, userID, id, energyConfig, amount)
		if err != nil {
			logger.Error("Failed to process energy reward: %v", err)
			// Continue even if reward processing fails
		} else if r != nil {
			if reward == nil {
				reward = r
			} else {
				// Merge rewards
				e.mergeRewards(reward, r)
			}
		}
	}

	return energies, reward, nil
}

// Grant will add the amounts to each energy (while applying any energy modifiers) for a user by ID.
func (e *NakamaEnergySystem) Grant(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, amounts map[string]int32, modifiers []*RewardEnergyModifier) (map[string]*Energy, error) {
	if e.config == nil || len(e.config.Energies) == 0 {
		// No energies are configured
		return make(map[string]*Energy), nil
	}

	// Fetch current energy values
	energies, err := e.Get(ctx, logger, nk, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()

	// Create temporary map of modifiers to apply
	modifiersToApply := make(map[string][]*RewardEnergyModifier)

	// Apply any energy modifiers
	if len(modifiers) > 0 {
		for _, modifier := range modifiers {
			energy, exists := energies[modifier.Id]
			if !exists {
				// Skip modifiers for energies that don't exist
				continue
			}

			// Create and add a new EnergyModifier
			energyMod := &EnergyModifier{
				Operator:     modifier.Operator,
				Value:        int32(modifier.Value),
				StartTimeSec: now,
				EndTimeSec:   now + int64(modifier.DurationSec),
			}

			if energy.Modifiers == nil {
				energy.Modifiers = []*EnergyModifier{energyMod}
			} else {
				energy.Modifiers = append(energy.Modifiers, energyMod)
			}

			// Add to our temporary map for instant application
			if _, ok := modifiersToApply[modifier.Id]; !ok {
				modifiersToApply[modifier.Id] = make([]*RewardEnergyModifier, 0)
			}
			modifiersToApply[modifier.Id] = append(modifiersToApply[modifier.Id], modifier)
		}
	}

	// Apply the grant amounts
	for id, amount := range amounts {
		energyConfig, exists := e.config.Energies[id]
		if !exists {
			logger.Warn("Attempted to grant non-existent energy: %s", id)
			continue
		}

		energy, exists := energies[id]
		if !exists {
			// Create a new energy entry with default values
			energy = &Energy{
				Id:                   id,
				Current:              energyConfig.StartCount,
				Max:                  energyConfig.MaxCount,
				Refill:               energyConfig.RefillCount,
				RefillSec:            energyConfig.RefillTimeSec,
				NextRefillTimeSec:    0,
				MaxRefillTimeSec:     0,
				StartRefillTimeSec:   now,
				CurrentTimeSec:       now,
				AdditionalProperties: energyConfig.AdditionalProperties,
			}
			energies[id] = energy
		}

		// Apply modifiers to the amount
		modifiedAmount := amount
		if mods, ok := modifiersToApply[id]; ok {
			for _, mod := range mods {
				modifiedAmount = applyModifier(modifiedAmount, mod.Operator, int32(mod.Value))
			}
		}

		// Add the energy, cap at max + overfill limit
		maxWithOverfill := energy.Max
		if energyConfig.MaxOverfill > 0 {
			maxWithOverfill += energyConfig.MaxOverfill
		}

		energy.Current += modifiedAmount
		if energy.Current > maxWithOverfill {
			energy.Current = maxWithOverfill
		}

		// Update refill times
		if energy.Current >= energy.Max {
			// If we're at max, no refills needed
			energy.NextRefillTimeSec = 0
			energy.MaxRefillTimeSec = 0
		} else if energy.RefillSec > 0 {
			// Calculate next refill time
			energy.StartRefillTimeSec = now
			e.calculateRefillTimes(energy, now)
		}
	}

	// Save the updated energy values
	if err := e.saveUserEnergies(ctx, logger, nk, userID, energies); err != nil {
		logger.Error("Failed to save user energies: %v", err)
		return nil, ErrInternal
	}

	return energies, nil
}

// SetOnSpendReward sets a custom reward function which will run after an energy reward's value has been rolled.
func (e *NakamaEnergySystem) SetOnSpendReward(fn OnReward[*EnergyConfigEnergy]) {
	e.onSpendReward = fn
}

// Helper Functions

// getUserEnergies fetches the stored energy data for a user from Nakama storage.
func (e *NakamaEnergySystem) getUserEnergies(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (map[string]*Energy, error) {
	collection := "energy"
	key := "user_energies"

	// Read from storage
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: collection,
			Key:        key,
			UserID:     userID,
		},
	})

	if err != nil {
		logger.Error("Failed to read user energies: %v", err)
		return nil, err
	}

	energies := make(map[string]*Energy)

	// If no data found, return empty map
	if len(objects) == 0 {
		return energies, nil
	}

	// Unmarshal the stored energy data
	energyList := &EnergyList{}
	if err := json.Unmarshal([]byte(objects[0].Value), energyList); err != nil {
		logger.Error("Failed to unmarshal user energies: %v", err)
		return nil, err
	}

	if energyList.Energies != nil {
		return energyList.Energies, nil
	}

	return energies, nil
}

// saveUserEnergies stores the updated energy data for a user in Nakama storage.
func (e *NakamaEnergySystem) saveUserEnergies(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, energies map[string]*Energy) error {
	collection := "energy"
	key := "user_energies"

	// Marshal the energy data
	energyList := &EnergyList{
		Energies: energies,
	}

	data, err := json.Marshal(energyList)
	if err != nil {
		logger.Error("Failed to marshal user energies: %v", err)
		return err
	}

	// Write to storage
	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{
		{
			Collection:      collection,
			Key:             key,
			UserID:          userID,
			Value:           string(data),
			PermissionRead:  runtime.STORAGE_PERMISSION_OWNER_READ,
			PermissionWrite: runtime.STORAGE_PERMISSION_OWNER_WRITE,
		},
	})

	if err != nil {
		logger.Error("Failed to write user energies: %v", err)
		return err
	}

	return nil
}

// applyRefills calculates and applies any energy refills that should have occurred since the last check.
func (e *NakamaEnergySystem) applyRefills(energy *Energy, now int64) {
	// If already at max, no refills needed
	if energy.Current >= energy.Max {
		return
	}

	// Check for infinite energy modifier, if active, set current to max
	if hasInfiniteEnergyModifier(energy) {
		energy.Current = energy.Max
		return
	}

	// If no refill rate, can't refill
	if energy.RefillSec <= 0 || energy.Refill <= 0 {
		return
	}

	// Calculate elapsed time since last refill start
	elapsedSec := now - energy.StartRefillTimeSec
	if elapsedSec <= 0 {
		return
	}

	// Calculate how many refills have occurred
	refillCount := elapsedSec / energy.RefillSec
	if refillCount <= 0 {
		return
	}

	// Apply the refills
	refillAmount := refillCount * int64(energy.Refill)
	energy.Current += int32(refillAmount)

	// Cap at max
	if energy.Current > energy.Max {
		energy.Current = energy.Max
	}

	// Update start time to reflect refills that have been applied
	energy.StartRefillTimeSec += refillCount * energy.RefillSec
}

// calculateRefillTimes updates the next refill time and max refill time for an energy.
func (e *NakamaEnergySystem) calculateRefillTimes(energy *Energy, now int64) {
	// If already at max, no refills needed
	if energy.Current >= energy.Max {
		energy.NextRefillTimeSec = 0
		energy.MaxRefillTimeSec = 0
		return
	}

	// Check for infinite energy modifier
	if hasInfiniteEnergyModifier(energy) {
		energy.NextRefillTimeSec = 0
		energy.MaxRefillTimeSec = 0
		return
	}

	// If no refill rate, can't calculate refill times
	if energy.RefillSec <= 0 || energy.Refill <= 0 {
		energy.NextRefillTimeSec = 0
		energy.MaxRefillTimeSec = 0
		return
	}

	// Calculate time since last refill
	timeSinceLastRefill := now - energy.StartRefillTimeSec

	// Calculate time until next refill
	timeUntilNextRefill := energy.RefillSec - (timeSinceLastRefill % energy.RefillSec)
	if timeUntilNextRefill == energy.RefillSec {
		timeUntilNextRefill = 0
	}

	// Set next refill time
	if timeUntilNextRefill > 0 {
		energy.NextRefillTimeSec = now + timeUntilNextRefill
	} else {
		energy.NextRefillTimeSec = now
	}

	// Calculate max refill time (time until fully refilled)
	refillsNeeded := int64(0)
	energyNeeded := energy.Max - energy.Current
	if energyNeeded > 0 {
		refillsNeeded = int64((energyNeeded + energy.Refill - 1) / energy.Refill) // Ceiling division
	}

	if refillsNeeded <= 0 {
		energy.MaxRefillTimeSec = energy.NextRefillTimeSec
	} else {
		energy.MaxRefillTimeSec = energy.NextRefillTimeSec + energy.RefillSec*(refillsNeeded-1)
	}
}

// hasInfiniteEnergyModifier checks if the energy has an active infinite energy modifier
func hasInfiniteEnergyModifier(energy *Energy) bool {
	if energy.Modifiers == nil {
		return false
	}

	now := time.Now().Unix()
	for _, mod := range energy.Modifiers {
		if mod.Operator == "infinite_energy" && mod.Value > 0 &&
			(mod.EndTimeSec == 0 || mod.EndTimeSec > now) {
			return true
		}
	}
	return false
}

// processEnergyReward handles reward generation when energy is spent.
func (e *NakamaEnergySystem) processEnergyReward(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, energyID string, energyConfig *EnergyConfigEnergy, amount int32) (*Reward, error) {
	if energyConfig.Reward == nil {
		return nil, nil
	}

	// Create a base reward
	reward := &Reward{
		Items:           make(map[string]int64),
		Currencies:      make(map[string]int64),
		Energies:        make(map[string]int32),
		EnergyModifiers: make([]*RewardEnergyModifier, 0),
		RewardModifiers: make([]*RewardModifier, 0),
		GrantTimeSec:    time.Now().Unix(),
		ItemInstances:   make(map[string]*RewardInventoryItem),
	}

	// Get the economy system from the Pamlogix instance
	var economySystem EconomySystem

	if e.pamlogix != nil {
		economySystem = e.pamlogix.GetEconomySystem()
	}

	// If we couldn't get the economy system from Pamlogix, create a minimal temporary one
	if economySystem == nil {
		logger.Warn("No economy system available through Pamlogix, creating a minimal temporary one")
		economySystem = NewNakamaEconomySystem(nil)
	}

	// Roll the reward using the reward configuration
	rolledReward, err := economySystem.RewardRoll(ctx, logger, nk, userID, energyConfig.Reward)
	if err != nil {
		logger.Error("Failed to roll reward for energy spend: %v", err)
		// Continue with empty reward
	} else if rolledReward != nil {
		// Use the rolled reward
		reward = rolledReward
	}

	// If a custom reward function is set, let it modify the reward
	if e.onSpendReward != nil {
		customReward, err := e.onSpendReward(ctx, logger, nk, userID, energyID, energyConfig, energyConfig.Reward, reward)
		if err != nil {
			logger.Error("Custom reward function failed: %v", err)
			return reward, err
		}
		if customReward != nil {
			reward = customReward
		}
	}

	return reward, nil
}

// mergeRewards combines two rewards into one.
func (e *NakamaEnergySystem) mergeRewards(target, source *Reward) {
	if source == nil {
		return
	}

	// Merge items
	for id, count := range source.Items {
		target.Items[id] += count
	}

	// Merge currencies
	for id, count := range source.Currencies {
		target.Currencies[id] += count
	}

	// Merge energies
	for id, count := range source.Energies {
		target.Energies[id] += count
	}

	// Append energy modifiers
	target.EnergyModifiers = append(target.EnergyModifiers, source.EnergyModifiers...)

	// Append reward modifiers
	target.RewardModifiers = append(target.RewardModifiers, source.RewardModifiers...)

	// Merge item instances
	for id, item := range source.ItemInstances {
		target.ItemInstances[id] = item
	}
}

// applyModifier applies an energy modifier to a value.
func applyModifier(value int32, operator string, modValue int32) int32 {
	switch operator {
	case "add":
		return value + modValue
	case "subtract":
		return value - modValue
	case "multiply":
		return value * modValue
	case "divide":
		if modValue != 0 {
			return value / modValue
		}
		return value
	case "set":
		return modValue
	case "min":
		if value < modValue {
			return value
		}
		return modValue
	case "max":
		if value > modValue {
			return value
		}
		return modValue
	case "mod":
		if modValue != 0 {
			return value % modValue
		}
		return value
	case "pow":
		// Simple power implementation for small exponents
		if modValue <= 0 {
			return 1
		}
		result := value
		for i := int32(1); i < modValue; i++ {
			result *= value
		}
		return result
	// These operators are handled specially in applyActiveModifiers
	case "max_energy", "refill_rate", "refill_speed", "infinite_energy":
		return value // Just pass through in the general case
	default:
		return value
	}
}

// applyActiveModifiers updates energy parameters based on active modifiers
func (e *NakamaEnergySystem) applyActiveModifiers(energy *Energy, now int64) {
	if energy.Modifiers == nil || len(energy.Modifiers) == 0 {
		return
	}

	// Get the original config values to use as base
	energyConfig, exists := e.config.Energies[energy.Id]
	if !exists {
		return
	}

	// Reset to base values first
	baseMax := energyConfig.MaxCount
	baseRefill := energyConfig.RefillCount
	baseRefillSec := energyConfig.RefillTimeSec

	// Keep track of active modifiers, and remove expired ones
	activeModifiers := make([]*EnergyModifier, 0, len(energy.Modifiers))

	for _, mod := range energy.Modifiers {
		// Skip expired modifiers
		if mod.EndTimeSec > 0 && mod.EndTimeSec < now {
			continue
		}

		// Add to active modifiers list
		activeModifiers = append(activeModifiers, mod)

		// Apply modifier based on operator
		switch mod.Operator {
		case "max_energy":
			// Modifier for maximum energy capacity
			baseMax = applyModifier(baseMax, "add", mod.Value) // Add to max energy
		case "refill_rate":
			// Modifier for refill rate
			baseRefill = applyModifier(baseRefill, "multiply", mod.Value) // Multiply refill amount
		case "refill_speed":
			// Modifier for refill speed (lower is faster)
			if mod.Value > 0 {
				baseRefillSec = int64(applyModifier(int32(baseRefillSec), "divide", mod.Value))
			}
		case "infinite_energy":
			// Special case for infinite energy
			if mod.Value > 0 {
				baseMax = math.MaxInt32
				energy.Current = baseMax
			}
		}
	}

	// Update energy with modified values
	energy.Max = baseMax
	energy.Refill = baseRefill
	energy.RefillSec = baseRefillSec

	// Replace modifiers list with only active ones
	energy.Modifiers = activeModifiers
}

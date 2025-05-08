package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/heroiclabs/nakama-common/api"
	"github.com/heroiclabs/nakama-common/runtime"
)

// NakamaEconomySystem implements the EconomySystem interface using Nakama as the backend.
type NakamaEconomySystem struct {
	config *EconomyConfig
}

func NewNakamaEconomySystem(config *EconomyConfig) *NakamaEconomySystem {
	return &NakamaEconomySystem{config: config}
}

func (e *NakamaEconomySystem) GetType() SystemType {
	return SystemTypeEconomy
}

func (e *NakamaEconomySystem) GetConfig() any {
	return e.config
}

func (e *NakamaEconomySystem) RewardCreate() (rewardConfig *EconomyConfigReward) {
	// Returns a new, empty reward config for further customization.
	return &EconomyConfigReward{}
}

func (e *NakamaEconomySystem) RewardConvert(contents *AvailableRewards) (rewardConfig *EconomyConfigReward) {
	if contents == nil {
		return nil
	}

	rewardConfig = &EconomyConfigReward{
		MaxRolls:       contents.GetMaxRolls(),
		MaxRepeatRolls: contents.GetMaxRepeatRolls(),
		TotalWeight:    contents.GetTotalWeight(),
	}

	// Process guaranteed rewards if present
	if guaranteed := contents.GetGuaranteed(); guaranteed != nil {
		rewardConfig.Guaranteed = &EconomyConfigRewardContents{
			Items:           make(map[string]*EconomyConfigRewardItem),
			Currencies:      make(map[string]*EconomyConfigRewardCurrency),
			Energies:        make(map[string]*EconomyConfigRewardEnergy),
			EnergyModifiers: make([]*EconomyConfigRewardEnergyModifier, 0),
			RewardModifiers: make([]*EconomyConfigRewardRewardModifier, 0),
		}

		// Convert currencies
		for k, v := range guaranteed.GetCurrencies() {
			if v.GetCount() != nil {
				rewardConfig.Guaranteed.Currencies[k] = &EconomyConfigRewardCurrency{
					EconomyConfigRewardRangeInt64: EconomyConfigRewardRangeInt64{
						Min:      v.GetCount().GetMin(),
						Max:      v.GetCount().GetMax(),
						Multiple: v.GetCount().GetMultiple(),
					},
				}
			}
		}

		// Convert items
		for k, v := range guaranteed.GetItems() {
			if v.GetCount() != nil {
				item := &EconomyConfigRewardItem{
					EconomyConfigRewardRangeInt64: EconomyConfigRewardRangeInt64{
						Min:      v.GetCount().GetMin(),
						Max:      v.GetCount().GetMax(),
						Multiple: v.GetCount().GetMultiple(),
					},
				}

				// Convert string properties if present
				if len(v.GetStringProperties()) > 0 {
					item.StringProperties = make(map[string]*EconomyConfigRewardStringProperty)
					for propKey, propVal := range v.GetStringProperties() {
						stringProp := &EconomyConfigRewardStringProperty{
							TotalWeight: propVal.GetTotalWeight(),
							Options:     make(map[string]*EconomyConfigRewardStringPropertyOption),
						}

						for optKey, optVal := range propVal.GetOptions() {
							stringProp.Options[optKey] = &EconomyConfigRewardStringPropertyOption{
								Weight: optVal.GetWeight(),
							}
						}
						item.StringProperties[propKey] = stringProp
					}
				}

				// Convert numeric properties if present
				if len(v.GetNumericProperties()) > 0 {
					item.NumericProperties = make(map[string]*EconomyConfigRewardRangeFloat64)
					for propKey, propVal := range v.GetNumericProperties() {
						item.NumericProperties[propKey] = &EconomyConfigRewardRangeFloat64{
							Min:      propVal.GetMin(),
							Max:      propVal.GetMax(),
							Multiple: propVal.GetMultiple(),
						}
					}
				}

				rewardConfig.Guaranteed.Items[k] = item
			}
		}

		// Convert energies
		for k, v := range guaranteed.GetEnergies() {
			if v.GetCount() != nil {
				rewardConfig.Guaranteed.Energies[k] = &EconomyConfigRewardEnergy{
					EconomyConfigRewardRangeInt32: EconomyConfigRewardRangeInt32{
						Min:      v.GetCount().GetMin(),
						Max:      v.GetCount().GetMax(),
						Multiple: v.GetCount().GetMultiple(),
					},
				}
			}
		}

		// Convert item sets
		if len(guaranteed.GetItemSets()) > 0 {
			rewardConfig.Guaranteed.ItemSets = make([]*EconomyConfigRewardItemSet, len(guaranteed.GetItemSets()))
			for i, itemSet := range guaranteed.GetItemSets() {
				var min, max, multiple int64
				if itemSet.GetCount() != nil {
					min = itemSet.GetCount().GetMin()
					max = itemSet.GetCount().GetMax()
					multiple = itemSet.GetCount().GetMultiple()
				}
				configItemSet := &EconomyConfigRewardItemSet{
					EconomyConfigRewardRangeInt64: EconomyConfigRewardRangeInt64{
						Min:      min,
						Max:      max,
						Multiple: multiple,
					},
					MaxRepeats: itemSet.GetMaxRepeats(),
					Set:        itemSet.GetSet(),
				}
				rewardConfig.Guaranteed.ItemSets[i] = configItemSet
			}
		}

		// Convert energy modifiers
		if len(guaranteed.GetEnergyModifiers()) > 0 {
			rewardConfig.Guaranteed.EnergyModifiers = make([]*EconomyConfigRewardEnergyModifier, len(guaranteed.GetEnergyModifiers()))
			for i, modifier := range guaranteed.GetEnergyModifiers() {
				configModifier := &EconomyConfigRewardEnergyModifier{
					Id:       modifier.GetId(),
					Operator: modifier.GetOperator(),
				}

				if modifier.GetValue() != nil {
					configModifier.Value = &EconomyConfigRewardRangeInt64{
						Min:      modifier.GetValue().GetMin(),
						Max:      modifier.GetValue().GetMax(),
						Multiple: modifier.GetValue().GetMultiple(),
					}
				}

				if modifier.GetDurationSec() != nil {
					configModifier.DurationSec = &EconomyConfigRewardRangeUInt64{
						Min:      modifier.GetDurationSec().GetMin(),
						Max:      modifier.GetDurationSec().GetMax(),
						Multiple: modifier.GetDurationSec().GetMultiple(),
					}
				}

				rewardConfig.Guaranteed.EnergyModifiers[i] = configModifier
			}
		}

		// Convert reward modifiers
		if len(guaranteed.GetRewardModifiers()) > 0 {
			rewardConfig.Guaranteed.RewardModifiers = make([]*EconomyConfigRewardRewardModifier, len(guaranteed.GetRewardModifiers()))
			for i, modifier := range guaranteed.GetRewardModifiers() {
				configModifier := &EconomyConfigRewardRewardModifier{
					Id:       modifier.GetId(),
					Type:     modifier.GetType(),
					Operator: modifier.GetOperator(),
				}

				if modifier.GetValue() != nil {
					configModifier.Value = &EconomyConfigRewardRangeInt64{
						Min:      modifier.GetValue().GetMin(),
						Max:      modifier.GetValue().GetMax(),
						Multiple: modifier.GetValue().GetMultiple(),
					}
				}

				if modifier.GetDurationSec() != nil {
					configModifier.DurationSec = &EconomyConfigRewardRangeUInt64{
						Min:      modifier.GetDurationSec().GetMin(),
						Max:      modifier.GetDurationSec().GetMax(),
						Multiple: modifier.GetDurationSec().GetMultiple(),
					}
				}

				rewardConfig.Guaranteed.RewardModifiers[i] = configModifier
			}
		}
	}

	// Process weighted rewards if present
	if weightedRewards := contents.GetWeighted(); len(weightedRewards) > 0 {
		rewardConfig.Weighted = make([]*EconomyConfigRewardContents, len(weightedRewards))

		for i, weighted := range weightedRewards {
			weightedContent := &EconomyConfigRewardContents{
				Weight:          weighted.GetWeight(),
				Items:           make(map[string]*EconomyConfigRewardItem),
				Currencies:      make(map[string]*EconomyConfigRewardCurrency),
				Energies:        make(map[string]*EconomyConfigRewardEnergy),
				EnergyModifiers: make([]*EconomyConfigRewardEnergyModifier, 0),
				RewardModifiers: make([]*EconomyConfigRewardRewardModifier, 0),
			}

			// Convert currencies
			for k, v := range weighted.GetCurrencies() {
				if v.GetCount() != nil {
					weightedContent.Currencies[k] = &EconomyConfigRewardCurrency{
						EconomyConfigRewardRangeInt64: EconomyConfigRewardRangeInt64{
							Min:      v.GetCount().GetMin(),
							Max:      v.GetCount().GetMax(),
							Multiple: v.GetCount().GetMultiple(),
						},
					}
				}
			}

			// Convert items
			for k, v := range weighted.GetItems() {
				if v.GetCount() != nil {
					item := &EconomyConfigRewardItem{
						EconomyConfigRewardRangeInt64: EconomyConfigRewardRangeInt64{
							Min:      v.GetCount().GetMin(),
							Max:      v.GetCount().GetMax(),
							Multiple: v.GetCount().GetMultiple(),
						},
					}

					// Convert string properties if present
					if len(v.GetStringProperties()) > 0 {
						item.StringProperties = make(map[string]*EconomyConfigRewardStringProperty)
						for propKey, propVal := range v.GetStringProperties() {
							stringProp := &EconomyConfigRewardStringProperty{
								TotalWeight: propVal.GetTotalWeight(),
								Options:     make(map[string]*EconomyConfigRewardStringPropertyOption),
							}

							for optKey, optVal := range propVal.GetOptions() {
								stringProp.Options[optKey] = &EconomyConfigRewardStringPropertyOption{
									Weight: optVal.GetWeight(),
								}
							}
							item.StringProperties[propKey] = stringProp
						}
					}

					// Convert numeric properties if present
					if len(v.GetNumericProperties()) > 0 {
						item.NumericProperties = make(map[string]*EconomyConfigRewardRangeFloat64)
						for propKey, propVal := range v.GetNumericProperties() {
							item.NumericProperties[propKey] = &EconomyConfigRewardRangeFloat64{
								Min:      propVal.GetMin(),
								Max:      propVal.GetMax(),
								Multiple: propVal.GetMultiple(),
							}
						}
					}

					weightedContent.Items[k] = item
				}
			}

			// Convert energies
			for k, v := range weighted.GetEnergies() {
				if v.GetCount() != nil {
					weightedContent.Energies[k] = &EconomyConfigRewardEnergy{
						EconomyConfigRewardRangeInt32: EconomyConfigRewardRangeInt32{
							Min:      v.GetCount().GetMin(),
							Max:      v.GetCount().GetMax(),
							Multiple: v.GetCount().GetMultiple(),
						},
					}
				}
			}

			// Convert item sets
			if len(weighted.GetItemSets()) > 0 {
				weightedContent.ItemSets = make([]*EconomyConfigRewardItemSet, len(weighted.GetItemSets()))
				for j, itemSet := range weighted.GetItemSets() {
					var min, max, multiple int64
					if itemSet.GetCount() != nil {
						min = itemSet.GetCount().GetMin()
						max = itemSet.GetCount().GetMax()
						multiple = itemSet.GetCount().GetMultiple()
					}
					configItemSet := &EconomyConfigRewardItemSet{
						EconomyConfigRewardRangeInt64: EconomyConfigRewardRangeInt64{
							Min:      min,
							Max:      max,
							Multiple: multiple,
						},
						MaxRepeats: itemSet.GetMaxRepeats(),
						Set:        itemSet.GetSet(),
					}
					weightedContent.ItemSets[j] = configItemSet
				}
			}

			// Convert energy modifiers
			if len(weighted.GetEnergyModifiers()) > 0 {
				weightedContent.EnergyModifiers = make([]*EconomyConfigRewardEnergyModifier, len(weighted.GetEnergyModifiers()))
				for j, modifier := range weighted.GetEnergyModifiers() {
					configModifier := &EconomyConfigRewardEnergyModifier{
						Id:       modifier.GetId(),
						Operator: modifier.GetOperator(),
					}

					if modifier.GetValue() != nil {
						configModifier.Value = &EconomyConfigRewardRangeInt64{
							Min:      modifier.GetValue().GetMin(),
							Max:      modifier.GetValue().GetMax(),
							Multiple: modifier.GetValue().GetMultiple(),
						}
					}

					if modifier.GetDurationSec() != nil {
						configModifier.DurationSec = &EconomyConfigRewardRangeUInt64{
							Min:      modifier.GetDurationSec().GetMin(),
							Max:      modifier.GetDurationSec().GetMax(),
							Multiple: modifier.GetDurationSec().GetMultiple(),
						}
					}

					weightedContent.EnergyModifiers[j] = configModifier
				}
			}

			// Convert reward modifiers
			if len(weighted.GetRewardModifiers()) > 0 {
				weightedContent.RewardModifiers = make([]*EconomyConfigRewardRewardModifier, len(weighted.GetRewardModifiers()))
				for j, modifier := range weighted.GetRewardModifiers() {
					configModifier := &EconomyConfigRewardRewardModifier{
						Id:       modifier.GetId(),
						Type:     modifier.GetType(),
						Operator: modifier.GetOperator(),
					}

					if modifier.GetValue() != nil {
						configModifier.Value = &EconomyConfigRewardRangeInt64{
							Min:      modifier.GetValue().GetMin(),
							Max:      modifier.GetValue().GetMax(),
							Multiple: modifier.GetValue().GetMultiple(),
						}
					}

					if modifier.GetDurationSec() != nil {
						configModifier.DurationSec = &EconomyConfigRewardRangeUInt64{
							Min:      modifier.GetDurationSec().GetMin(),
							Max:      modifier.GetDurationSec().GetMax(),
							Multiple: modifier.GetDurationSec().GetMultiple(),
						}
					}

					weightedContent.RewardModifiers[j] = configModifier
				}
			}

			rewardConfig.Weighted[i] = weightedContent
		}
	}

	return rewardConfig
}

func (e *NakamaEconomySystem) RewardRoll(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, rewardConfig *EconomyConfigReward) (reward *Reward, err error) {
	if rewardConfig == nil {
		return nil, runtime.NewError("reward config is nil", 3) // INVALID_ARGUMENT
	}

	reward = &Reward{
		Items:           make(map[string]int64),
		Currencies:      make(map[string]int64),
		Energies:        make(map[string]int32),
		EnergyModifiers: make([]*RewardEnergyModifier, 0),
		RewardModifiers: make([]*RewardModifier, 0),
	}

	// Process guaranteed rewards first
	if rewardConfig.Guaranteed != nil {
		if err = e.processRewardContents(reward, rewardConfig.Guaranteed); err != nil {
			return nil, err
		}
	}

	// Process weighted rewards if available
	if len(rewardConfig.Weighted) > 0 && rewardConfig.MaxRolls > 0 {
		// Calculate total weight if not provided
		totalWeight := rewardConfig.TotalWeight
		if totalWeight == 0 {
			for _, contentGroup := range rewardConfig.Weighted {
				totalWeight += contentGroup.Weight
			}
		}

		if totalWeight <= 0 {
			logger.Warn("Total weight for weighted rewards is zero or negative, skipping weighted rolls")
		} else {
			// Track which indices have been rolled to avoid repeats if configured
			rolledIndices := make(map[int]bool)

			// Perform rolls
			for roll := int64(0); roll < rewardConfig.MaxRolls; roll++ {
				// If we've already used all available options with repeat protection
				if len(rolledIndices) >= len(rewardConfig.Weighted) && rewardConfig.MaxRepeatRolls == 0 {
					break
				}

				// Choose a random reward group
				randVal := e.randomInt64(0, totalWeight)
				var cumulativeWeight int64 = 0
				var selectedIndex int = -1

				for i, contentGroup := range rewardConfig.Weighted {
					cumulativeWeight += contentGroup.Weight
					if randVal < cumulativeWeight {
						selectedIndex = i
						break
					}
				}

				if selectedIndex == -1 {
					logger.Warn("Failed to select a weighted reward, possibly due to weight calculation error")
					continue
				}

				// Handle repeat protection
				if rolledIndices[selectedIndex] {
					// Skip if we've already hit our repeat limit
					if rewardConfig.MaxRepeatRolls > 0 {
						repeatCount := e.getRepeatCount(rolledIndices, selectedIndex)
						if repeatCount >= rewardConfig.MaxRepeatRolls {
							// Try again with this roll
							roll--
							continue
						}
					}
				}

				// Mark this index as rolled
				rolledIndices[selectedIndex] = true

				// Process the selected reward group
				if err = e.processRewardContents(reward, rewardConfig.Weighted[selectedIndex]); err != nil {
					return nil, err
				}
			}
		}
	}

	return reward, nil
}

// Helper function to count repeats of a specific index
func (e *NakamaEconomySystem) getRepeatCount(rolledIndices map[int]bool, index int) int64 {
	count := int64(0)
	if rolledIndices[index] {
		count = 1
	}
	return count
}

// Helper function to process a single reward contents group
func (e *NakamaEconomySystem) processRewardContents(reward *Reward, contents *EconomyConfigRewardContents) error {
	// Process currencies
	for currencyID, currencyReward := range contents.Currencies {
		amount := e.rollRangeInt64(currencyReward.Min, currencyReward.Max, currencyReward.Multiple)
		reward.Currencies[currencyID] += amount
	}

	// Process items
	for itemID, itemReward := range contents.Items {
		amount := e.rollRangeInt64(itemReward.Min, itemReward.Max, itemReward.Multiple)
		reward.Items[itemID] += amount

		// Process item properties if needed
		if len(itemReward.StringProperties) > 0 || len(itemReward.NumericProperties) > 0 {
			if reward.ItemInstances == nil {
				reward.ItemInstances = make(map[string]*RewardInventoryItem)
			}

			if _, exists := reward.ItemInstances[itemID]; !exists {
				reward.ItemInstances[itemID] = &RewardInventoryItem{
					Id:                itemID,
					Count:             0,
					StringProperties:  make(map[string]string),
					NumericProperties: make(map[string]float64),
					InstanceId:        "",
				}
			}

			// Roll string properties
			for propKey, propConfig := range itemReward.StringProperties {
				if propConfig.TotalWeight > 0 && len(propConfig.Options) > 0 {
					// Select a random property value based on weights
					randValue := e.randomInt64(0, propConfig.TotalWeight)
					var cumulativeWeight int64 = 0
					var selectedValue string

					for value, option := range propConfig.Options {
						cumulativeWeight += option.Weight
						if randValue < cumulativeWeight {
							selectedValue = value
							break
						}
					}

					if selectedValue != "" {
						reward.ItemInstances[itemID].StringProperties[propKey] = selectedValue
					}
				}
			}

			// Roll numeric properties
			for propKey, propConfig := range itemReward.NumericProperties {
				value := e.rollRangeFloat64(propConfig.Min, propConfig.Max, propConfig.Multiple)
				reward.ItemInstances[itemID].NumericProperties[propKey] = value
			}
		}
	}

	// Process item sets
	for _, itemSet := range contents.ItemSets {
		if len(itemSet.Set) == 0 {
			continue
		}

		// Determine number of items to select from the set
		count := e.rollRangeInt64(itemSet.Min, itemSet.Max, itemSet.Multiple)
		if count <= 0 {
			continue
		}

		// Create a copy of the set to avoid modifying the original
		availableItems := make([]string, len(itemSet.Set))
		copy(availableItems, itemSet.Set)

		// Track items we've already selected to handle max repeats
		selectedItems := make(map[string]int64)

		// Select random items from the set
		for i := int64(0); i < count; i++ {
			if len(availableItems) == 0 {
				break
			}

			// Choose a random item
			index := e.randomInt64(0, int64(len(availableItems)))
			selectedItem := availableItems[index]

			// Increment the item count
			selectedItems[selectedItem]++
			reward.Items[selectedItem]++

			// Remove the item from available options if we've hit max repeats
			if itemSet.MaxRepeats > 0 && selectedItems[selectedItem] >= itemSet.MaxRepeats {
				// Remove this item from available items
				availableItems = append(availableItems[:index], availableItems[index+1:]...)
			}
		}
	}

	// Process energies
	for energyID, energyReward := range contents.Energies {
		amount := e.rollRangeInt32(energyReward.Min, energyReward.Max, energyReward.Multiple)
		reward.Energies[energyID] += amount
	}

	// Process energy modifiers
	for _, modifierConfig := range contents.EnergyModifiers {
		if modifierConfig.Value != nil {
			value := e.rollRangeInt64(modifierConfig.Value.Min, modifierConfig.Value.Max, modifierConfig.Value.Multiple)
			var duration uint64 = 0

			if modifierConfig.DurationSec != nil {
				duration = e.rollRangeUInt64(modifierConfig.DurationSec.Min, modifierConfig.DurationSec.Max, modifierConfig.DurationSec.Multiple)
			}

			modifier := &RewardEnergyModifier{
				Id:          modifierConfig.Id,
				Operator:    modifierConfig.Operator,
				Value:       value,
				DurationSec: duration,
			}

			reward.EnergyModifiers = append(reward.EnergyModifiers, modifier)
		}
	}

	// Process reward modifiers
	for _, modifierConfig := range contents.RewardModifiers {
		if modifierConfig.Value != nil {
			value := e.rollRangeInt64(modifierConfig.Value.Min, modifierConfig.Value.Max, modifierConfig.Value.Multiple)
			var duration uint64 = 0

			if modifierConfig.DurationSec != nil {
				duration = e.rollRangeUInt64(modifierConfig.DurationSec.Min, modifierConfig.DurationSec.Max, modifierConfig.DurationSec.Multiple)
			}

			modifier := &RewardModifier{
				Id:          modifierConfig.Id,
				Type:        modifierConfig.Type,
				Operator:    modifierConfig.Operator,
				Value:       value,
				DurationSec: duration,
			}

			reward.RewardModifiers = append(reward.RewardModifiers, modifier)
		}
	}

	return nil
}

// Helper methods for random number generation
func (e *NakamaEconomySystem) randomInt64(min, max int64) int64 {
	// For simplicity, using just the Nakama module's random function
	// In a real implementation, you would access the module via the struct
	return min + rand.Int63n(max-min+1)
}

func (e *NakamaEconomySystem) rollRangeInt64(min, max, multiple int64) int64 {
	if min == max {
		return min
	}

	if multiple <= 0 {
		multiple = 1
	}

	// Generate a random value between min and max
	value := e.randomInt64(min, max)

	// Adjust to be a multiple if needed
	if multiple > 1 {
		remainder := value % multiple
		if remainder != 0 {
			value = value - remainder
		}
	}

	return value
}

func (e *NakamaEconomySystem) rollRangeInt32(min, max, multiple int32) int32 {
	return int32(e.rollRangeInt64(int64(min), int64(max), int64(multiple)))
}

func (e *NakamaEconomySystem) rollRangeUInt64(min, max, multiple uint64) uint64 {
	return uint64(e.rollRangeInt64(int64(min), int64(max), int64(multiple)))
}

func (e *NakamaEconomySystem) rollRangeFloat64(min, max, multiple float64) float64 {
	if min == max {
		return min
	}

	if multiple <= 0 {
		multiple = 1.0
	}

	// Generate a random value between min and max
	value := min + rand.Float64()*(max-min)

	// Adjust to be a multiple if needed
	if multiple > 0 {
		value = math.Floor(value/multiple) * multiple
	}

	return value
}

func (e *NakamaEconomySystem) RewardGrant(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, reward *Reward, metadata map[string]interface{}, ignoreLimits bool) (newItems map[string]*InventoryItem, updatedItems map[string]*InventoryItem, notGrantedItemIDs map[string]int64, err error) {
	if reward == nil {
		return nil, nil, nil, runtime.NewError("reward is nil", 3) // INVALID_ARGUMENT
	}

	if userID == "" {
		return nil, nil, nil, runtime.NewError("user ID is empty", 3) // INVALID_ARGUMENT
	}

	// Initialize return values
	newItems = make(map[string]*InventoryItem)
	updatedItems = make(map[string]*InventoryItem)
	notGrantedItemIDs = make(map[string]int64)

	// Transaction to ensure atomicity
	//err = nk.WalletUpdate(ctx, userID, reward.Currencies, metadata, false)
	_, _, err = nk.WalletUpdate(ctx, userID, reward.Currencies, metadata, false)

	if err != nil {
		logger.Error("Failed to update wallet: %v", err)
		return nil, nil, nil, runtime.NewError("Failed to update wallet", 13) // INTERNAL
	}

	// Process inventory items
	if len(reward.Items) > 0 {
		// Get current inventory to check for item updates vs new items
		inventory, err := e.getInventory(ctx, nk, userID)
		if err != nil {
			logger.Error("Failed to retrieve inventory: %v", err)
			return nil, nil, nil, runtime.NewError("Failed to retrieve inventory", 13) // INTERNAL
		}

		// Prepare item operations
		itemsToAdd := make([]*runtime.StorageWrite, 0)

		for itemID, count := range reward.Items {
			if count <= 0 {
				continue
			}

			// Create string key for the item in the storage engine
			itemKey := fmt.Sprintf("inventory:%s", itemID)

			// Check if item already exists
			existingItem, exists := inventory.Items[itemID]

			var itemInstance *RewardInventoryItem
			// If we have instance data for this item
			if reward.ItemInstances != nil && reward.ItemInstances[itemID] != nil {
				itemInstance = reward.ItemInstances[itemID]
			}

			if exists {
				// Update existing item
				newCount := existingItem.Count + count
				existingItem.Count = newCount

				// Update properties if they exist in the reward
				if itemInstance != nil {
					// Merge string properties
					for propKey, propValue := range itemInstance.StringProperties {
						if existingItem.StringProperties == nil {
							existingItem.StringProperties = make(map[string]string)
						}
						existingItem.StringProperties[propKey] = propValue
					}

					// Merge numeric properties
					for propKey, propValue := range itemInstance.NumericProperties {
						if existingItem.NumericProperties == nil {
							existingItem.NumericProperties = make(map[string]float64)
						}
						existingItem.NumericProperties[propKey] = propValue
					}
				}

				// Prepare item for storage update
				itemData, err := json.Marshal(existingItem)
				if err != nil {
					logger.Error("Failed to marshal item data: %v", err)
					continue
				}

				// Add to storage operations
				itemsToAdd = append(itemsToAdd, &runtime.StorageWrite{
					Collection:      "inventory",
					Key:             itemKey,
					UserID:          userID,
					Value:           string(itemData),
					Version:         "", // No Version field in InventoryItem, use empty string
					PermissionRead:  1,  // Only owner can read
					PermissionWrite: 1,  // Only owner can write
				})

				updatedItems[itemID] = existingItem
			} else {
				// Create new item
				newItem := &InventoryItem{
					Id:    itemID,
					Count: count,
				}

				// Set instance properties if available
				if itemInstance != nil {
					newItem.StringProperties = itemInstance.StringProperties
					newItem.NumericProperties = itemInstance.NumericProperties
					if itemInstance.InstanceId != "" {
						newItem.InstanceId = itemInstance.InstanceId
					} else {
						// Generate a new instance ID if none was provided
						newItem.InstanceId = uuid.New().String()
					}
				}

				// Prepare item for storage
				itemData, err := json.Marshal(newItem)
				if err != nil {
					logger.Error("Failed to marshal new item data: %v", err)
					continue
				}

				// Add to storage operations
				itemsToAdd = append(itemsToAdd, &runtime.StorageWrite{
					Collection:      "inventory",
					Key:             itemKey,
					UserID:          userID,
					Value:           string(itemData),
					Version:         "", // New item, so version is empty
					PermissionRead:  1,  // Only owner can read
					PermissionWrite: 1,  // Only owner can write
				})

				newItems[itemID] = newItem
			}
		}

		// Execute storage operations if we have any
		if len(itemsToAdd) > 0 {
			_, err = nk.StorageWrite(ctx, itemsToAdd)
			if err != nil {
				logger.Error("Failed to write inventory updates: %v", err)
				return nil, nil, nil, runtime.NewError("Failed to update inventory", 13) // INTERNAL
			}
		}
	}

	// Process energy updates
	if len(reward.Energies) > 0 {
		err = e.updateEnergies(ctx, nk, userID, reward.Energies)
		if err != nil {
			logger.Error("Failed to update energies: %v", err)
			// Continue execution, don't fail the entire operation
		}
	}

	// Process energy modifiers
	if len(reward.EnergyModifiers) > 0 {
		err = e.applyEnergyModifiers(ctx, nk, userID, reward.EnergyModifiers)
		if err != nil {
			logger.Error("Failed to apply energy modifiers: %v", err)
			// Continue execution, don't fail the entire operation
		}
	}

	// Process reward modifiers
	if len(reward.RewardModifiers) > 0 {
		err = e.applyRewardModifiers(ctx, nk, userID, reward.RewardModifiers)
		if err != nil {
			logger.Error("Failed to apply reward modifiers: %v", err)
			// Continue execution, don't fail the entire operation
		}
	}

	return newItems, updatedItems, notGrantedItemIDs, nil
}

// Helper function to retrieve a user's inventory
func (e *NakamaEconomySystem) getInventory(ctx context.Context, nk runtime.NakamaModule, userID string) (*Inventory, error) {
	// Query the storage for all inventory items
	storageObjects, _, err := nk.StorageList(ctx, "", userID, "inventory", 100, "")
	if err != nil {
		return nil, err
	}

	inventory := &Inventory{
		Items: make(map[string]*InventoryItem),
	}

	// Parse each storage object into inventory items
	for _, obj := range storageObjects {
		if !strings.HasPrefix(obj.Key, "inventory:") {
			continue
		}

		// Extract item ID from key
		itemID := strings.TrimPrefix(obj.Key, "inventory:")

		// Parse the item data
		var item InventoryItem
		err = json.Unmarshal([]byte(obj.Value), &item)
		if err != nil {
			continue // Skip invalid items
		}

		// Add to inventory (no Version field in InventoryItem)
		inventory.Items[itemID] = &item
	}

	return inventory, nil
}

// Helper function to update energy values
func (e *NakamaEconomySystem) updateEnergies(ctx context.Context, nk runtime.NakamaModule, userID string, energies map[string]int32) error {
	// Get current energy values
	energyStorageObj, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: "energies",
			Key:        "user_energies",
			UserID:     userID,
		},
	})

	var userEnergies map[string]int32
	if err != nil || len(energyStorageObj) == 0 {
		// No existing energies, create new map
		userEnergies = make(map[string]int32)
	} else {
		// Parse existing energies
		err = json.Unmarshal([]byte(energyStorageObj[0].Value), &userEnergies)
		if err != nil {
			return err
		}
	}

	// Update energy values
	for energyID, amount := range energies {
		userEnergies[energyID] += amount
	}

	// Store updated energies
	energyData, err := json.Marshal(userEnergies)
	if err != nil {
		return err
	}

	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{
		{
			Collection:      "energies",
			Key:             "user_energies",
			UserID:          userID,
			Value:           string(energyData),
			Version:         energyStorageObj[0].Version,
			PermissionRead:  1,
			PermissionWrite: 1,
		},
	})

	return err
}

// Helper function to apply energy modifiers
func (e *NakamaEconomySystem) applyEnergyModifiers(ctx context.Context, nk runtime.NakamaModule, userID string, modifiers []*RewardEnergyModifier) error {
	// Get current modifiers
	modifiersObj, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: "modifiers",
			Key:        "energy_modifiers",
			UserID:     userID,
		},
	})

	var activeModifiers []*ActiveRewardModifier
	var version string

	if err != nil || len(modifiersObj) == 0 {
		// No existing modifiers
		activeModifiers = make([]*ActiveRewardModifier, 0)
	} else {
		// Parse existing modifiers
		err = json.Unmarshal([]byte(modifiersObj[0].Value), &activeModifiers)
		if err != nil {
			return err
		}
		version = modifiersObj[0].Version
	}

	// Current timestamp
	now := time.Now().Unix()

	// Add new modifiers
	for _, modifier := range modifiers {
		var expiryTime int64 = 0
		if modifier.DurationSec > 0 {
			expiryTime = now + int64(modifier.DurationSec)
		}

		activeModifier := &ActiveRewardModifier{
			Id:           modifier.Id,
			Operator:     modifier.Operator,
			Value:        modifier.Value,
			StartTimeSec: now,
			EndTimeSec:   expiryTime,
		}

		activeModifiers = append(activeModifiers, activeModifier)
	}

	// Store updated modifiers
	modifiersData, err := json.Marshal(activeModifiers)
	if err != nil {
		return err
	}

	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{
		{
			Collection:      "modifiers",
			Key:             "energy_modifiers",
			UserID:          userID,
			Value:           string(modifiersData),
			Version:         version,
			PermissionRead:  1,
			PermissionWrite: 1,
		},
	})

	return err
}

// Helper function to apply reward modifiers
func (e *NakamaEconomySystem) applyRewardModifiers(ctx context.Context, nk runtime.NakamaModule, userID string, modifiers []*RewardModifier) error {
	// Get current modifiers
	modifiersObj, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: "modifiers",
			Key:        "reward_modifiers",
			UserID:     userID,
		},
	})

	var activeModifiers []*ActiveRewardModifier
	var version string

	if err != nil || len(modifiersObj) == 0 {
		// No existing modifiers
		activeModifiers = make([]*ActiveRewardModifier, 0)
	} else {
		// Parse existing modifiers
		err = json.Unmarshal([]byte(modifiersObj[0].Value), &activeModifiers)
		if err != nil {
			return err
		}
		version = modifiersObj[0].Version
	}

	// Current timestamp
	now := time.Now().Unix()

	// Add new modifiers
	for _, modifier := range modifiers {
		var expiryTime int64 = 0
		if modifier.DurationSec > 0 {
			expiryTime = now + int64(modifier.DurationSec)
		}

		activeModifier := &ActiveRewardModifier{
			Id:           modifier.Id,
			Type:         modifier.Type,
			Operator:     modifier.Operator,
			Value:        modifier.Value,
			EndTimeSec:   expiryTime,
			StartTimeSec: now,
		}

		activeModifiers = append(activeModifiers, activeModifier)
	}

	// Store updated modifiers
	modifiersData, err := json.Marshal(activeModifiers)
	if err != nil {
		return err
	}

	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{
		{
			Collection:      "modifiers",
			Key:             "reward_modifiers",
			UserID:          userID,
			Value:           string(modifiersData),
			Version:         version,
			PermissionRead:  1,
			PermissionWrite: 1,
		},
	})

	return err
}

func (e *NakamaEconomySystem) DonationClaim(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, donationClaims map[string]*EconomyDonationClaimRequestDetails) (donationsList *EconomyDonationsList, err error) {
	if userID == "" {
		return nil, runtime.NewError("user ID is empty", 3) // INVALID_ARGUMENT
	}

	if len(donationClaims) == 0 {
		return nil, runtime.NewError("donation claims are empty", 3) // INVALID_ARGUMENT
	}

	// Initialize the response
	donationsList = &EconomyDonationsList{
		Donations: make([]*EconomyDonation, 0, len(donationClaims)),
	}

	// Get existing donations for the user
	existingDonations, err := e.getUserDonations(ctx, nk, userID)
	if err != nil {
		logger.Error("Failed to get user donations: %v", err)
		return nil, runtime.NewError("Failed to get user donations", 13) // INTERNAL
	}

	// Track donations to update in storage
	donationsToUpdate := make([]*runtime.StorageWrite, 0)

	// Process each donation claim
	for donationID, claimDetails := range donationClaims {
		if donationID == "" || claimDetails == nil {
			continue
		}

		// Find the donation in existing donations
		donation, exists := existingDonations[donationID]
		if !exists {
			logger.Warn("Donation %s not found for user %s", donationID, userID)
			continue
		}

		// Verify donation is claimable
		if donation.CurrentTimeSec > 0 {
			logger.Warn("Donation %s already claimed by user %s", donationID, userID)
			continue
		}

		// Mark donation as claimed
		now := time.Now().Unix()
		donation.CurrentTimeSec = now

		// Process reward if configured and available
		var reward *Reward
		if e.config != nil && e.config.Donations != nil {
			// Find donation config
			donationConfig, configExists := e.config.Donations[donationID]
			if configExists && donationConfig.RecipientReward != nil {
				// Roll the reward
				reward, err = e.RewardRoll(ctx, logger, nk, userID, donationConfig.RecipientReward)
				if err != nil {
					logger.Error("Failed to roll reward for donation %s: %v", donationID, err)
				} else if reward != nil {
					// Grant the reward to user
					_, _, _, err = e.RewardGrant(ctx, logger, nk, userID, reward, map[string]interface{}{
						"donation_id": donationID,
					}, false)
					if err != nil {
						logger.Error("Failed to grant reward for donation %s: %v", donationID, err)
					}
				}
			}
		}

		// Prepare donation for storage update
		key := fmt.Sprintf("donation:%s", donationID)
		donationData, err := json.Marshal(donation)
		if err != nil {
			logger.Error("Failed to marshal donation data: %v", err)
			continue
		}

		// Add donation to storage update batch
		donationsToUpdate = append(donationsToUpdate, &runtime.StorageWrite{
			Collection:      "donations",
			Key:             key,
			UserID:          userID,
			Value:           string(donationData),
			Version:         "", // Use empty string for new donation
			PermissionRead:  1,  // Only owner can read
			PermissionWrite: 1,  // Only owner can write
		})

		// Add to response list
		donationsList.Donations = append(donationsList.Donations, donation)
	}

	// Execute storage updates if we have any
	if len(donationsToUpdate) > 0 {
		_, err = nk.StorageWrite(ctx, donationsToUpdate)
		if err != nil {
			logger.Error("Failed to update donations: %v", err)
			return nil, runtime.NewError("Failed to update donations", 13) // INTERNAL
		}
	}

	return donationsList, nil
}

// Helper function to retrieve a user's donations
func (e *NakamaEconomySystem) getUserDonations(ctx context.Context, nk runtime.NakamaModule, userID string) (map[string]*EconomyDonation, error) {
	// Query the storage for all donations
	storageObjects, _, err := nk.StorageList(ctx, "donations", userID, "", 100, "")
	if err != nil {
		return nil, err
	}

	donations := make(map[string]*EconomyDonation)

	// Parse each storage object into donations
	for _, obj := range storageObjects {
		if !strings.HasPrefix(obj.Key, "donation:") {
			continue
		}

		// Extract donation ID from key
		donationID := strings.TrimPrefix(obj.Key, "donation:")

		// Parse the donation data
		var donation EconomyDonation
		err = json.Unmarshal([]byte(obj.Value), &donation)
		if err != nil {
			continue // Skip invalid donations
		}

		donations[donationID] = &donation
	}

	return donations, nil
}

func (e *NakamaEconomySystem) DonationGet(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userIDs []string) (donationsList *EconomyDonationsByUserList, err error) {
	// TODO: Implement donation get logic
	return nil, nil
}

func (e *NakamaEconomySystem) DonationGive(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, donationID, fromUserID string) (updatedWallet map[string]int64, updatedInventory *Inventory, rewardModifiers []*ActiveRewardModifier, contributorReward *Reward, timestamp int64, err error) {
	// TODO: Implement donation give logic
	return nil, nil, nil, nil, 0, nil
}

func (e *NakamaEconomySystem) DonationRequest(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, donationID string) (donation *EconomyDonation, success bool, err error) {
	// TODO: Implement donation request logic
	return nil, false, nil
}

func (e *NakamaEconomySystem) List(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (storeItems map[string]*EconomyConfigStoreItem, placements map[string]*EconomyConfigPlacement, rewardModifiers []*ActiveRewardModifier, timestamp int64, err error) {
	// TODO: Implement list logic
	return nil, nil, nil, 0, nil
}

func (e *NakamaEconomySystem) Grant(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, currencies map[string]int64, items map[string]int64, modifiers []*RewardModifier, walletMetadata map[string]interface{}) (updatedWallet map[string]int64, rewardModifiers []*ActiveRewardModifier, timestamp int64, err error) {
	// TODO: Implement grant logic
	return nil, nil, 0, nil
}

func (e *NakamaEconomySystem) UnmarshalWallet(account *api.Account) (wallet map[string]int64, err error) {
	// TODO: Implement wallet unmarshal logic
	return nil, nil
}

func (e *NakamaEconomySystem) PurchaseIntent(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, itemID string, store EconomyStoreType, sku string) (err error) {
	// TODO: Implement purchase intent logic
	return nil
}

func (e *NakamaEconomySystem) PurchaseItem(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, userID, itemID string, store EconomyStoreType, receipt string) (updatedWallet map[string]int64, updatedInventory *Inventory, reward *Reward, isSandboxPurchase bool, err error) {
	// TODO: Implement purchase item logic
	return nil, nil, nil, false, nil
}

func (e *NakamaEconomySystem) PurchaseRestore(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, store EconomyStoreType, receipts []string) (err error) {
	// TODO: Implement purchase restore logic
	return nil
}

func (e *NakamaEconomySystem) PlacementStatus(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, rewardID, placementID string, retryCount int) (resp *EconomyPlacementStatus, err error) {
	// TODO: Implement placement status logic
	return nil, nil
}

func (e *NakamaEconomySystem) PlacementStart(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, placementID string, metadata map[string]string) (resp *EconomyPlacementStatus, err error) {
	// TODO: Implement placement start logic
	return nil, nil
}

func (e *NakamaEconomySystem) PlacementSuccess(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, rewardID, placementID string) (reward *Reward, placementMetadata map[string]string, err error) {
	// TODO: Implement placement success logic
	return nil, nil, nil
}

func (e *NakamaEconomySystem) PlacementFail(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, rewardID, placementID string) (placementMetadata map[string]string, err error) {
	// TODO: Implement placement fail logic
	return nil, nil
}

func (e *NakamaEconomySystem) SetOnDonationClaimReward(fn OnReward[*EconomyConfigDonation]) {
	// TODO: Implement custom reward hook
}

func (e *NakamaEconomySystem) SetOnDonationContributorReward(fn OnReward[*EconomyConfigDonation]) {
	// TODO: Implement custom reward hook
}

func (e *NakamaEconomySystem) SetOnPlacementReward(fn OnReward[*EconomyPlacementInfo]) {
	// TODO: Implement custom reward hook
}

func (e *NakamaEconomySystem) SetOnStoreItemReward(fn OnReward[*EconomyConfigStoreItem]) {
	// TODO: Implement custom reward hook
}

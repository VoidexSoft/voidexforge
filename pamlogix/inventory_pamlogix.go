package pamlogix

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/heroiclabs/nakama-common/runtime"
)

// NakamaInventorySystem implements the InventorySystem interface using Nakama as the backend.
type NakamaInventorySystem struct {
	config          *InventoryConfig
	onConsumeReward OnReward[*InventoryConfigItem]
	configSource    ConfigSource[*InventoryConfigItem]
}

// NewNakamaInventorySystem creates a new instance of the inventory system with the given configuration.
func NewNakamaInventorySystem(config *InventoryConfig) *NakamaInventorySystem {
	// Pre-compute item sets for fast lookup
	if config != nil && len(config.Items) > 0 {
		config.ItemSets = make(map[string]map[string]bool)
		for itemID, item := range config.Items {
			for _, setID := range item.ItemSets {
				if _, exists := config.ItemSets[setID]; !exists {
					config.ItemSets[setID] = make(map[string]bool)
				}
				config.ItemSets[setID][itemID] = true
			}
		}
	}

	return &NakamaInventorySystem{
		config: config,
	}
}

// GetType returns the system type for the inventory system.
func (i *NakamaInventorySystem) GetType() SystemType {
	return SystemTypeInventory
}

// GetConfig returns the configuration for the inventory system.
func (i *NakamaInventorySystem) GetConfig() any {
	return i.config
}

// List will return the items defined as well as the computed item sets for the user by ID.
func (i *NakamaInventorySystem) List(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, category string) (items map[string]*InventoryConfigItem, itemSets map[string][]string, err error) {
	if i.config == nil || len(i.config.Items) == 0 {
		// No items are configured
		return make(map[string]*InventoryConfigItem), make(map[string][]string), nil
	}

	// Filter items by category if specified
	filteredItems := make(map[string]*InventoryConfigItem)
	for id, item := range i.config.Items {
		if category == "" || item.Category == category {
			if !item.Disabled {
				filteredItems[id] = item
			}
		}
	}

	// Build item sets response
	resultItemSets := make(map[string][]string)
	for setID, itemMap := range i.config.ItemSets {
		itemsList := make([]string, 0, len(itemMap))
		for itemID := range itemMap {
			// Only include items that match the category filter and are not disabled
			if item, exists := i.config.Items[itemID]; exists && !item.Disabled {
				if category == "" || item.Category == category {
					itemsList = append(itemsList, itemID)
				}
			}
		}
		if len(itemsList) > 0 {
			resultItemSets[setID] = itemsList
		}
	}

	// Check for additional config items if a config source is set
	if i.configSource != nil {
		additionalItems, err := i.getAdditionalItems(ctx, logger, nk, userID, category)
		if err != nil {
			logger.Error("Failed to get additional items: %v", err)
			// Continue with base items
		} else {
			// Merge additional items with filtered items
			for id, item := range additionalItems {
				if !item.Disabled {
					filteredItems[id] = item
				}
			}
		}
	}

	return filteredItems, resultItemSets, nil
}

// ListInventoryItems will return the items which are part of a user's inventory by ID.
func (i *NakamaInventorySystem) ListInventoryItems(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, category string) (*Inventory, error) {
	if i.config == nil || len(i.config.Items) == 0 {
		// No items are configured
		return &Inventory{Items: make(map[string]*InventoryItem)}, nil
	}

	// Get user's inventory from storage
	userInventory, err := i.getUserInventory(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user inventory: %v", err)
		return nil, ErrInternal
	}

	// Filter by category if specified
	if category != "" {
		filteredInventory := &Inventory{Items: make(map[string]*InventoryItem)}
		for id, item := range userInventory.Items {
			configItem, exists := i.config.Items[id]
			if exists && configItem.Category == category && !configItem.Disabled {
				filteredInventory.Items[id] = item
			}
		}
		return filteredInventory, nil
	}

	return userInventory, nil
}

// ConsumeItems will deduct the item(s) from the user's inventory and run the consume reward for each one, if defined.
func (i *NakamaInventorySystem) ConsumeItems(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, itemIDs, instanceIDs map[string]int64, overConsume bool) (updatedInventory *Inventory, rewards map[string][]*Reward, instanceRewards map[string][]*Reward, err error) {
	if i.config == nil || len(i.config.Items) == 0 {
		// No items are configured
		return &Inventory{Items: make(map[string]*InventoryItem)}, nil, nil, nil
	}

	// Initialize return values
	rewards = make(map[string][]*Reward)
	instanceRewards = make(map[string][]*Reward)

	// Get user's current inventory
	userInventory, err := i.getUserInventory(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user inventory: %v", err)
		return nil, nil, nil, ErrInternal
	}

	// Prepare storage operations
	var storageOps []*runtime.StorageWrite

	// Process item ID-based consumption
	for itemID, count := range itemIDs {
		// Check if item exists in configuration
		configItem, exists := i.config.Items[itemID]
		if !exists {
			logger.Warn("Attempted to consume non-existent item: %s", itemID)
			return nil, nil, nil, ErrBadInput
		}

		// Check if item exists in user's inventory
		inventoryItem, exists := userInventory.Items[itemID]
		if !exists {
			logger.Warn("Attempted to consume item user doesn't have: %s", itemID)
			return nil, nil, nil, ErrBadInput
		}

		// Check if item is consumable
		if !configItem.Consumable {
			logger.Warn("Attempted to consume non-consumable item: %s", itemID)
			return nil, nil, nil, ErrBadInput
		}

		// Check if user has enough items
		if inventoryItem.Count < count && !overConsume {
			logger.Warn("Insufficient items to consume: %s (have: %d, need: %d)", itemID, inventoryItem.Count, count)
			return nil, nil, nil, ErrBadInput
		}

		// Consume the items
		inventoryItem.Count -= count
		inventoryItem.UpdateTimeSec = time.Now().Unix()

		// Remove item if count reaches 0 and keep_zero is false
		if inventoryItem.Count <= 0 && !configItem.KeepZero {
			delete(userInventory.Items, itemID)

			// Add delete operation - use regular write with empty value
			storageOps = append(storageOps, &runtime.StorageWrite{
				Collection:      "inventory",
				Key:             itemID,
				UserID:          userID,
				Version:         "*",
				PermissionRead:  1,
				PermissionWrite: 1,
				Value:           "", // Empty value for deletion
			})
		} else {
			// Update item in storage
			itemData, err := json.Marshal(inventoryItem)
			if err != nil {
				logger.Error("Failed to marshal inventory item: %v", err)
				return nil, nil, nil, ErrInternal
			}

			storageOps = append(storageOps, &runtime.StorageWrite{
				Collection:      "inventory",
				Key:             itemID,
				UserID:          userID,
				Value:           string(itemData),
				Version:         "*",
				PermissionRead:  1,
				PermissionWrite: 1,
			})
		}

		// Process reward if applicable
		if configItem.ConsumeReward != nil {
			r, err := i.processItemReward(ctx, logger, nk, userID, itemID, configItem)
			if err != nil {
				logger.Error("Failed to process item reward: %v", err)
				// Continue, don't fail the entire operation
			} else if r != nil {
				if rewards[itemID] == nil {
					rewards[itemID] = []*Reward{r}
				} else {
					rewards[itemID] = append(rewards[itemID], r)
				}
			}
		}
	}

	// Process instance ID-based consumption
	for instanceID, count := range instanceIDs {
		var foundItem *InventoryItem
		var itemID string

		// Find the item with this instance ID
		for id, item := range userInventory.Items {
			if item.InstanceId == instanceID {
				foundItem = item
				itemID = id
				break
			}
		}

		if foundItem == nil || itemID == "" {
			logger.Warn("Attempted to consume non-existent instance: %s", instanceID)
			return nil, nil, nil, ErrBadInput
		}

		// Check if item exists in configuration
		configItem, exists := i.config.Items[itemID]
		if !exists {
			logger.Warn("Attempted to consume item with invalid configuration: %s", itemID)
			return nil, nil, nil, ErrBadInput
		}

		// Check if item is consumable
		if !configItem.Consumable {
			logger.Warn("Attempted to consume non-consumable item instance: %s", instanceID)
			return nil, nil, nil, ErrBadInput
		}

		// Check if user has enough items
		if foundItem.Count < count && !overConsume {
			logger.Warn("Insufficient items to consume: %s (have: %d, need: %d)", instanceID, foundItem.Count, count)
			return nil, nil, nil, ErrBadInput
		}

		// Consume the items
		foundItem.Count -= count
		foundItem.UpdateTimeSec = time.Now().Unix()

		// Remove item if count reaches 0 and keep_zero is false
		if foundItem.Count <= 0 && !configItem.KeepZero {
			delete(userInventory.Items, itemID)

			// Add delete operation - use regular write with empty value
			storageOps = append(storageOps, &runtime.StorageWrite{
				Collection:      "inventory",
				Key:             itemID,
				UserID:          userID,
				Version:         "*",
				PermissionRead:  1,
				PermissionWrite: 1,
				Value:           "", // Empty value for deletion
			})
		} else {
			// Update item in storage
			itemData, err := json.Marshal(foundItem)
			if err != nil {
				logger.Error("Failed to marshal inventory item: %v", err)
				return nil, nil, nil, ErrInternal
			}

			storageOps = append(storageOps, &runtime.StorageWrite{
				Collection:      "inventory",
				Key:             itemID,
				UserID:          userID,
				Value:           string(itemData),
				Version:         "*",
				PermissionRead:  1,
				PermissionWrite: 1,
			})
		}

		// Process reward if applicable
		if configItem.ConsumeReward != nil {
			r, err := i.processItemReward(ctx, logger, nk, userID, itemID, configItem)
			if err != nil {
				logger.Error("Failed to process item reward: %v", err)
				// Continue, don't fail the entire operation
			} else if r != nil {
				if instanceRewards[instanceID] == nil {
					instanceRewards[instanceID] = []*Reward{r}
				} else {
					instanceRewards[instanceID] = append(instanceRewards[instanceID], r)
				}
			}
		}
	}

	// Write changes to storage if there are any
	if len(storageOps) > 0 {
		_, err = nk.StorageWrite(ctx, storageOps)
		if err != nil {
			logger.Error("Failed to update inventory storage: %v", err)
			return nil, nil, nil, ErrInternal
		}
	}

	return userInventory, rewards, instanceRewards, nil
}

// GrantItems will add the item(s) to a user's inventory by ID.
func (i *NakamaInventorySystem) GrantItems(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, itemIDs map[string]int64, ignoreLimits bool) (updatedInventory *Inventory, newItems map[string]*InventoryItem, updatedItems map[string]*InventoryItem, notGrantedItemIDs map[string]int64, err error) {
	if i.config == nil || len(i.config.Items) == 0 {
		// No items are configured
		return &Inventory{Items: make(map[string]*InventoryItem)}, nil, nil, nil, nil
	}

	// Initialize return values
	newItems = make(map[string]*InventoryItem)
	updatedItems = make(map[string]*InventoryItem)
	notGrantedItemIDs = make(map[string]int64)

	// Get user's current inventory
	userInventory, err := i.getUserInventory(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user inventory: %v", err)
		return nil, nil, nil, nil, ErrInternal
	}

	// Check if we need to validate limits
	if !ignoreLimits && i.config.Limits != nil {
		// Check category limits
		categoryCount := make(map[string]int64)
		for _, item := range userInventory.Items {
			configItem, exists := i.config.Items[item.Id]
			if exists && configItem.Category != "" {
				categoryCount[configItem.Category] += item.Count
			}
		}

		// Check item set limits
		itemSetCount := make(map[string]int64)
		for id, item := range userInventory.Items {
			configItem, exists := i.config.Items[id]
			if exists {
				for _, setID := range configItem.ItemSets {
					itemSetCount[setID] += item.Count
				}
			}
		}

		// Pre-calculate what would be added to each category and item set
		categoryAdds := make(map[string]int64)
		itemSetAdds := make(map[string]int64)

		for itemID, count := range itemIDs {
			configItem, exists := i.config.Items[itemID]
			if !exists {
				notGrantedItemIDs[itemID] = count
				continue
			}

			if configItem.Category != "" {
				categoryAdds[configItem.Category] += count
			}

			for _, setID := range configItem.ItemSets {
				itemSetAdds[setID] += count
			}
		}

		// Check if adding would exceed category limits
		for category, limit := range i.config.Limits.Categories {
			if add, exists := categoryAdds[category]; exists {
				if categoryCount[category]+add > limit {
					// Reject all items of this category
					for itemID, count := range itemIDs {
						configItem, exists := i.config.Items[itemID]
						if exists && configItem.Category == category {
							notGrantedItemIDs[itemID] = count
							delete(itemIDs, itemID)
						}
					}
				}
			}
		}

		// Check if adding would exceed item set limits
		for setID, limit := range i.config.Limits.ItemSets {
			if add, exists := itemSetAdds[setID]; exists {
				if itemSetCount[setID]+add > limit {
					// Reject all items in this set
					for itemID, count := range itemIDs {
						configItem, exists := i.config.Items[itemID]
						if exists {
							for _, itemSetID := range configItem.ItemSets {
								if itemSetID == setID {
									notGrantedItemIDs[itemID] = count
									delete(itemIDs, itemID)
									break
								}
							}
						}
					}
				}
			}
		}
	}

	// Prepare storage operations
	var storageOps []*runtime.StorageWrite

	// Process remaining items
	now := time.Now().Unix()
	for itemID, count := range itemIDs {
		configItem, exists := i.config.Items[itemID]
		if !exists {
			logger.Warn("Attempted to grant non-existent item: %s", itemID)
			notGrantedItemIDs[itemID] = count
			continue
		}

		if configItem.Disabled {
			logger.Warn("Attempted to grant disabled item: %s", itemID)
			notGrantedItemIDs[itemID] = count
			continue
		}

		// Check item-specific limit
		existingItem, exists := userInventory.Items[itemID]
		if exists && configItem.MaxCount > 0 && existingItem.Count+count > configItem.MaxCount && !ignoreLimits {
			logger.Info("Item limit reached: %s (have: %d, max: %d)", itemID, existingItem.Count, configItem.MaxCount)
			notGrantedItemIDs[itemID] = count
			continue
		}

		if exists {
			// Update existing item
			existingItem.Count += count
			existingItem.UpdateTimeSec = now
			existingItem.MaxCount = configItem.MaxCount
			existingItem.Stackable = configItem.Stackable
			existingItem.Consumable = configItem.Consumable

			// Update properties from config if they exist
			if configItem.StringProperties != nil {
				if existingItem.StringProperties == nil {
					existingItem.StringProperties = make(map[string]string)
				}
				for key, value := range configItem.StringProperties {
					existingItem.StringProperties[key] = value
				}
			}

			if configItem.NumericProperties != nil {
				if existingItem.NumericProperties == nil {
					existingItem.NumericProperties = make(map[string]float64)
				}
				for key, value := range configItem.NumericProperties {
					existingItem.NumericProperties[key] = value
				}
			}

			updatedItems[itemID] = existingItem

			// Prepare storage update
			itemData, err := json.Marshal(existingItem)
			if err != nil {
				logger.Error("Failed to marshal inventory item: %v", err)
				return nil, nil, nil, nil, ErrInternal
			}

			storageOps = append(storageOps, &runtime.StorageWrite{
				Collection:      "inventory",
				Key:             itemID,
				UserID:          userID,
				Value:           string(itemData),
				Version:         "*",
				PermissionRead:  1,
				PermissionWrite: 1,
			})
		} else {
			// Create new item
			newItem := &InventoryItem{
				Id:                itemID,
				Name:              configItem.Name,
				Description:       configItem.Description,
				Category:          configItem.Category,
				ItemSets:          configItem.ItemSets,
				Count:             count,
				MaxCount:          configItem.MaxCount,
				Stackable:         configItem.Stackable,
				Consumable:        configItem.Consumable,
				OwnedTimeSec:      now,
				UpdateTimeSec:     now,
				StringProperties:  configItem.StringProperties,
				NumericProperties: configItem.NumericProperties,
			}

			// Generate instance ID for non-stackable items
			if !configItem.Stackable || count == 1 {
				newItem.InstanceId = uuid.New().String()
			}

			userInventory.Items[itemID] = newItem
			newItems[itemID] = newItem

			// Prepare storage update
			itemData, err := json.Marshal(newItem)
			if err != nil {
				logger.Error("Failed to marshal new inventory item: %v", err)
				return nil, nil, nil, nil, ErrInternal
			}

			storageOps = append(storageOps, &runtime.StorageWrite{
				Collection:      "inventory",
				Key:             itemID,
				UserID:          userID,
				Value:           string(itemData),
				Version:         "*",
				PermissionRead:  1,
				PermissionWrite: 1,
			})
		}
	}

	// Write changes to storage if there are any
	if len(storageOps) > 0 {
		_, err = nk.StorageWrite(ctx, storageOps)
		if err != nil {
			logger.Error("Failed to update inventory storage: %v", err)
			return nil, nil, nil, nil, ErrInternal
		}
	}

	return userInventory, newItems, updatedItems, notGrantedItemIDs, nil
}

// UpdateItems will update the properties which are stored on each item by instance ID for a user.
func (i *NakamaInventorySystem) UpdateItems(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, instanceIDs map[string]*InventoryUpdateItemProperties) (updatedInventory *Inventory, err error) {
	if i.config == nil || len(i.config.Items) == 0 {
		// No items are configured
		return &Inventory{Items: make(map[string]*InventoryItem)}, nil
	}

	// Get user's current inventory
	userInventory, err := i.getUserInventory(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user inventory: %v", err)
		return nil, ErrInternal
	}

	// Prepare storage operations
	var storageOps []*runtime.StorageWrite

	// Find and update each instanced item
	for instanceID, props := range instanceIDs {
		var foundItem *InventoryItem
		var itemID string

		// Find the item with this instance ID
		for id, item := range userInventory.Items {
			if item.InstanceId == instanceID {
				foundItem = item
				itemID = id
				break
			}
		}

		if foundItem == nil || itemID == "" {
			logger.Warn("Attempted to update non-existent instance: %s", instanceID)
			continue
		}

		// Update string properties
		if props.StringProperties != nil {
			if foundItem.StringProperties == nil {
				foundItem.StringProperties = make(map[string]string)
			}
			for key, value := range props.StringProperties {
				foundItem.StringProperties[key] = value
			}
		}

		// Update numeric properties
		if props.NumericProperties != nil {
			if foundItem.NumericProperties == nil {
				foundItem.NumericProperties = make(map[string]float64)
			}
			for key, value := range props.NumericProperties {
				foundItem.NumericProperties[key] = value
			}
		}

		// Update timestamp
		foundItem.UpdateTimeSec = time.Now().Unix()

		// Prepare storage update
		itemData, err := json.Marshal(foundItem)
		if err != nil {
			logger.Error("Failed to marshal inventory item: %v", err)
			return nil, ErrInternal
		}

		storageOps = append(storageOps, &runtime.StorageWrite{
			Collection:      "inventory",
			Key:             itemID,
			UserID:          userID,
			Value:           string(itemData),
			Version:         "*",
			PermissionRead:  1,
			PermissionWrite: 1,
		})
	}

	// Write changes to storage if there are any
	if len(storageOps) > 0 {
		_, err = nk.StorageWrite(ctx, storageOps)
		if err != nil {
			logger.Error("Failed to update inventory storage: %v", err)
			return nil, ErrInternal
		}
	}

	return userInventory, nil
}

// SetOnConsumeReward sets a custom reward function which will run after an inventory items' consume reward is rolled.
func (i *NakamaInventorySystem) SetOnConsumeReward(fn OnReward[*InventoryConfigItem]) {
	i.onConsumeReward = fn
}

// SetConfigSource sets a custom additional config lookup function.
func (i *NakamaInventorySystem) SetConfigSource(fn ConfigSource[*InventoryConfigItem]) {
	i.configSource = fn
	if i.config != nil {
		i.config.ConfigSource = fn
	}
}

// Helper function to get a user's inventory from storage
func (i *NakamaInventorySystem) getUserInventory(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (*Inventory, error) {
	inventory := &Inventory{
		Items: make(map[string]*InventoryItem),
	}

	// Retrieve all inventory objects from storage
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: "inventory",
			UserID:     userID,
			Key:        "",
		},
	})
	if err != nil {
		logger.Error("Failed to read inventory from storage: %v", err)
		return inventory, err
	}

	// Process each item in the inventory
	for _, obj := range objects {
		var item InventoryItem
		err = json.Unmarshal([]byte(obj.Value), &item)
		if err != nil {
			logger.Error("Failed to unmarshal inventory item: %v", err)
			continue
		}

		// Only include non-disabled items
		configItem, exists := i.config.Items[obj.Key]
		if exists && !configItem.Disabled {
			// Update from config if needed
			item.Name = configItem.Name
			item.Description = configItem.Description
			item.Category = configItem.Category
			item.ItemSets = configItem.ItemSets
			item.MaxCount = configItem.MaxCount
			item.Stackable = configItem.Stackable
			item.Consumable = configItem.Consumable

			// Add to inventory
			inventory.Items[obj.Key] = &item
		}
	}

	return inventory, nil
}

// Helper function to process an item's consume reward
func (i *NakamaInventorySystem) processItemReward(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, itemID string, configItem *InventoryConfigItem) (*Reward, error) {
	if configItem.ConsumeReward == nil {
		return nil, nil
	}

	// Create empty reward
	reward := &Reward{
		Items:           make(map[string]int64),
		Currencies:      make(map[string]int64),
		Energies:        make(map[string]int32),
		EnergyModifiers: make([]*RewardEnergyModifier, 0),
		RewardModifiers: make([]*RewardModifier, 0),
		GrantTimeSec:    time.Now().Unix(),
	}

	// Apply custom reward handler if set
	if i.onConsumeReward != nil {
		var err error
		reward, err = i.onConsumeReward(ctx, logger, nk, userID, itemID, configItem, configItem.ConsumeReward, reward)
		if err != nil {
			logger.Error("Error in custom reward handler: %v", err)
			return nil, err
		}
	}

	return reward, nil
}

// Helper function to get additional items from the config source
func (i *NakamaInventorySystem) getAdditionalItems(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, category string) (map[string]*InventoryConfigItem, error) {
	if i.configSource == nil {
		return nil, nil
	}

	additionalItems := make(map[string]*InventoryConfigItem)

	// If a category is specified, only get config for that category
	if category != "" {
		item, err := i.configSource(ctx, logger, nk, userID, category)
		if err != nil {
			logger.Error("Failed to get additional config item for category %s: %v", category, err)
			return additionalItems, err
		}
		if item != nil {
			additionalItems[category] = item
		}
	} else {
		// Get all possible items from main config categories
		for itemID, item := range i.config.Items {
			if item.Category != "" {
				configItem, err := i.configSource(ctx, logger, nk, userID, item.Category)
				if err != nil {
					logger.Error("Failed to get additional config item for category %s: %v", item.Category, err)
					continue
				}
				if configItem != nil {
					additionalItems[itemID] = configItem
				}
			}
		}
	}

	return additionalItems, nil
}

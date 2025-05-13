package pamlogix

import (
	"context"

	"github.com/heroiclabs/nakama-common/runtime"
)

// InventorySystemImpl implements the InventorySystem interface for managing player inventories.
type InventorySystemImpl struct {
	config          *InventoryConfig
	onConsumeReward OnReward[*InventoryConfigItem]
	configSource    ConfigSource[*InventoryConfigItem]
}

// NewInventorySystem creates a new instance of the InventorySystem implementation.
func NewInventorySystem(config *InventoryConfig) InventorySystem {
	// Make sure the item sets map is initialized
	if config.ItemSets == nil {
		config.ItemSets = make(map[string]map[string]bool)
	}

	// Populate the item sets map from the item definitions
	for itemID, item := range config.Items {
		if item != nil && len(item.ItemSets) > 0 {
			for _, setName := range item.ItemSets {
				if _, ok := config.ItemSets[setName]; !ok {
					config.ItemSets[setName] = make(map[string]bool)
				}
				config.ItemSets[setName][itemID] = true
			}
		}
	}

	return &InventorySystemImpl{
		config: config,
	}
}

// GetType provides the runtime type of the gameplay system.
func (is *InventorySystemImpl) GetType() SystemType {
	return SystemTypeInventory
}

// GetConfig returns the configuration type of the gameplay system.
func (is *InventorySystemImpl) GetConfig() any {
	return is.config
}

// List will return the items defined as well as the computed item sets for the user by ID.
func (is *InventorySystemImpl) List(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, category string) (items map[string]*InventoryConfigItem, itemSets map[string][]string, err error) {
	// Initialize return maps
	items = make(map[string]*InventoryConfigItem)
	itemSets = make(map[string][]string)

	// Check if we should use a personalized config via the config source
	var personalizedItems map[string]*InventoryConfigItem
	if is.configSource != nil && userID != "" {
		for itemID := range is.config.Items {
			item, err := is.configSource(ctx, logger, nk, userID, itemID)
			if err != nil {
				logger.Error("Error loading personalized item config: %v", err)
				continue
			}
			if personalizedItems == nil {
				personalizedItems = make(map[string]*InventoryConfigItem)
			}
			personalizedItems[itemID] = item
		}
	}

	// Filter items by category if specified
	for itemID, item := range is.config.Items {
		// Skip disabled items
		if item.Disabled {
			continue
		}

		// Apply category filter if specified
		if category != "" && item.Category != category {
			continue
		}

		// Use personalized item if available
		if personalizedItems != nil {
			if personalizedItem, ok := personalizedItems[itemID]; ok {
				items[itemID] = personalizedItem
				continue
			}
		}

		// Use the original item
		items[itemID] = item
	}

	// Convert item sets map to the return format
	for setName, itemsInSet := range is.config.ItemSets {
		itemsList := make([]string, 0, len(itemsInSet))
		for itemID := range itemsInSet {
			itemsList = append(itemsList, itemID)
		}
		itemSets[setName] = itemsList
	}

	return items, itemSets, nil
}

// ListInventoryItems will return the items which are part of a user's inventory by ID.
func (is *InventorySystemImpl) ListInventoryItems(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, category string) (inventory *Inventory, err error) {
	// Use storage engine to retrieve user's inventory
	objectIDs := []*runtime.StorageRead{
		{
			Collection: "inventory",
			Key:        "user_items",
			UserID:     userID,
		},
	}

	// Read from storage
	objects, err := nk.StorageRead(ctx, objectIDs)
	if err != nil {
		logger.Error("Error reading inventory from storage: %v", err)
		return nil, ErrInternal
	}

	// Initialize empty inventory
	inventory = &Inventory{
		Items: make(map[string]*InventoryItem),
	}

	// If no inventory found, return empty inventory
	if len(objects) == 0 {
		return inventory, nil
	}

	// Deserialize the inventory
	if err := nk.JSON.Unmarshal(objects[0].Value, &inventory); err != nil {
		logger.Error("Error unmarshaling inventory: %v", err)
		return nil, ErrInternal
	}

	// Filter by category if specified
	if category != "" && inventory.Items != nil {
		filteredItems := make(map[string]*InventoryItem)

		// Get item definitions
		itemDefs, _, err := is.List(ctx, logger, nk, userID, "")
		if err != nil {
			logger.Error("Error getting item definitions: %v", err)
			return nil, err
		}

		// Filter items by category
		for instanceID, item := range inventory.Items {
			if itemDef, ok := itemDefs[item.Id]; ok && itemDef.Category == category {
				filteredItems[instanceID] = item
			}
		}

		inventory.Items = filteredItems
	}

	return inventory, nil
}

// ConsumeItems will deduct the item(s) from the user's inventory and run the consume reward for each one, if defined.
func (is *InventorySystemImpl) ConsumeItems(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, itemIDs, instanceIDs map[string]int64, overConsume bool) (updatedInventory *Inventory, rewards map[string][]*Reward, instanceRewards map[string][]*Reward, err error) {
	// Get current inventory
	inventory, err := is.ListInventoryItems(ctx, logger, nk, userID, "")
	if err != nil {
		return nil, nil, nil, err
	}

	// Get item definitions
	itemDefs, _, err := is.List(ctx, logger, nk, userID, "")
	if err != nil {
		return nil, nil, nil, err
	}

	// Initialize return values
	rewards = make(map[string][]*Reward)
	instanceRewards = make(map[string][]*Reward)

	// Track changes for storage update
	hasChanges := false

	// Process consumption by itemID
	for itemID, consumeCount := range itemIDs {
		// Skip invalid counts
		if consumeCount <= 0 {
			continue
		}

		// Check if item exists in definitions
		itemDef, exists := itemDefs[itemID]
		if !exists {
			logger.Warn("Attempted to consume unknown item: %s", itemID)
			continue
		}

		// Check if consumable
		if !itemDef.Consumable {
			logger.Warn("Attempted to consume non-consumable item: %s", itemID)
			continue
		}

		// Count how many of this item we have across all instances
		totalCount := int64(0)
		for _, invItem := range inventory.Items {
			if invItem.Id == itemID {
				totalCount += invItem.Count
			}
		}

		// Check if we have enough
		if totalCount < consumeCount && !overConsume {
			logger.Warn("Not enough items to consume. Item: %s, Have: %d, Need: %d", itemID, totalCount, consumeCount)
			continue
		}

		// Prepare to consume items
		remainingToConsume := consumeCount
		itemRewards := make([]*Reward, 0)

		// Go through inventory items and consume as needed
		for instanceID, invItem := range inventory.Items {
			if invItem.Id == itemID && remainingToConsume > 0 {
				// Determine how many to consume from this instance
				toConsume := invItem.Count
				if toConsume > remainingToConsume {
					toConsume = remainingToConsume
				}

				// Update inventory
				invItem.Count -= toConsume
				remainingToConsume -= toConsume
				hasChanges = true

				// Remove item if count is zero and not marked to keep zero count items
				if invItem.Count <= 0 && !itemDef.KeepZero {
					delete(inventory.Items, instanceID)
				}

				// Process consume reward if defined
				if itemDef.ConsumeReward != nil {
					reward := &Reward{
						GrantTimeSec: nk.Time(),
						Items:        make(map[string]int64),
						Currencies:   make(map[string]int64),
						Energies:     make(map[string]int32),
					}

					// Call onConsumeReward if set
					if is.onConsumeReward != nil {
						reward, err = is.onConsumeReward(ctx, logger, nk, userID, itemID, itemDef, itemDef.ConsumeReward, reward)
						if err != nil {
							logger.Error("Error in onConsumeReward callback: %v", err)
						}
					}

					// Add reward to list
					itemRewards = append(itemRewards, reward)
				}
			}
		}

		// Store rewards if any were generated
		if len(itemRewards) > 0 {
			rewards[itemID] = itemRewards
		}
	}

	// Process consumption by instanceID
	for instanceID, consumeCount := range instanceIDs {
		// Skip invalid counts
		if consumeCount <= 0 {
			continue
		}

		// Check if instance exists
		invItem, exists := inventory.Items[instanceID]
		if !exists {
			logger.Warn("Attempted to consume unknown item instance: %s", instanceID)
			continue
		}

		// Get item definition
		itemDef, exists := itemDefs[invItem.Id]
		if !exists {
			logger.Warn("Item definition not found for: %s", invItem.Id)
			continue
		}

		// Check if consumable
		if !itemDef.Consumable {
			logger.Warn("Attempted to consume non-consumable item: %s", invItem.Id)
			continue
		}

		// Check if we have enough
		if invItem.Count < consumeCount && !overConsume {
			logger.Warn("Not enough items to consume. Instance: %s, Have: %d, Need: %d", instanceID, invItem.Count, consumeCount)
			continue
		}

		// Update inventory
		toConsume := consumeCount
		if toConsume > invItem.Count {
			toConsume = invItem.Count
		}
		invItem.Count -= toConsume
		hasChanges = true

		// Remove item if count is zero and not marked to keep zero count items
		if invItem.Count <= 0 && !itemDef.KeepZero {
			delete(inventory.Items, instanceID)
		}

		// Process consume reward if defined
		if itemDef.ConsumeReward != nil {
			reward := &Reward{
				GrantTimeSec: nk.Time(),
				Items:        make(map[string]int64),
				Currencies:   make(map[string]int64),
				Energies:     make(map[string]int32),
			}

			// Call onConsumeReward if set
			if is.onConsumeReward != nil {
				reward, err = is.onConsumeReward(ctx, logger, nk, userID, invItem.Id, itemDef, itemDef.ConsumeReward, reward)
				if err != nil {
					logger.Error("Error in onConsumeReward callback: %v", err)
				}
			}

			// Add reward to instance rewards
			instanceRewards[instanceID] = []*Reward{reward}
		}
	}

	// Save updated inventory if changes were made
	if hasChanges {
		// Serialize inventory
		inventoryBytes, err := nk.JSON.Marshal(inventory)
		if err != nil {
			logger.Error("Error marshaling inventory: %v", err)
			return nil, nil, nil, ErrInternal
		}

		// Write to storage
		objects := []*runtime.StorageWrite{
			{
				Collection:      "inventory",
				Key:             "user_items",
				UserID:          userID,
				Value:           inventoryBytes,
				PermissionRead:  1, // Only owner can read
				PermissionWrite: 1, // Only owner can write
			},
		}

		if _, err := nk.StorageWrite(ctx, objects); err != nil {
			logger.Error("Error writing inventory to storage: %v", err)
			return nil, nil, nil, ErrInternal
		}
	}

	return inventory, rewards, instanceRewards, nil
}

// GrantItems will add the item(s) to a user's inventory by ID.
func (is *InventorySystemImpl) GrantItems(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, itemIDs map[string]int64, ignoreLimits bool) (updatedInventory *Inventory, newItems map[string]*InventoryItem, updatedItems map[string]*InventoryItem, notGrantedItemIDs map[string]int64, err error) {
	// Get current inventory
	inventory, err := is.ListInventoryItems(ctx, logger, nk, userID, "")
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Get item definitions
	itemDefs, _, err := is.List(ctx, logger, nk, userID, "")
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Initialize result maps
	newItems = make(map[string]*InventoryItem)
	updatedItems = make(map[string]*InventoryItem)
	notGrantedItemIDs = make(map[string]int64)

	// Check if we need to apply limits
	if !ignoreLimits && is.config.Limits != nil {
		// Track counts by category and item set
		categoryCounts := make(map[string]int64)
		itemSetCounts := make(map[string]int64)

		// Count existing items
		for _, item := range inventory.Items {
			if itemDef, ok := itemDefs[item.Id]; ok {
				// Add to category counts
				if itemDef.Category != "" {
					categoryCounts[itemDef.Category] += item.Count
				}

				// Add to item set counts
				for _, setName := range itemDef.ItemSets {
					itemSetCounts[setName] += item.Count
				}
			}
		}

		// Check limits for items to be granted
		for itemID, count := range itemIDs {
			itemDef, exists := itemDefs[itemID]
			if !exists {
				logger.Warn("Item definition not found for: %s", itemID)
				notGrantedItemIDs[itemID] = count
				continue
			}

			// Check max count limit for the item
			if itemDef.MaxCount > 0 {
				// Count how many of this item we have
				currentCount := int64(0)
				for _, item := range inventory.Items {
					if item.Id == itemID {
						currentCount += item.Count
					}
				}

				// Check if granting would exceed limit
				if currentCount+count > itemDef.MaxCount {
					// Grant only up to the limit
					grantable := itemDef.MaxCount - currentCount
					if grantable <= 0 {
						notGrantedItemIDs[itemID] = count
						continue
					}
					notGrantedItemIDs[itemID] = count - grantable
					count = grantable
				}
			}

			// Check category limits
			if itemDef.Category != "" && is.config.Limits != nil && is.config.Limits.Categories != nil {
				if maxCategory, ok := is.config.Limits.Categories[itemDef.Category]; ok {
					if categoryCounts[itemDef.Category]+count > maxCategory {
						grantable := maxCategory - categoryCounts[itemDef.Category]
						if grantable <= 0 {
							notGrantedItemIDs[itemID] = count
							continue
						}
						notGrantedItemIDs[itemID] = count - grantable
						count = grantable
					}
				}
			}

			// Check item set limits
			if len(itemDef.ItemSets) > 0 && is.config.Limits != nil && is.config.Limits.ItemSets != nil {
				for _, setName := range itemDef.ItemSets {
					if maxSet, ok := is.config.Limits.ItemSets[setName]; ok {
						if itemSetCounts[setName]+count > maxSet {
							grantable := maxSet - itemSetCounts[setName]
							if grantable <= 0 {
								notGrantedItemIDs[itemID] = count
								count = 0
								break
							}
							notGrantedItemIDs[itemID] = count - grantable
							count = grantable
						}
					}
				}

				// If count was reduced to 0, skip this item
				if count <= 0 {
					continue
				}
			}

			// Update tracking counts for subsequent items
			if itemDef.Category != "" {
				categoryCounts[itemDef.Category] += count
			}

			for _, setName := range itemDef.ItemSets {
				itemSetCounts[setName] += count
			}

			// Adjust itemIDs for the actual amount to grant
			itemIDs[itemID] = count
		}
	}

	// Initialize inventory if it doesn't exist
	if inventory.Items == nil {
		inventory.Items = make(map[string]*InventoryItem)
	}

	// Process each item to grant
	for itemID, count := range itemIDs {
		// Skip if count is zero
		if count <= 0 {
			continue
		}

		itemDef, exists := itemDefs[itemID]
		if !exists {
			logger.Warn("Item definition not found for: %s", itemID)
			notGrantedItemIDs[itemID] = count
			continue
		}

		// Handle stackable vs non-stackable items
		if itemDef.Stackable {
			// Look for existing stack
			found := false
			for instanceID, item := range inventory.Items {
				if item.Id == itemID {
					// Update existing item
					item.Count += count
					updatedItems[instanceID] = item
					found = true
					break
				}
			}

			// Create new item if not found
			if !found {
				instanceID := nk.UuidV4()
				newItem := &InventoryItem{
					Id:                itemID,
					Count:             count,
					InstanceId:        instanceID,
					StringProperties:  make(map[string]string),
					NumericProperties: make(map[string]float64),
				}

				// Copy default properties from definition
				if itemDef.StringProperties != nil {
					for k, v := range itemDef.StringProperties {
						newItem.StringProperties[k] = v
					}
				}

				if itemDef.NumericProperties != nil {
					for k, v := range itemDef.NumericProperties {
						newItem.NumericProperties[k] = v
					}
				}

				inventory.Items[instanceID] = newItem
				newItems[instanceID] = newItem
			}
		} else {
			// Non-stackable items always create new instances
			for i := int64(0); i < count; i++ {
				instanceID := nk.UuidV4()
				newItem := &InventoryItem{
					Id:                itemID,
					Count:             1,
					InstanceId:        instanceID,
					StringProperties:  make(map[string]string),
					NumericProperties: make(map[string]float64),
				}

				// Copy default properties from definition
				if itemDef.StringProperties != nil {
					for k, v := range itemDef.StringProperties {
						newItem.StringProperties[k] = v
					}
				}

				if itemDef.NumericProperties != nil {
					for k, v := range itemDef.NumericProperties {
						newItem.NumericProperties[k] = v
					}
				}

				inventory.Items[instanceID] = newItem
				newItems[instanceID] = newItem
			}
		}
	}

	// Save updated inventory
	inventoryBytes, err := nk.JSON.Marshal(inventory)
	if err != nil {
		logger.Error("Error marshaling inventory: %v", err)
		return nil, nil, nil, nil, ErrInternal
	}

	// Write to storage
	objects := []*runtime.StorageWrite{
		{
			Collection:      "inventory",
			Key:             "user_items",
			UserID:          userID,
			Value:           inventoryBytes,
			PermissionRead:  1, // Only owner can read
			PermissionWrite: 1, // Only owner can write
		},
	}

	if _, err := nk.StorageWrite(ctx, objects); err != nil {
		logger.Error("Error writing inventory to storage: %v", err)
		return nil, nil, nil, nil, ErrInternal
	}

	return inventory, newItems, updatedItems, notGrantedItemIDs, nil
}

// UpdateItems will update the properties which are stored on each item by instance ID for a user.
func (is *InventorySystemImpl) UpdateItems(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, instanceIDs map[string]*InventoryUpdateItemProperties) (updatedInventory *Inventory, err error) {
	// Get current inventory
	inventory, err := is.ListInventoryItems(ctx, logger, nk, userID, "")
	if err != nil {
		return nil, err
	}

	// Check if we have any items to update
	if len(instanceIDs) == 0 {
		return inventory, nil
	}

	// Initialize inventory if it doesn't exist
	if inventory.Items == nil {
		// No items to update
		return inventory, nil
	}

	// Track if any changes were made
	hasChanges := false

	// Process each item update
	for instanceID, properties := range instanceIDs {
		// Check if instance exists
		item, exists := inventory.Items[instanceID]
		if !exists {
			logger.Warn("Attempted to update non-existent item instance: %s", instanceID)
			continue
		}

		// Update string properties
		if properties.StringProperties != nil {
			if item.StringProperties == nil {
				item.StringProperties = make(map[string]string)
			}

			for k, v := range properties.StringProperties {
				if v.Value != nil {
					item.StringProperties[k] = *v.Value
					hasChanges = true
				}
			}
		}

		// Update numeric properties
		if properties.NumericProperties != nil {
			if item.NumericProperties == nil {
				item.NumericProperties = make(map[string]float64)
			}

			for k, v := range properties.NumericProperties {
				if v.Value != nil {
					item.NumericProperties[k] = *v.Value
					hasChanges = true
				}
			}
		}
	}

	// Save updated inventory if changes were made
	if hasChanges {
		// Serialize inventory
		inventoryBytes, err := nk.JSON.Marshal(inventory)
		if err != nil {
			logger.Error("Error marshaling inventory: %v", err)
			return nil, ErrInternal
		}

		// Write to storage
		objects := []*runtime.StorageWrite{
			{
				Collection:      "inventory",
				Key:             "user_items",
				UserID:          userID,
				Value:           inventoryBytes,
				PermissionRead:  1, // Only owner can read
				PermissionWrite: 1, // Only owner can write
			},
		}

		if _, err := nk.StorageWrite(ctx, objects); err != nil {
			logger.Error("Error writing inventory to storage: %v", err)
			return nil, ErrInternal
		}
	}

	return inventory, nil
}

// SetOnConsumeReward sets a custom reward function which will run after an inventory items' consume reward is rolled.
func (is *InventorySystemImpl) SetOnConsumeReward(fn OnReward[*InventoryConfigItem]) {
	is.onConsumeReward = fn
}

// SetConfigSource sets a custom additional config lookup function.
func (is *InventorySystemImpl) SetConfigSource(fn ConfigSource[*InventoryConfigItem]) {
	is.configSource = fn
}

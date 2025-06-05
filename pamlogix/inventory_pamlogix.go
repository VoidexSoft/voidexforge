package pamlogix

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/heroiclabs/nakama-common/runtime"
)

const (
	inventoryStorageCollection = "inventory"
	// Configuration constants for pagination and memory management
	defaultInventoryPageSize = 100
	maxInventoryPageSize     = 1000 // Hard limit to prevent excessive memory usage
)

// InventoryLoadOptions provides configuration for how inventory data should be loaded
type InventoryLoadOptions struct {
	// PageSize determines how many items to load per page (default: 100, max: 1000)
	PageSize int
	// MaxItems limits the total number of items to load (0 = no limit)
	MaxItems int
	// LoadAllPages determines if all pages should be loaded (for complete inventory)
	LoadAllPages bool
	// SpecificItems is a list of item IDs or instance IDs to load specifically
	SpecificItems []string
	// Category filters items by category
	Category string
}

// NakamaInventorySystem implements the InventorySystem interface using Nakama as the backend.
type NakamaInventorySystem struct {
	config          *InventoryConfig
	onConsumeReward OnReward[*InventoryConfigItem]
	configSource    ConfigSource[*InventoryConfigItem]
	pamlogix        Pamlogix
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

// SetPamlogix sets the Pamlogix instance for this inventory system
func (i *NakamaInventorySystem) SetPamlogix(pl Pamlogix) {
	i.pamlogix = pl
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

	// Use efficient loading with category filter
	loadOptions := &InventoryLoadOptions{
		PageSize:     defaultInventoryPageSize,
		LoadAllPages: true, // For list operations, we typically want all items
		Category:     category,
	}

	userInventory, err := i.getUserInventoryWithOptions(ctx, logger, nk, userID, loadOptions)
	if err != nil {
		logger.Error("Failed to get user inventory: %v", err)
		return nil, ErrInternal
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

	// For item ID-based consumption, we need to load all inventory items to find all instances
	var loadOptions *InventoryLoadOptions
	if len(itemIDs) > 0 {
		// Load all items to find all instances with matching item IDs
		loadOptions = &InventoryLoadOptions{
			PageSize:     defaultInventoryPageSize,
			LoadAllPages: true, // Need all items to find all instances with matching item IDs
		}
	} else {
		// Only instance-based consumption, load specific items
		specificItems := make([]string, 0, len(instanceIDs))
		for instanceID := range instanceIDs {
			specificItems = append(specificItems, instanceID)
		}
		loadOptions = &InventoryLoadOptions{
			PageSize:      defaultInventoryPageSize,
			SpecificItems: specificItems,
			LoadAllPages:  false, // Only load what we need for consumption
		}
	}

	userInventory, err := i.getUserInventoryWithOptions(ctx, logger, nk, userID, loadOptions)
	if err != nil {
		logger.Error("Failed to get user inventory: %v", err)
		return nil, nil, nil, ErrInternal
	}

	// Prepare storage operations
	var storageWrites []*runtime.StorageWrite
	var storageDeletes []*runtime.StorageDelete

	// Process item ID-based consumption
	for itemID, count := range itemIDs {
		// Check if item exists in configuration
		configItem, exists := i.config.Items[itemID]
		if !exists {
			logger.Warn("Attempted to consume non-existent item: %s", itemID)
			return nil, nil, nil, ErrBadInput
		}

		// Check if item is consumable
		if !configItem.Consumable {
			logger.Warn("Attempted to consume non-consumable item: %s", itemID)
			return nil, nil, nil, ErrBadInput
		}

		// Find all items in user's inventory with this item ID
		var matchingItems []*struct {
			key  string
			item *InventoryItem
		}
		totalAvailable := int64(0)

		for key, item := range userInventory.Items {
			if item.Id == itemID {
				matchingItems = append(matchingItems, &struct {
					key  string
					item *InventoryItem
				}{key: key, item: item})
				totalAvailable += item.Count
			}
		}

		if len(matchingItems) == 0 {
			logger.Warn("Attempted to consume item user doesn't have: %s", itemID)
			return nil, nil, nil, ErrBadInput
		}

		// Check if user has enough items across all instances
		if totalAvailable < count && !overConsume {
			logger.Warn("Insufficient items to consume: %s (have: %d, need: %d)", itemID, totalAvailable, count)
			return nil, nil, nil, ErrBadInput
		}

		// Consume items across all instances until the required count is met
		remainingToConsume := count
		for _, matchingItem := range matchingItems {
			if remainingToConsume <= 0 {
				break
			}

			inventoryKey := matchingItem.key
			inventoryItem := matchingItem.item

			// Calculate how much to consume from this instance
			consumeFromThis := remainingToConsume
			if consumeFromThis > inventoryItem.Count {
				consumeFromThis = inventoryItem.Count
			}

			// Consume the items from this instance
			inventoryItem.Count -= consumeFromThis
			inventoryItem.UpdateTimeSec = time.Now().Unix()
			remainingToConsume -= consumeFromThis

			// Remove item if count reaches 0 and keep_zero is false
			if inventoryItem.Count <= 0 && !configItem.KeepZero {
				delete(userInventory.Items, inventoryKey)

				// Determine storage key for deletion - use instance ID if available, otherwise item ID
				storageKey := itemID
				if inventoryItem.InstanceId != "" {
					storageKey = inventoryItem.InstanceId
				}

				// Add delete operation - use proper StorageDelete
				storageDeletes = append(storageDeletes, &runtime.StorageDelete{
					Collection: inventoryStorageCollection,
					Key:        storageKey,
					UserID:     userID,
				})
			} else {
				// Update item in storage
				itemData, err := json.Marshal(inventoryItem)
				if err != nil {
					logger.Error("Failed to marshal inventory item: %v", err)
					return nil, nil, nil, ErrInternal
				}

				// Determine storage key for update - use instance ID if available, otherwise item ID
				storageKey := itemID
				if inventoryItem.InstanceId != "" {
					storageKey = inventoryItem.InstanceId
				}

				storageWrites = append(storageWrites, &runtime.StorageWrite{
					Collection:      inventoryStorageCollection,
					Key:             storageKey,
					UserID:          userID,
					Value:           string(itemData),
					PermissionRead:  runtime.STORAGE_PERMISSION_OWNER_READ,
					PermissionWrite: runtime.STORAGE_PERMISSION_OWNER_WRITE,
				})
			}
		}

		//i.processAndStoreRewards(ctx, logger, nk, userID, itemID, itemID, configItem, count, rewards)
		reward, err := i.processItemReward(ctx, logger, nk, userID, itemID, configItem)
		if err != nil {
			logger.Error("Failed to process item reward: %v", err)
			return nil, nil, nil, ErrInternal
		}

		//append the reward to the rewards map
		if reward != nil {
			if _, exists := rewards[itemID]; !exists {
				rewards[itemID] = make([]*Reward, 0)
			}
			rewards[itemID] = append(rewards[itemID], reward)
		}
	}

	// Process instance ID-based consumption
	for instanceID, count := range instanceIDs {
		// The inventory's key is the instance ID, so we can directly access it
		foundItem := userInventory.Items[instanceID]
		inventoryKey := instanceID

		if foundItem == nil {
			logger.Warn("Attempted to consume non-existent instance: %s", instanceID)
			return nil, nil, nil, ErrBadInput
		}

		// Check if item exists in configuration
		configItem, exists := i.config.Items[foundItem.Id]
		if !exists {
			logger.Warn("Attempted to consume item with unknown config: %s", foundItem.Id)
			return nil, nil, nil, ErrBadInput
		}

		// Check if item is consumable
		if !configItem.Consumable {
			logger.Warn("Attempted to consume non-consumable item instance: %s", instanceID)
			return nil, nil, nil, ErrBadInput
		}

		// Check if user has enough items
		if foundItem.Count < count && !overConsume {
			logger.Warn("Insufficient items to consume for instance: %s (have: %d, need: %d)", instanceID, foundItem.Count, count)
			return nil, nil, nil, ErrBadInput
		}

		// Consume the items
		foundItem.Count -= count
		foundItem.UpdateTimeSec = time.Now().Unix()

		// Remove item if count reaches 0 and keep_zero is false
		if foundItem.Count <= 0 && !configItem.KeepZero {
			delete(userInventory.Items, inventoryKey)

			// For instance-based operations, the storage key is always the instance ID
			storageDeletes = append(storageDeletes, &runtime.StorageDelete{
				Collection: inventoryStorageCollection,
				Key:        instanceID,
				UserID:     userID,
			})
		} else {
			// Update item in storage
			itemData, err := json.Marshal(foundItem)
			if err != nil {
				logger.Error("Failed to marshal inventory item: %v", err)
				return nil, nil, nil, ErrInternal
			}

			// For instance-based operations, the storage key is always the instance ID
			storageWrites = append(storageWrites, &runtime.StorageWrite{
				Collection:      inventoryStorageCollection,
				Key:             instanceID,
				UserID:          userID,
				Value:           string(itemData),
				PermissionRead:  runtime.STORAGE_PERMISSION_OWNER_READ,
				PermissionWrite: runtime.STORAGE_PERMISSION_OWNER_WRITE,
			})
		}

		//i.processAndStoreRewards(ctx, logger, nk, userID, foundItem.Id, instanceID, configItem, count, instanceRewards)

		reward, err := i.processItemReward(ctx, logger, nk, userID, foundItem.Id, configItem)
		if err != nil {
			logger.Error("Failed to process item reward for instance: %v", err)
			return nil, nil, nil, ErrInternal
		}
		if reward != nil {
			if _, exists := instanceRewards[instanceID]; !exists {
				instanceRewards[instanceID] = make([]*Reward, 0)
			}
			instanceRewards[instanceID] = append(instanceRewards[instanceID], reward)
		}
	}

	// Write changes to storage if there are any
	if len(storageWrites) > 0 {
		_, err = nk.StorageWrite(ctx, storageWrites)
		if err != nil {
			logger.Error("Failed to write inventory updates: %v", err)
			return nil, nil, nil, ErrInternal
		}
	}

	// Delete items from storage if there are any
	if len(storageDeletes) > 0 {
		err = nk.StorageDelete(ctx, storageDeletes)
		if err != nil {
			logger.Error("Failed to delete inventory items: %v", err)
			return nil, nil, nil, ErrInternal
		}
	}

	return userInventory, rewards, instanceRewards, nil
}

// GrantItems will add the item(s) to a user's inventory by ID.
func (i *NakamaInventorySystem) GrantItems(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, itemIDs map[string]int64, ignoreLimits bool) (updatedInventory *Inventory, newItems map[string]*InventoryItem, updatedItems map[string]*InventoryItem, notGrantedItemIDs map[string]int64, err error) {
	if i.config == nil || len(i.config.Items) == 0 {
		// No items are configured
		return &Inventory{Items: make(map[string]*InventoryItem)}, make(map[string]*InventoryItem), make(map[string]*InventoryItem), make(map[string]int64), nil
	}

	// Initialize return values
	newItems = make(map[string]*InventoryItem)
	updatedItems = make(map[string]*InventoryItem)
	notGrantedItemIDs = make(map[string]int64)

	loadOptions := &InventoryLoadOptions{
		PageSize:     defaultInventoryPageSize,
		LoadAllPages: true,
	}

	userInventory, err := i.getUserInventoryWithOptions(ctx, logger, nk, userID, loadOptions)
	if err != nil {
		logger.Error("Failed to get user inventory: %v", err)
		return nil, nil, nil, nil, ErrInternal
	}

	// Process each item to grant
	var storageOps []*runtime.StorageWrite
	now := time.Now().Unix()

	for itemID, count := range itemIDs {
		// Check if item exists in configuration
		configItem, exists := i.config.Items[itemID]
		if !exists {
			logger.Warn("Attempted to grant non-existent item: %s", itemID)
			notGrantedItemIDs[itemID] = count
			continue
		}

		// Check limits if not ignoring them
		if !ignoreLimits {
			// Check category limits
			if i.config.Limits != nil && i.config.Limits.Categories != nil {
				if categoryLimit, exists := i.config.Limits.Categories[configItem.Category]; exists && categoryLimit > 0 {
					// Count items in this category
					categoryCount := int64(0)
					for _, item := range userInventory.Items {
						if item.Category == configItem.Category {
							categoryCount += item.Count
						}
					}
					if categoryCount+count > categoryLimit {
						logger.Info("Category limit exceeded for item %s (category: %s, limit: %d)", itemID, configItem.Category, categoryLimit)
						notGrantedItemIDs[itemID] = count
						continue
					}
				}
			}

			// Check item set limits
			if i.config.Limits != nil && i.config.Limits.ItemSets != nil {
				limitExceeded := false
				for _, setID := range configItem.ItemSets {
					if setLimit, exists := i.config.Limits.ItemSets[setID]; exists && setLimit > 0 {
						// Count items in this set
						setCount := int64(0)
						for _, item := range userInventory.Items {
							for _, itemSetID := range item.ItemSets {
								if itemSetID == setID {
									setCount += item.Count
									break
								}
							}
						}
						if setCount+count > setLimit {
							logger.Info("Item set limit exceeded for item %s (set: %s, limit: %d)", itemID, setID, setLimit)
							notGrantedItemIDs[itemID] = count
							limitExceeded = true
							break
						}
					}
				}
				if limitExceeded {
					continue
				}
			}

			// Check individual item limits
			if configItem.MaxCount > 0 {
				currentCount := int64(0)
				for _, item := range userInventory.Items {
					if item.Id == itemID {
						currentCount += item.Count
					}
				}
				if currentCount+count > configItem.MaxCount {
					logger.Info("Item limit exceeded for item %s (limit: %d)", itemID, configItem.MaxCount)
					notGrantedItemIDs[itemID] = count
					continue
				}
			}
		}

		// Check if the item is stackable and user already has it
		var existingItem *InventoryItem
		var existingKey string
		if configItem.Stackable {
			for key, item := range userInventory.Items {
				if item.Id == itemID {
					existingItem = item
					existingKey = key
					break
				}
			}
		}

		if existingItem != nil {
			// Update existing stackable item
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

			updatedItems[existingKey] = existingItem

			// Prepare storage update
			itemData, err := json.Marshal(existingItem)
			if err != nil {
				logger.Error("Failed to marshal inventory item: %v", err)
				return nil, nil, nil, nil, ErrInternal
			}

			// Determine storage key: use instance ID if it exists, otherwise fall back to itemID
			storageKey := existingKey
			if existingItem.InstanceId != "" {
				storageKey = existingItem.InstanceId
			}

			storageOps = append(storageOps, &runtime.StorageWrite{
				Collection:      inventoryStorageCollection,
				Key:             storageKey,
				UserID:          userID,
				Value:           string(itemData),
				PermissionRead:  runtime.STORAGE_PERMISSION_OWNER_READ,
				PermissionWrite: runtime.STORAGE_PERMISSION_OWNER_WRITE,
			})
		} else {
			// Create new item(s)
			// For non-stackable items with count > 1, create multiple instances
			if !configItem.Stackable && count > 1 {
				for instanceNum := int64(0); instanceNum < count; instanceNum++ {
					newItem := &InventoryItem{
						Id:                itemID,
						Name:              configItem.Name,
						Description:       configItem.Description,
						Category:          configItem.Category,
						ItemSets:          configItem.ItemSets,
						Count:             1, // Each instance has count = 1 for non-stackable items
						MaxCount:          configItem.MaxCount,
						Stackable:         configItem.Stackable,
						Consumable:        configItem.Consumable,
						OwnedTimeSec:      now,
						UpdateTimeSec:     now,
						StringProperties:  configItem.StringProperties,
						NumericProperties: configItem.NumericProperties,
					}

					newItem.InstanceId = uuid.New().String()

					inventoryKey := newItem.InstanceId
					storageKey := newItem.InstanceId

					userInventory.Items[inventoryKey] = newItem
					newItems[inventoryKey] = newItem

					// Prepare storage update
					itemData, err := json.Marshal(newItem)
					if err != nil {
						logger.Error("Failed to marshal new inventory item: %v", err)
						return nil, nil, nil, nil, ErrInternal
					}

					storageOps = append(storageOps, &runtime.StorageWrite{
						Collection:      inventoryStorageCollection,
						Key:             storageKey,
						UserID:          userID,
						Value:           string(itemData),
						PermissionRead:  runtime.STORAGE_PERMISSION_OWNER_READ,
						PermissionWrite: runtime.STORAGE_PERMISSION_OWNER_WRITE,
					})
				}
			} else {
				// Create single item (for stackable items or count = 1)
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

				newItem.InstanceId = uuid.New().String()

				inventoryKey := newItem.InstanceId
				storageKey := newItem.InstanceId

				userInventory.Items[inventoryKey] = newItem
				newItems[inventoryKey] = newItem

				// Prepare storage update
				itemData, err := json.Marshal(newItem)
				if err != nil {
					logger.Error("Failed to marshal new inventory item: %v", err)
					return nil, nil, nil, nil, ErrInternal
				}

				storageOps = append(storageOps, &runtime.StorageWrite{
					Collection:      inventoryStorageCollection,
					Key:             storageKey,
					UserID:          userID,
					Value:           string(itemData),
					PermissionRead:  runtime.STORAGE_PERMISSION_OWNER_READ,
					PermissionWrite: runtime.STORAGE_PERMISSION_OWNER_WRITE,
				})
			}
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

	if len(instanceIDs) == 0 {
		// No updates requested, return empty inventory
		return &Inventory{Items: make(map[string]*InventoryItem)}, nil
	}

	logger.Info("Updating %d inventory items for user %s", len(instanceIDs), userID)

	// Collect specific instance IDs we need to load
	specificItems := make([]string, 0, len(instanceIDs))
	for instanceID := range instanceIDs {
		specificItems = append(specificItems, instanceID)
	}

	// Load only the items we need to operate on
	loadOptions := &InventoryLoadOptions{
		PageSize:      defaultInventoryPageSize,
		SpecificItems: specificItems,
		LoadAllPages:  false, // Only load what we need for updating
	}

	userInventory, err := i.getUserInventoryWithOptions(ctx, logger, nk, userID, loadOptions)
	if err != nil {
		logger.Error("Failed to get user inventory: %v", err)
		return nil, ErrInternal
	}

	// Pre-allocate storage operations slice with known capacity
	storageOps := make([]*runtime.StorageWrite, 0, len(instanceIDs))

	instanceToKey := make(map[string]string, len(userInventory.Items))
	for key, item := range userInventory.Items {
		if item.InstanceId != "" {
			instanceToKey[item.InstanceId] = key
		}
	}

	// Track failed updates for better error reporting
	var failedUpdates []string
	updateCount := 0

	for instanceID, props := range instanceIDs {
		// Use efficient lookup
		inventoryKey, exists := instanceToKey[instanceID]
		if !exists {
			logger.Warn("Attempted to update non-existent instance: %s", instanceID)
			failedUpdates = append(failedUpdates, instanceID)
			continue
		}

		foundItem := userInventory.Items[inventoryKey]
		if foundItem == nil {
			logger.Warn("Item not found in inventory for instance: %s", instanceID)
			failedUpdates = append(failedUpdates, instanceID)
			continue
		}

		// Validate that we can update this item type
		configItem, configExists := i.config.Items[foundItem.Id]
		if configExists && configItem.Disabled {
			logger.Warn("Attempted to update disabled item: %s (instance: %s)", foundItem.Id, instanceID)
			failedUpdates = append(failedUpdates, instanceID)
			continue
		}

		// Track if any changes were made
		itemModified := false

		// Update string properties
		if props.StringProperties != nil && len(props.StringProperties) > 0 {
			if foundItem.StringProperties == nil {
				foundItem.StringProperties = make(map[string]string)
			}
			for key, value := range props.StringProperties {
				// Log property updates for audit trail
				if existingValue, exists := foundItem.StringProperties[key]; exists {
					logger.Debug("Updating string property %s from '%s' to '%s' for instance %s", key, existingValue, value, instanceID)
				} else {
					logger.Debug("Adding string property %s='%s' for instance %s", key, value, instanceID)
				}
				foundItem.StringProperties[key] = value
				itemModified = true
			}
		}

		// Update numeric properties
		if props.NumericProperties != nil && len(props.NumericProperties) > 0 {
			if foundItem.NumericProperties == nil {
				foundItem.NumericProperties = make(map[string]float64)
			}
			for key, value := range props.NumericProperties {
				// Log property updates for audit trail
				if existingValue, exists := foundItem.NumericProperties[key]; exists {
					logger.Debug("Updating numeric property %s from %f to %f for instance %s", key, existingValue, value, instanceID)
				} else {
					logger.Debug("Adding numeric property %s=%f for instance %s", key, value, instanceID)
				}
				foundItem.NumericProperties[key] = value
				itemModified = true
			}
		}

		// Only update timestamp and prepare storage if item was actually modified
		if itemModified {
			// Update timestamp
			foundItem.UpdateTimeSec = time.Now().Unix()

			// Prepare storage update
			itemData, err := json.Marshal(foundItem)
			if err != nil {
				logger.Error("Failed to marshal inventory item %s: %v", instanceID, err)
				failedUpdates = append(failedUpdates, instanceID)
				continue
			}

			// For instance-based operations, the storage key is always the instance ID
			storageKey := instanceID

			storageOps = append(storageOps, &runtime.StorageWrite{
				Collection:      inventoryStorageCollection,
				Key:             storageKey,
				UserID:          userID,
				Value:           string(itemData),
				PermissionRead:  runtime.STORAGE_PERMISSION_OWNER_READ,
				PermissionWrite: runtime.STORAGE_PERMISSION_OWNER_WRITE,
			})
			updateCount++
		} else {
			logger.Debug("No properties to update for instance %s", instanceID)
		}
	}

	// Write changes to storage if there are any
	if len(storageOps) > 0 {
		_, err = nk.StorageWrite(ctx, storageOps)
		if err != nil {
			logger.Error("Failed to update inventory storage: %v", err)
			return nil, ErrInternal
		}
		logger.Info("Successfully updated %d inventory items for user %s", updateCount, userID)
	} else {
		logger.Info("No inventory items required updates for user %s", userID)
	}

	// Log failed updates if any
	if len(failedUpdates) > 0 {
		logger.Warn("Failed to update %d items for user %s: %v", len(failedUpdates), userID, failedUpdates)
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

// Helper function to get a user's inventory from storage with improved pagination and memory management
func (i *NakamaInventorySystem) getUserInventoryWithOptions(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, options *InventoryLoadOptions) (*Inventory, error) {
	inventory := &Inventory{
		Items: make(map[string]*InventoryItem),
	}

	// Apply default options if not provided
	if options == nil {
		options = &InventoryLoadOptions{
			PageSize:     defaultInventoryPageSize,
			LoadAllPages: true,
		}
	}

	// Validate and set page size
	pageSize := options.PageSize
	if pageSize <= 0 {
		pageSize = defaultInventoryPageSize
	}
	if pageSize > maxInventoryPageSize {
		pageSize = maxInventoryPageSize
		logger.Warn("Page size capped at %d to prevent excessive memory usage", maxInventoryPageSize)
	}

	// If we have specific items to load, use targeted storage reads
	if len(options.SpecificItems) > 0 {
		return i.getUserInventorySpecificItems(ctx, logger, nk, userID, options.SpecificItems)
	}

	// Track total items loaded and cursor for pagination
	totalItemsLoaded := 0
	cursor := ""
	hasMore := true

	for hasMore {
		// Retrieve inventory objects from storage with pagination
		objects, nextCursor, err := nk.StorageList(ctx, "", userID, inventoryStorageCollection, pageSize, cursor)
		if err != nil {
			logger.Error("Failed to read inventory from storage: %v", err)
			return inventory, err
		}

		// Process each item in the current page
		for _, obj := range objects {
			// Check if we've hit the max items limit
			if options.MaxItems > 0 && totalItemsLoaded >= options.MaxItems {
				hasMore = false
				break
			}

			var item InventoryItem
			err = json.Unmarshal([]byte(obj.Value), &item)
			if err != nil {
				logger.Error("Failed to unmarshal inventory item: %v", err)
				continue
			}

			// Determine the item ID from the stored item data
			itemID := item.Id
			if itemID == "" {
				// Fallback to storage key for backward compatibility
				itemID = obj.Key
				item.Id = itemID
			}

			// Apply category filter if specified
			if options.Category != "" {
				configItem, exists := i.config.Items[itemID]
				if !exists || configItem.Category != options.Category {
					continue
				}
			}

			// Only include non-disabled items
			configItem, exists := i.config.Items[itemID]
			if exists && !configItem.Disabled {
				// Update from config if needed
				item.Name = configItem.Name
				item.Description = configItem.Description
				item.Category = configItem.Category
				item.ItemSets = configItem.ItemSets
				item.MaxCount = configItem.MaxCount
				item.Stackable = configItem.Stackable
				item.Consumable = configItem.Consumable

				// Use instance ID as key if available, otherwise use storage key
				key := obj.Key
				if item.InstanceId != "" {
					key = item.InstanceId
				}

				// Add to inventory using appropriate key
				inventory.Items[key] = &item
				totalItemsLoaded++
			}
		}

		// Check if we should continue pagination
		if nextCursor == "" || !options.LoadAllPages {
			hasMore = false
		} else {
			cursor = nextCursor
		}

		// If we got fewer items than requested, we've reached the end
		if len(objects) < pageSize {
			hasMore = false
		}
	}

	logger.Debug("Loaded %d inventory items for user %s", totalItemsLoaded, userID)
	return inventory, nil
}

// Helper function to load specific inventory items by ID/instance ID
func (i *NakamaInventorySystem) getUserInventorySpecificItems(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, itemKeys []string) (*Inventory, error) {
	inventory := &Inventory{
		Items: make(map[string]*InventoryItem),
	}

	if len(itemKeys) == 0 {
		return inventory, nil
	}

	// Prepare storage read operations for specific items
	var storageReadIDs []*runtime.StorageRead
	for _, key := range itemKeys {
		storageReadIDs = append(storageReadIDs, &runtime.StorageRead{
			Collection: inventoryStorageCollection,
			Key:        key,
			UserID:     userID,
		})
	}

	// Perform batch read operation
	objects, err := nk.StorageRead(ctx, storageReadIDs)
	if err != nil {
		logger.Error("Failed to read specific inventory items from storage: %v", err)
		return inventory, err
	}

	// Process each retrieved item
	for _, obj := range objects {
		if obj == nil {
			continue // Item not found in storage
		}

		var item InventoryItem
		err = json.Unmarshal([]byte(obj.Value), &item)
		if err != nil {
			logger.Error("Failed to unmarshal inventory item: %v", err)
			continue
		}

		// Determine the item ID from the stored item data
		itemID := item.Id
		if itemID == "" {
			// Fallback to storage key for backward compatibility
			itemID = obj.Key
			item.Id = itemID
		}

		// Only include non-disabled items
		configItem, exists := i.config.Items[itemID]
		if exists && !configItem.Disabled {
			// Update from config if needed
			item.Name = configItem.Name
			item.Description = configItem.Description
			item.Category = configItem.Category
			item.ItemSets = configItem.ItemSets
			item.MaxCount = configItem.MaxCount
			item.Stackable = configItem.Stackable
			item.Consumable = configItem.Consumable

			// Use instance ID as key if available, otherwise use storage key
			key := obj.Key
			if item.InstanceId != "" {
				key = item.InstanceId
			}

			// Add to inventory using appropriate key
			inventory.Items[key] = &item
		}
	}

	logger.Debug("Loaded %d specific inventory items for user %s", len(inventory.Items), userID)
	return inventory, nil
}

// Helper function to process an item's consume reward
func (i *NakamaInventorySystem) processItemReward(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, itemID string, configItem *InventoryConfigItem) (*Reward, error) {
	if configItem.ConsumeReward == nil {
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

	// Get the economy system from the Pamlogix instance to process the reward configuration
	var economySystem EconomySystem

	if i.pamlogix != nil {
		economySystem = i.pamlogix.GetEconomySystem()
	}

	if economySystem == nil {
		logger.Error("Economy system is not available in the system")
		return nil, ErrInternal
	}

	// Roll the reward using the reward configuration
	rolledReward, err := economySystem.RewardRoll(ctx, logger, nk, userID, configItem.ConsumeReward)
	if err != nil {
		logger.Error("Failed to roll reward for item consumption: %v", err)
		// Continue with empty reward
	} else if rolledReward != nil {
		_, _, _, grantErr := economySystem.RewardGrant(ctx, logger, nk, userID, reward, map[string]interface{}{
			"reason": "consume_item",
		}, false)
		if grantErr != nil {
			logger.Error("Failed to grant rolled reward for item consumption: %v", grantErr)
			return nil, grantErr
		}

		reward = rolledReward
	}

	// Apply custom reward handler if set
	if i.onConsumeReward != nil {
		customReward, err := i.onConsumeReward(ctx, logger, nk, userID, itemID, configItem, configItem.ConsumeReward, reward)
		if err != nil {
			logger.Error("Error in custom reward handler: %v", err)
			return nil, err
		}
		if customReward != nil {
			reward = customReward
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

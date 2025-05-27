package pamlogix

import (
	"testing"

	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations for testing
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(format string, v ...interface{}) {
	m.Called(format, v)
}

func (m *MockLogger) Info(format string, v ...interface{}) {
	m.Called(format, v)
}

func (m *MockLogger) Warn(format string, v ...interface{}) {
	m.Called(format, v)
}

func (m *MockLogger) Error(format string, v ...interface{}) {
	m.Called(format, v)
}

func (m *MockLogger) Fatal(format string, v ...interface{}) {
	m.Called(format, v)
}

func (m *MockLogger) WithField(key string, v interface{}) runtime.Logger {
	return m
}

func (m *MockLogger) WithFields(fields map[string]interface{}) runtime.Logger {
	return m
}

func (m *MockLogger) Fields() map[string]interface{} {
	return make(map[string]interface{})
}

type MockPamlogix struct {
	mock.Mock
	inventorySystem InventorySystem
}

func (m *MockPamlogix) GetInventorySystem() InventorySystem {
	return m.inventorySystem
}

func (m *MockPamlogix) GetEconomySystem() EconomySystem {
	args := m.Called()
	return args.Get(0).(EconomySystem)
}

func (m *MockPamlogix) GetEnergySystem() EnergySystem {
	args := m.Called()
	return args.Get(0).(EnergySystem)
}

func (m *MockPamlogix) GetAuctionsSystem() AuctionsSystem {
	args := m.Called()
	return args.Get(0).(AuctionsSystem)
}

func (m *MockPamlogix) GetAchievementsSystem() AchievementsSystem {
	args := m.Called()
	return args.Get(0).(AchievementsSystem)
}

func (m *MockPamlogix) AddPersonalizer(personalizer Personalizer) {
	m.Called(personalizer)
}

func (m *MockPamlogix) SetPersonalizer(personalizer Personalizer) {
	m.Called(personalizer)
}

func (m *MockPamlogix) AddPublisher(publisher Publisher) {
	m.Called(publisher)
}

func (m *MockPamlogix) SetAfterAuthenticate(fn AfterAuthenticateFn) {
	m.Called(fn)
}

func (m *MockPamlogix) SetCollectionResolver(fn CollectionResolverFn) {
	m.Called(fn)
}

func (m *MockPamlogix) GetBaseSystem() BaseSystem {
	args := m.Called()
	return args.Get(0).(BaseSystem)
}

func (m *MockPamlogix) GetLeaderboardsSystem() LeaderboardsSystem {
	args := m.Called()
	return args.Get(0).(LeaderboardsSystem)
}

func (m *MockPamlogix) GetStatsSystem() StatsSystem {
	args := m.Called()
	return args.Get(0).(StatsSystem)
}

func (m *MockPamlogix) GetTeamsSystem() TeamsSystem {
	args := m.Called()
	return args.Get(0).(TeamsSystem)
}

func (m *MockPamlogix) GetTutorialsSystem() TutorialsSystem {
	args := m.Called()
	return args.Get(0).(TutorialsSystem)
}

func (m *MockPamlogix) GetUnlockablesSystem() UnlockablesSystem {
	args := m.Called()
	return args.Get(0).(UnlockablesSystem)
}

func (m *MockPamlogix) GetEventLeaderboardsSystem() EventLeaderboardsSystem {
	args := m.Called()
	return args.Get(0).(EventLeaderboardsSystem)
}

func (m *MockPamlogix) GetProgressionSystem() ProgressionSystem {
	args := m.Called()
	return args.Get(0).(ProgressionSystem)
}

func (m *MockPamlogix) GetIncentivesSystem() IncentivesSystem {
	args := m.Called()
	return args.Get(0).(IncentivesSystem)
}

func (m *MockPamlogix) GetStreaksSystem() StreaksSystem {
	args := m.Called()
	return args.Get(0).(StreaksSystem)
}

func TestAuctionItemSetValidation(t *testing.T) {
	// Create inventory config with item sets
	inventoryConfig := &InventoryConfig{
		Items: map[string]*InventoryConfigItem{
			"sword_basic": {
				Name:     "Basic Sword",
				Category: "weapon",
				ItemSets: []string{"starter_equipment", "weapons"},
			},
			"sword_advanced": {
				Name:     "Advanced Sword",
				Category: "weapon",
				ItemSets: []string{"weapons"},
			},
			"health_potion": {
				Name:     "Health Potion",
				Category: "consumable",
				ItemSets: []string{"potions"},
			},
			"mana_potion": {
				Name:     "Mana Potion",
				Category: "consumable",
				ItemSets: []string{"potions"},
			},
			"rare_gem": {
				Name:     "Rare Gem",
				Category: "material",
				ItemSets: []string{"materials", "rare_items"},
			},
		},
	}

	// Create inventory system and let it pre-compute item sets
	inventorySystem := NewNakamaInventorySystem(inventoryConfig)

	// Create mock pamlogix
	mockPamlogix := &MockPamlogix{
		inventorySystem: inventorySystem,
	}

	// Create auction config with item set restrictions
	auctionConfig := &AuctionsConfigAuction{
		Items:    []string{"sword_basic"}, // Allow specific item
		ItemSets: []string{"potions"},     // Allow items from potions set
	}

	// Create auctions system
	auctionsSystem := &AuctionsPamlogix{
		config:   &AuctionsConfig{},
		pamlogix: mockPamlogix,
	}

	tests := []struct {
		name        string
		items       []*InventoryItem
		expectError bool
		description string
	}{
		{
			name: "Allow item from individual items list",
			items: []*InventoryItem{
				{Id: "sword_basic", Count: 1},
			},
			expectError: false,
			description: "sword_basic is explicitly allowed in the Items list",
		},
		{
			name: "Allow item from item set",
			items: []*InventoryItem{
				{Id: "health_potion", Count: 1},
			},
			expectError: false,
			description: "health_potion is in the potions set which is allowed",
		},
		{
			name: "Allow multiple items from item set",
			items: []*InventoryItem{
				{Id: "health_potion", Count: 1},
				{Id: "mana_potion", Count: 1},
			},
			expectError: false,
			description: "Both potions are in the allowed potions set",
		},
		{
			name: "Allow mix of individual item and item set",
			items: []*InventoryItem{
				{Id: "sword_basic", Count: 1},
				{Id: "health_potion", Count: 1},
			},
			expectError: false,
			description: "sword_basic is individually allowed, health_potion is from allowed set",
		},
		{
			name: "Reject item not in allowed lists or sets",
			items: []*InventoryItem{
				{Id: "sword_advanced", Count: 1},
			},
			expectError: true,
			description: "sword_advanced is not individually allowed and not in allowed sets",
		},
		{
			name: "Reject item from non-allowed set",
			items: []*InventoryItem{
				{Id: "rare_gem", Count: 1},
			},
			expectError: true,
			description: "rare_gem is in materials/rare_items sets which are not allowed",
		},
		{
			name: "Reject mix with invalid item",
			items: []*InventoryItem{
				{Id: "health_potion", Count: 1}, // Valid
				{Id: "rare_gem", Count: 1},      // Invalid
			},
			expectError: true,
			description: "One valid item (health_potion) and one invalid item (rare_gem)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auctionsSystem.validateItems(tt.items, auctionConfig)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.description)
				assert.Equal(t, ErrAuctionItemsInvalid, err, "Expected ErrAuctionItemsInvalid")
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.description)
			}
		})
	}
}

func TestAuctionItemSetValidationNoRestrictions(t *testing.T) {
	// Create inventory config
	inventoryConfig := &InventoryConfig{
		Items: map[string]*InventoryConfigItem{
			"any_item": {
				Name:     "Any Item",
				Category: "misc",
			},
		},
	}

	// Create inventory system
	inventorySystem := NewNakamaInventorySystem(inventoryConfig)

	// Create mock pamlogix
	mockPamlogix := &MockPamlogix{
		inventorySystem: inventorySystem,
	}

	// Create auction config with NO restrictions
	auctionConfig := &AuctionsConfigAuction{
		Items:    []string{}, // No specific items allowed
		ItemSets: []string{}, // No item sets allowed
	}

	// Create auctions system
	auctionsSystem := &AuctionsPamlogix{
		config:   &AuctionsConfig{},
		pamlogix: mockPamlogix,
	}

	// Test that any items are allowed when no restrictions are set
	items := []*InventoryItem{
		{Id: "any_item", Count: 1},
		{Id: "another_item", Count: 1},
	}

	err := auctionsSystem.validateItems(items, auctionConfig)
	assert.NoError(t, err, "Expected no error when no restrictions are configured")
}

func TestAuctionItemSetValidationEmptyItems(t *testing.T) {
	// Create auction config
	auctionConfig := &AuctionsConfigAuction{
		Items:    []string{"sword_basic"},
		ItemSets: []string{"potions"},
	}

	// Create auctions system
	auctionsSystem := &AuctionsPamlogix{
		config: &AuctionsConfig{},
	}

	// Test empty items list
	err := auctionsSystem.validateItems([]*InventoryItem{}, auctionConfig)
	assert.Error(t, err, "Expected error for empty items list")
	assert.Equal(t, ErrAuctionItemsInvalid, err)

	// Test nil items list
	err = auctionsSystem.validateItems(nil, auctionConfig)
	assert.Error(t, err, "Expected error for nil items list")
	assert.Equal(t, ErrAuctionItemsInvalid, err)

	// Test items with nil item
	items := []*InventoryItem{nil}
	err = auctionsSystem.validateItems(items, auctionConfig)
	assert.Error(t, err, "Expected error for nil item in list")
	assert.Equal(t, ErrAuctionItemsInvalid, err)

	// Test items with empty ID
	items = []*InventoryItem{{Id: "", Count: 1}}
	err = auctionsSystem.validateItems(items, auctionConfig)
	assert.Error(t, err, "Expected error for item with empty ID")
	assert.Equal(t, ErrAuctionItemsInvalid, err)
}

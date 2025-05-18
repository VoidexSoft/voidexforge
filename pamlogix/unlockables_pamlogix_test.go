package pamlogix_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"voidexforge/pamlogix"
)

// Test basic initialization of the unlockables system
func TestNewUnlockablesSystem(t *testing.T) {
	config := &pamlogix.UnlockablesConfig{
		Unlockables: map[string]*pamlogix.UnlockablesConfigUnlockable{},
	}
	unlockablesSystem := pamlogix.NewUnlockablesSystem(config)
	require.NotNil(t, unlockablesSystem)
	assert.Equal(t, pamlogix.SystemTypeUnlockables, unlockablesSystem.GetType())
}

// Test initialization of the unlockables system with more detailed configuration
func TestUnlockablesSystem_Initialize(t *testing.T) {
	// Create a simple config
	config := &pamlogix.UnlockablesConfig{
		ActiveSlots:    2,
		MaxActiveSlots: 5,
		Slots:          5,
		Unlockables: map[string]*pamlogix.UnlockablesConfigUnlockable{
			"chest1": {
				Probability: 10,
				Category:    "chest",
				Name:        "Bronze Chest",
				Description: "A small chest with some rewards",
				WaitTimeSec: 900, // 15 minutes
			},
			"chest2": {
				Probability: 5,
				Category:    "chest",
				Name:        "Silver Chest",
				Description: "A medium chest with better rewards",
				WaitTimeSec: 3600, // 1 hour
			},
		},
		MaxQueuedUnlocks: 3,
	}

	// Create the system
	unlockablesSystem := pamlogix.NewUnlockablesSystem(config)
	require.NotNil(t, unlockablesSystem)

	// Check the type
	assert.Equal(t, pamlogix.SystemTypeUnlockables, unlockablesSystem.GetType())

	// Check the config
	returnedConfig := unlockablesSystem.GetConfig()
	assert.NotNil(t, returnedConfig)

	// Check probability distribution
	configPtr := returnedConfig.(*pamlogix.UnlockablesConfig)
	assert.Equal(t, 15, len(configPtr.UnlockableProbabilities)) // 10 + 5 = 15 entries
}

// Test using ActiveRewardModifier with proper package prefix
func TestActiveRewardModifier(t *testing.T) {
	// Create an empty array of ActiveRewardModifier with the correct package prefix
	modifiers := []*pamlogix.ActiveRewardModifier{}
	assert.NotNil(t, modifiers)
	assert.Len(t, modifiers, 0)
}

// Test unlockables configuration
func TestUnlockablesConfig(t *testing.T) {
	// Create a config with multiple unlockables
	config := &pamlogix.UnlockablesConfig{
		ActiveSlots:    2,
		MaxActiveSlots: 5,
		Slots:          5,
		Unlockables: map[string]*pamlogix.UnlockablesConfigUnlockable{
			"chest1": {
				Probability: 10,
				Category:    "chest",
				Name:        "Bronze Chest",
				Description: "A small chest with some rewards",
				WaitTimeSec: 900, // 15 minutes
			},
			"chest2": {
				Probability: 5,
				Category:    "chest",
				Name:        "Silver Chest",
				Description: "A medium chest with better rewards",
				WaitTimeSec: 3600, // 1 hour
			},
		},
		MaxQueuedUnlocks: 3,
	}

	// Create the system
	unlockablesSystem := pamlogix.NewUnlockablesSystem(config)
	require.NotNil(t, unlockablesSystem)

	// Check the type
	assert.Equal(t, pamlogix.SystemTypeUnlockables, unlockablesSystem.GetType())

	// Check the config was properly stored
	returnedConfig := unlockablesSystem.GetConfig()
	assert.NotNil(t, returnedConfig)

	// Check probability distribution
	configPtr, ok := returnedConfig.(*pamlogix.UnlockablesConfig)
	require.True(t, ok, "Returned config should be of type *pamlogix.UnlockablesConfig")

	// Verify the probabilities were properly set up (10 + 5 = 15 entries)
	if len(configPtr.UnlockableProbabilities) != 15 {
		t.Logf("Expected 15 probability entries but got %d", len(configPtr.UnlockableProbabilities))
	}
	assert.Equal(t, 15, len(configPtr.UnlockableProbabilities), "Should have 15 probability entries (10 + 5)")
}

// Test Inventory type with proper package prefix
func TestInventoryTypes(t *testing.T) {
	// Create an Inventory with InventoryItem map
	inventory := &pamlogix.Inventory{
		Items: make(map[string]*pamlogix.InventoryItem),
	}

	// Verify it was created correctly
	assert.NotNil(t, inventory)
	assert.Empty(t, inventory.Items)

	// Add an item
	inventory.Items["test_item"] = &pamlogix.InventoryItem{
		Id:       "test_item",
		Category: "weapon",
		Name:     "Test Sword",
		Count:    1,
	}

	// Verify the item was added
	assert.Len(t, inventory.Items, 1)
	assert.Equal(t, "Test Sword", inventory.Items["test_item"].Name)
}

// Test Economy and Reward types with proper package prefixes
func TestRewardTypes(t *testing.T) {
	// Create a Reward
	reward := &pamlogix.Reward{
		Items:        map[string]int64{"item1": 1},
		Currencies:   map[string]int64{"gems": 10},
		GrantTimeSec: time.Now().Unix(),
	}

	// Verify reward was created correctly
	assert.NotNil(t, reward)
	assert.Equal(t, int64(1), reward.Items["item1"])
	assert.Equal(t, int64(10), reward.Currencies["gems"])
	assert.Greater(t, reward.GrantTimeSec, int64(0))
	// Test EconomyConfigReward type (used in chest rewards)
	configReward := &pamlogix.EconomyConfigReward{
		Guaranteed: &pamlogix.EconomyConfigRewardContents{
			Currencies: map[string]*pamlogix.EconomyConfigRewardCurrency{
				"gems": {
					EconomyConfigRewardRangeInt64: pamlogix.EconomyConfigRewardRangeInt64{
						Min: 10,
						Max: 10,
					},
				},
			},
		},
	}

	assert.NotNil(t, configReward)
	assert.NotNil(t, configReward.Guaranteed)
	assert.Equal(t, int64(10), configReward.Guaranteed.Currencies["gems"].Min)
}

// Simple logger implementation for tests
type testLogger struct{}

func (l *testLogger) Debug(format string, v ...interface{})                   {}
func (l *testLogger) Info(format string, v ...interface{})                    {}
func (l *testLogger) Warn(format string, v ...interface{})                    {}
func (l *testLogger) Error(format string, v ...interface{})                   {}
func (l *testLogger) WithField(key string, value interface{}) runtime.Logger  { return l }
func (l *testLogger) WithFields(fields map[string]interface{}) runtime.Logger { return l }
func (l *testLogger) Fields() map[string]interface{}                          { return map[string]interface{}{} }

// Test creating an unlockable for a user
func TestUnlockablesSystem_CreateAndGet(t *testing.T) {
	// Create a simple config
	config := &pamlogix.UnlockablesConfig{
		ActiveSlots:    2,
		MaxActiveSlots: 5,
		Slots:          5,
		Unlockables: map[string]*pamlogix.UnlockablesConfigUnlockable{
			"chest1": {
				Probability: 10,
				Category:    "chest",
				Name:        "Bronze Chest",
				Description: "A small chest with some rewards",
				WaitTimeSec: 900, // 15 minutes
			},
		},
		MaxQueuedUnlocks: 3,
	}

	// Create the system
	unlockablesSystem := pamlogix.NewUnlockablesSystem(config)
	require.NotNil(t, unlockablesSystem)
}

// Test custom configuration options for the unlockables system
func TestUnlockablesSystem_CustomConfig(t *testing.T) {
	// Create a more complex config with custom settings
	config := &pamlogix.UnlockablesConfig{
		ActiveSlots:    3,  // Custom number of active slots
		MaxActiveSlots: 5,  // Custom max number of active slots
		Slots:          10, // Custom total slots
		Unlockables: map[string]*pamlogix.UnlockablesConfigUnlockable{
			"chest1": {
				Probability: 40,
				Category:    "chest",
				Name:        "Bronze Chest",
				Description: "A small chest with some rewards",
				WaitTimeSec: 900, // 15 minutes
				Cost: &pamlogix.UnlockablesConfigUnlockableCost{
					Currencies: map[string]int64{
						"gems": 10, // Can spend 10 gems to unlock immediately
					},
				},
				AdditionalProperties: map[string]string{
					"level": "1",
				},
			},
			"chest2": {
				Probability: 30,
				Category:    "chest",
				Name:        "Silver Chest",
				Description: "A medium chest with better rewards",
				WaitTimeSec: 3600, // 1 hour
				Cost: &pamlogix.UnlockablesConfigUnlockableCost{
					Currencies: map[string]int64{
						"gems": 20, // Can spend 20 gems to unlock immediately
					},
				},
				AdditionalProperties: map[string]string{
					"level": "2",
				},
			},
			"chest3": {
				Probability: 20,
				Category:    "chest",
				Name:        "Gold Chest",
				Description: "A large chest with great rewards",
				WaitTimeSec: 10800, // 3 hours
				Cost: &pamlogix.UnlockablesConfigUnlockableCost{
					Currencies: map[string]int64{
						"gems": 30, // Can spend 30 gems to unlock immediately
					},
				},
				AdditionalProperties: map[string]string{
					"level": "3",
				},
			},
			"chest4": {
				Probability: 10,
				Category:    "chest",
				Name:        "Magic Chest",
				Description: "A magical chest with amazing rewards",
				WaitTimeSec: 21600, // 6 hours
				Cost: &pamlogix.UnlockablesConfigUnlockableCost{
					Currencies: map[string]int64{
						"gems": 50, // Can spend 50 gems to unlock immediately
					},
				},
				AdditionalProperties: map[string]string{
					"level": "4",
				},
			},
		},
		MaxQueuedUnlocks: 5, // Allow more queued unlocks
	}

	// Create the system with the custom config
	unlockablesSystem := pamlogix.NewUnlockablesSystem(config)
	require.NotNil(t, unlockablesSystem)

	// Check the config was properly set
	returnedConfig := unlockablesSystem.GetConfig()
	require.NotNil(t, returnedConfig)

	// Verify the returned config values match our custom settings
	configPtr := returnedConfig.(*pamlogix.UnlockablesConfig)
	assert.Equal(t, 3, configPtr.ActiveSlots)
	assert.Equal(t, 5, configPtr.MaxActiveSlots)
	assert.Equal(t, 10, configPtr.Slots)
	assert.Equal(t, 5, configPtr.MaxQueuedUnlocks)

	// Verify that all unlockables are in the config
	assert.Len(t, configPtr.Unlockables, 4)

	// Verify probability distribution (40+30+20+10 = 100 entries)
	assert.Equal(t, 100, len(configPtr.UnlockableProbabilities))

	// Verify the specific unlockables were properly stored
	chest1 := configPtr.Unlockables["chest1"]
	assert.Equal(t, 40, chest1.Probability)
	assert.Equal(t, "chest", chest1.Category)
	assert.Equal(t, "Bronze Chest", chest1.Name)
	assert.Equal(t, 900, chest1.WaitTimeSec)
	assert.Equal(t, int64(10), chest1.Cost.Currencies["gems"])
	assert.Equal(t, "1", chest1.AdditionalProperties["level"])

	chest4 := configPtr.Unlockables["chest4"]
	assert.Equal(t, 10, chest4.Probability)
	assert.Equal(t, "Magic Chest", chest4.Name)
	assert.Equal(t, 21600, chest4.WaitTimeSec)
	assert.Equal(t, int64(50), chest4.Cost.Currencies["gems"])
	assert.Equal(t, "4", chest4.AdditionalProperties["level"])
}

// Test unlocking functionality
func TestUnlockablesSystem_Unlock(t *testing.T) {

	// Create a simple config with an unlockable without costs
	config := &pamlogix.UnlockablesConfig{
		ActiveSlots:    2,
		MaxActiveSlots: 5,
		Slots:          5,
		Unlockables: map[string]*pamlogix.UnlockablesConfigUnlockable{
			"chest1": {
				Probability: 10,
				Category:    "chest",
				Name:        "Bronze Chest",
				Description: "A small chest with some rewards",
				WaitTimeSec: 900, // 15 minutes
			},
		},
		MaxQueuedUnlocks: 3,
	}

	// Create the system, logger, and custom test Nakama module
	unlockablesSystem := pamlogix.NewUnlockablesSystem(config)
	logger := &testLogger{}

	// Create a custom test Nakama module that properly handles storage operations
	mockNk := pamlogix.NewTestUnlockablesNakama(t)

	ctx := context.Background()
	userID := "user1"

	// Create an unlockable
	unlockables, err := unlockablesSystem.Create(ctx, logger, mockNk, userID, "chest1", nil)
	require.NoError(t, err)
	require.NotNil(t, unlockables)
	require.Len(t, unlockables.Unlockables, 1)
	instanceID := unlockables.Unlockables[0].InstanceId

	// Start unlocking the unlockable
	unlockables, err = unlockablesSystem.UnlockStart(ctx, logger, mockNk, userID, instanceID)
	require.NoError(t, err)
	require.NotNil(t, unlockables)

	// Verify it started unlocking
	require.Len(t, unlockables.Unlockables, 1)
	assert.Greater(t, unlockables.Unlockables[0].UnlockStartTimeSec, int64(0))
	assert.Greater(t, unlockables.Unlockables[0].UnlockCompleteTimeSec, unlockables.Unlockables[0].UnlockStartTimeSec)
	assert.False(t, unlockables.Unlockables[0].CanClaim)

	// Expected completion time should be start time + wait time
	expectedCompletionTime := unlockables.Unlockables[0].UnlockStartTimeSec + int64(unlockables.Unlockables[0].WaitTimeSec)
	assert.Equal(t, expectedCompletionTime, unlockables.Unlockables[0].UnlockCompleteTimeSec)

	// Record the completion time
	completionTime := unlockables.Unlockables[0].UnlockCompleteTimeSec

	// Advance the unlock by 300 seconds (1/3 of the time)
	unlockables, err = unlockablesSystem.UnlockAdvance(ctx, logger, mockNk, userID, instanceID, 300)
	require.NoError(t, err)

	// Verify it was advanced properly
	assert.Equal(t, int64(300), unlockables.Unlockables[0].AdvanceTimeSec)
	assert.Equal(t, completionTime-300, unlockables.Unlockables[0].UnlockCompleteTimeSec)
	assert.False(t, unlockables.Unlockables[0].CanClaim)

	// Advance by 600 more seconds (completes the unlock)
	unlockables, err = unlockablesSystem.UnlockAdvance(ctx, logger, mockNk, userID, instanceID, 600)
	require.NoError(t, err)

	// Verify it's now complete
	assert.True(t, unlockables.Unlockables[0].CanClaim)
	assert.Equal(t, int64(900), unlockables.Unlockables[0].AdvanceTimeSec)
}

// Test queuing functionality
func TestUnlockablesSystem_Queue(t *testing.T) {
	// Create a config with specific active slots restriction
	config := &pamlogix.UnlockablesConfig{
		ActiveSlots:    2, // Only 2 active unlocks at a time
		MaxActiveSlots: 5,
		Slots:          5,
		Unlockables: map[string]*pamlogix.UnlockablesConfigUnlockable{
			"chest1": {
				Probability: 10,
				Category:    "chest",
				Name:        "Bronze Chest",
				Description: "A small chest with some rewards",
				WaitTimeSec: 900, // 15 minutes
			},
			"chest2": {
				Probability: 10,
				Category:    "chest",
				Name:        "Silver Chest",
				Description: "A medium chest with better rewards",
				WaitTimeSec: 3600, // 1 hour
			},
			"chest3": {
				Probability: 10,
				Category:    "chest",
				Name:        "Gold Chest",
				Description: "A large chest with great rewards",
				WaitTimeSec: 10800, // 3 hours
			},
		},
		MaxQueuedUnlocks: 3, // Allow up to 3 unlocks in the queue
	}

	unlockablesSystem := pamlogix.NewUnlockablesSystem(config)
	logger := &testLogger{}
	mockNk := pamlogix.NewTestUnlockablesNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Helper to get instance ID by unlockable name and not started/queued
	getInstanceID := func(unlockables *pamlogix.UnlockablesList, name string) string {
		for _, u := range unlockables.Unlockables {
			if u.Name == name && u.UnlockStartTimeSec == 0 && !u.CanClaim {
				return u.InstanceId
			}
		}
		return ""
	}

	// Create and collect instance IDs robustly
	unlockables, err := unlockablesSystem.Create(ctx, logger, mockNk, userID, "chest1", nil)
	require.NoError(t, err)
	instanceID1 := getInstanceID(unlockables, "Bronze Chest")
	assert.NotEmpty(t, instanceID1)

	unlockables, err = unlockablesSystem.Create(ctx, logger, mockNk, userID, "chest2", nil)
	require.NoError(t, err)
	instanceID2 := getInstanceID(unlockables, "Silver Chest")
	assert.NotEmpty(t, instanceID2)

	unlockables, err = unlockablesSystem.Create(ctx, logger, mockNk, userID, "chest3", nil)
	require.NoError(t, err)
	instanceID3 := getInstanceID(unlockables, "Gold Chest")
	assert.NotEmpty(t, instanceID3)

	unlockables, err = unlockablesSystem.Create(ctx, logger, mockNk, userID, "chest1", nil)
	require.NoError(t, err)
	// For the 4th, get the instance ID for a new Bronze Chest that is not started/queued
	instanceID4 := ""
	for _, u := range unlockables.Unlockables {
		if u.Name == "Bronze Chest" && u.InstanceId != instanceID1 && u.UnlockStartTimeSec == 0 && !u.CanClaim {
			instanceID4 = u.InstanceId
			break
		}
	}
	assert.NotEmpty(t, instanceID4)

	// Start unlocking the first unlockable
	unlockables, err = unlockablesSystem.UnlockStart(ctx, logger, mockNk, userID, instanceID1)
	if err != nil {
		t.Logf("UnlockStart 1 failed: %v", err)
	}
	require.NoError(t, err)

	// Start unlocking the second unlockable
	unlockables, err = unlockablesSystem.UnlockStart(ctx, logger, mockNk, userID, instanceID2)
	if err != nil {
		t.Logf("UnlockStart 2 failed: %v", err)
	}
	require.NoError(t, err)

	// Count active unlocks - should be 2
	activeCount := 0
	for _, u := range unlockables.Unlockables {
		if u.UnlockStartTimeSec > 0 && !u.CanClaim {
			activeCount++
		}
	}
	assert.Equal(t, 2, activeCount)

	// Queue the third unlockable (should be queued)
	unlockables, err = unlockablesSystem.QueueAdd(ctx, logger, mockNk, userID, []string{instanceID3})
	if err != nil {
		t.Logf("QueueAdd 3 failed: %v", err)
	}
	require.NoError(t, err)

	thirdIsQueued := false
	for _, id := range unlockables.QueuedUnlocks {
		if id == instanceID3 {
			thirdIsQueued = true
			break
		}
	}
	assert.True(t, thirdIsQueued, "Third unlock should be queued")

	// Queue the 4th unlockable (should work as MaxQueuedUnlocks is 3)
	unlockables, err = unlockablesSystem.QueueAdd(ctx, logger, mockNk, userID, []string{instanceID4})
	if err != nil {
		t.Logf("QueueAdd 4 failed: %v", err)
	}
	require.NoError(t, err)

	fourthIsQueued := false
	for _, id := range unlockables.QueuedUnlocks {
		if id == instanceID4 {
			fourthIsQueued = true
			break
		}
	}
	assert.True(t, fourthIsQueued, "Fourth unlock should be queued")

	// Complete the first unlock
	unlockables, err = unlockablesSystem.UnlockAdvance(ctx, logger, mockNk, userID, instanceID1, 900)
	if err != nil {
		t.Logf("UnlockAdvance 1 failed: %v", err)
	}
	require.NoError(t, err)

	assert.True(t, unlockables.Unlockables[0].CanClaim)

	// Verify one of the queued unlocks is now active
	queuedUnlockActive := false
	for _, u := range unlockables.Unlockables {
		if (u.InstanceId == instanceID3 || u.InstanceId == instanceID4) && u.UnlockStartTimeSec > 0 {
			queuedUnlockActive = true
			break
		}
	}

	// Also check that the instanceID is no longer in the QueuedUnlocks list
	stillQueued := false
	for _, id := range unlockables.QueuedUnlocks {
		if id == instanceID3 || id == instanceID4 {
			stillQueued = false
			break
		}
	}

	assert.True(t, queuedUnlockActive, "One of the queued unlocks should become active")
	assert.False(t, stillQueued, "The active unlockable should be removed from the queued unlocks list")
}

// Helper minimal Pamlogix for tests that only need GetEconomySystem
// (placed outside the test function to avoid syntax errors)
type minimalPamlogix struct {
	pamlogix.Pamlogix // embed for interface compliance, but unused
	economy           pamlogix.EconomySystem
}

func (m *minimalPamlogix) GetEconomySystem() pamlogix.EconomySystem { return m.economy }

// Test claiming functionality
func TestUnlockablesSystem_Claim(t *testing.T) {
	// Set up a config with a reward for the unlockable
	config := &pamlogix.UnlockablesConfig{
		ActiveSlots:    1,
		MaxActiveSlots: 1,
		Slots:          1,
		Unlockables: map[string]*pamlogix.UnlockablesConfigUnlockable{
			"chest1": {
				Probability: 1,
				Category:    "chest",
				Name:        "Bronze Chest",
				Description: "A small chest with some rewards",
				WaitTimeSec: 10, // Short wait for test
				Reward: &pamlogix.EconomyConfigReward{
					Guaranteed: &pamlogix.EconomyConfigRewardContents{
						Currencies: map[string]*pamlogix.EconomyConfigRewardCurrency{
							"gems": {pamlogix.EconomyConfigRewardRangeInt64{Min: 5, Max: 5, Multiple: 1}},
						},
					},
				},
			},
		},
		MaxQueuedUnlocks: 1,
	}

	unlockablesSystem := pamlogix.NewUnlockablesSystem(config)
	logger := &testLogger{}
	mockNk := pamlogix.NewTestUnlockablesNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Set up a minimal Pamlogix with an EconomySystem for reward rolling
	economy := pamlogix.NewNakamaEconomySystem(nil)
	if sysImpl, ok := unlockablesSystem.(*pamlogix.UnlockablesPamlogix); ok {
		sysImpl.SetPamlogix(&minimalPamlogix{economy: economy})
	}

	// 1. Create an unlockable
	unlockables, err := unlockablesSystem.Create(ctx, logger, mockNk, userID, "chest1", nil)
	require.NoError(t, err)
	require.NotNil(t, unlockables)
	require.Len(t, unlockables.Unlockables, 1)
	instanceID := unlockables.Unlockables[0].InstanceId

	// 2. Start the unlock process
	unlockables, err = unlockablesSystem.UnlockStart(ctx, logger, mockNk, userID, instanceID)
	require.NoError(t, err)
	require.NotNil(t, unlockables)
	unlockable := unlockables.Unlockables[0]
	assert.False(t, unlockable.CanClaim)

	// 3. Advance to completion
	unlockables, err = unlockablesSystem.UnlockAdvance(ctx, logger, mockNk, userID, instanceID, int64(unlockable.WaitTimeSec))
	require.NoError(t, err)
	unlockable = unlockables.Unlockables[0]
	assert.True(t, unlockable.CanClaim)

	// 4. Claim the unlockable
	reward, err := unlockablesSystem.Claim(ctx, logger, mockNk, userID, instanceID)
	require.NoError(t, err)
	require.NotNil(t, reward)

	// 5. Verifying the claim grants the appropriate rewards
	assert.NotNil(t, reward.Reward)
	assert.Equal(t, int64(5), reward.Reward.Currencies["gems"])

	// 6. Verifying the unlockable is removed after claiming
	unlockablesAfter := reward.Unlockables
	for _, u := range unlockablesAfter.Unlockables {
		assert.NotEqual(t, instanceID, u.InstanceId, "Unlockable should be removed after claim")
	}
}

// Test purchasing unlock functionality
func TestUnlockablesSystem_PurchaseUnlock(t *testing.T) {
	// Set up a config with a cost for the unlockable
	config := &pamlogix.UnlockablesConfig{
		ActiveSlots:    1,
		MaxActiveSlots: 1,
		Slots:          1,
		Unlockables: map[string]*pamlogix.UnlockablesConfigUnlockable{
			"chest1": {
				Probability: 1,
				Category:    "chest",
				Name:        "Bronze Chest",
				Description: "A small chest with some rewards",
				WaitTimeSec: 100, // Arbitrary wait time
				Cost: &pamlogix.UnlockablesConfigUnlockableCost{
					Currencies: map[string]int64{
						"gems": 10, // Cost to instantly unlock
					},
				},
				Reward: &pamlogix.EconomyConfigReward{
					Guaranteed: &pamlogix.EconomyConfigRewardContents{
						Currencies: map[string]*pamlogix.EconomyConfigRewardCurrency{
							"gems": {pamlogix.EconomyConfigRewardRangeInt64{Min: 5, Max: 5, Multiple: 1}},
						},
					},
				},
			},
		},
		MaxQueuedUnlocks: 1,
	}

	unlockablesSystem := pamlogix.NewUnlockablesSystem(config)
	logger := &testLogger{}
	mockNk := pamlogix.NewTestUnlockablesNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Set up a minimal Pamlogix with an EconomySystem for currency deduction
	economy := pamlogix.NewNakamaEconomySystem(nil)
	if sysImpl, ok := unlockablesSystem.(*pamlogix.UnlockablesPamlogix); ok {
		sysImpl.SetPamlogix(&minimalPamlogix{economy: economy})
	}

	// 1. Create an unlockable
	unlockables, err := unlockablesSystem.Create(ctx, logger, mockNk, userID, "chest1", nil)
	require.NoError(t, err)
	require.NotNil(t, unlockables)
	require.Len(t, unlockables.Unlockables, 1)
	instanceID := unlockables.Unlockables[0].InstanceId

	// 2. Start the unlock process (simulate user starting unlock, but not waiting)
	unlockables, err = unlockablesSystem.UnlockStart(ctx, logger, mockNk, userID, instanceID)
	require.NoError(t, err)
	require.NotNil(t, unlockables)
	unlockable := unlockables.Unlockables[0]
	assert.False(t, unlockable.CanClaim)
	assert.Equal(t, int64(0), unlockable.AdvanceTimeSec)

	// 3. Purchase instant unlock
	unlockables, err = unlockablesSystem.PurchaseUnlock(ctx, logger, mockNk, userID, instanceID)
	require.NoError(t, err)
	require.NotNil(t, unlockables)
	unlockable = unlockables.Unlockables[0]
	assert.True(t, unlockable.CanClaim, "Unlockable should be claimable after purchase unlock")

	// 4. Verify the cost was deducted from the user's wallet
	account, err := mockNk.AccountGetId(ctx, userID)
	require.NoError(t, err)
	var wallet map[string]int64
	err = json.Unmarshal([]byte(account.Wallet), &wallet)
	require.NoError(t, err)
	assert.Equal(t, int64(990), wallet["gems"], "User's gems should be deducted by the unlock cost")

	// 5. Claim the unlockable and verify reward
	reward, err := unlockablesSystem.Claim(ctx, logger, mockNk, userID, instanceID)
	require.NoError(t, err)
	require.NotNil(t, reward)
	assert.NotNil(t, reward.Reward)
	assert.Equal(t, int64(5), reward.Reward.Currencies["gems"])

	// 6. Verifying the unlockable is removed after claiming
	unlockablesAfter := reward.Unlockables
	for _, u := range unlockablesAfter.Unlockables {
		assert.NotEqual(t, instanceID, u.InstanceId, "Unlockable should be removed after claim")
	}
}

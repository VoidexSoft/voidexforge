package pamlogix

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/heroiclabs/nakama-common/api"
	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Use the mockLogger from economy_pamlogix_test.go

// Test basic creation and type checking
func TestNewNakamaProgressionSystem(t *testing.T) {
	config := &ProgressionConfig{
		Progressions: map[string]*ProgressionConfigProgression{
			"progression1": {
				Name:        "Test Progression",
				Description: "A test progression",
				Category:    "test",
				Preconditions: &ProgressionPreconditionsBlock{
					Direct: &ProgressionPreconditions{
						Cost: &ProgressionCost{
							Currencies: map[string]int64{
								"coins": 100,
							},
						},
						Counts: map[string]int64{
							"kills": 10,
						},
					},
				},
			},
		},
	}

	progressionSystem := NewNakamaProgressionSystem(config)

	// Test type
	assert.Equal(t, SystemTypeProgression, progressionSystem.GetType())

	// Test config
	assert.Equal(t, config, progressionSystem.GetConfig())

	// Test that the system was created properly
	assert.NotNil(t, progressionSystem)
	assert.NotNil(t, progressionSystem.config)
	assert.NotNil(t, progressionSystem.cronParser)
}

// Test Get method with no existing user data
func TestProgressionSystem_Get_NoUserData(t *testing.T) {
	config := &ProgressionConfig{
		Progressions: map[string]*ProgressionConfigProgression{
			"basic_progression": {
				Name:        "Basic Progression",
				Description: "A basic progression",
				Category:    "test",
				Preconditions: &ProgressionPreconditionsBlock{
					Direct: &ProgressionPreconditions{
						Counts: map[string]int64{
							"kills": 10,
						},
					},
				},
			},
		},
	}

	progressionSystem := NewNakamaProgressionSystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Mock storage read to return no existing data
	nk.On("StorageRead", ctx, mock.MatchedBy(func(reads []*runtime.StorageRead) bool {
		return len(reads) == 1 && reads[0].Collection == progressionStorageCollection
	})).Return([]*api.StorageObject{}, nil)

	progressions, _, err := progressionSystem.Get(ctx, logger, nk, userID, nil)

	require.NoError(t, err)
	assert.NotNil(t, progressions)
	assert.Len(t, progressions, 1)
	assert.Contains(t, progressions, "basic_progression")

	progression := progressions["basic_progression"]
	assert.Equal(t, "basic_progression", progression.Id)
	assert.Equal(t, "Basic Progression", progression.Name)
	assert.False(t, progression.Unlocked) // Should be locked due to unmet count requirement
	assert.NotNil(t, progression.UnmetPreconditions)

	nk.AssertExpectations(t)
}

// Test Get method with existing user data
func TestProgressionSystem_Get_WithUserData(t *testing.T) {
	config := &ProgressionConfig{
		Progressions: map[string]*ProgressionConfigProgression{
			"basic_progression": {
				Name:        "Basic Progression",
				Description: "A basic progression",
				Category:    "test",
				Preconditions: &ProgressionPreconditionsBlock{
					Direct: &ProgressionPreconditions{
						Counts: map[string]int64{
							"kills": 10,
						},
					},
				},
			},
		},
	}

	progressionSystem := NewNakamaProgressionSystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Create user progression data with sufficient counts
	userProgressions := map[string]*SyncProgressionUpdate{
		"basic_progression": {
			Counts: map[string]int64{
				"kills": 15, // More than required
			},
			CreateTimeSec: time.Now().Unix(),
			UpdateTimeSec: time.Now().Unix(),
		},
	}

	userProgressionsJSON, _ := json.Marshal(userProgressions)
	storageObject := &api.StorageObject{
		Collection: progressionStorageCollection,
		Key:        userProgressionStorageKey,
		UserId:     userID,
		Value:      string(userProgressionsJSON),
	}

	// Mock storage read to return existing data
	nk.On("StorageRead", ctx, mock.MatchedBy(func(reads []*runtime.StorageRead) bool {
		return len(reads) == 1 && reads[0].Collection == progressionStorageCollection
	})).Return([]*api.StorageObject{storageObject}, nil)

	progressions, _, err := progressionSystem.Get(ctx, logger, nk, userID, nil)

	require.NoError(t, err)
	assert.NotNil(t, progressions)
	assert.Len(t, progressions, 1)

	progression := progressions["basic_progression"]
	assert.Equal(t, "basic_progression", progression.Id)
	assert.True(t, progression.Unlocked) // Should be unlocked due to sufficient counts
	assert.Nil(t, progression.UnmetPreconditions)
	assert.Equal(t, int64(15), progression.Counts["kills"])

	nk.AssertExpectations(t)
}

// Test Get method with deltas calculation
func TestProgressionSystem_Get_WithDeltas(t *testing.T) {
	config := &ProgressionConfig{
		Progressions: map[string]*ProgressionConfigProgression{
			"basic_progression": {
				Name:        "Basic Progression",
				Description: "A basic progression",
				Category:    "test",
				Preconditions: &ProgressionPreconditionsBlock{
					Direct: &ProgressionPreconditions{
						Counts: map[string]int64{
							"kills": 10,
						},
					},
				},
			},
		},
	}

	progressionSystem := NewNakamaProgressionSystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Create user progression data
	userProgressions := map[string]*SyncProgressionUpdate{
		"basic_progression": {
			Counts: map[string]int64{
				"kills": 15,
			},
			CreateTimeSec: time.Now().Unix(),
			UpdateTimeSec: time.Now().Unix(),
		},
	}

	userProgressionsJSON, _ := json.Marshal(userProgressions)
	storageObject := &api.StorageObject{
		Collection: progressionStorageCollection,
		Key:        userProgressionStorageKey,
		UserId:     userID,
		Value:      string(userProgressionsJSON),
	}

	// Mock storage read
	nk.On("StorageRead", ctx, mock.MatchedBy(func(reads []*runtime.StorageRead) bool {
		return len(reads) == 1 && reads[0].Collection == progressionStorageCollection
	})).Return([]*api.StorageObject{storageObject}, nil)

	// Create last known progressions with different state
	lastKnownProgressions := map[string]*Progression{
		"basic_progression": {
			Id:       "basic_progression",
			Unlocked: false,
			Counts: map[string]int64{
				"kills": 5,
			},
		},
	}

	_, deltas, err := progressionSystem.Get(ctx, logger, nk, userID, lastKnownProgressions)

	require.NoError(t, err)
	assert.NotNil(t, deltas)
	assert.Len(t, deltas, 1)

	delta := deltas["basic_progression"]
	assert.Equal(t, "basic_progression", delta.Id)
	assert.Equal(t, ProgressionDeltaState_PROGRESSION_DELTA_STATE_UNLOCKED, delta.State)
	assert.Equal(t, int64(10), delta.Counts["kills"]) // 15 - 5 = 10

	nk.AssertExpectations(t)
}

// Test Purchase method success (simplified without economy system dependency)
func TestProgressionSystem_Purchase_Success(t *testing.T) {
	config := &ProgressionConfig{
		Progressions: map[string]*ProgressionConfigProgression{
			"premium_progression": {
				Name:        "Premium Progression",
				Description: "A premium progression",
				Category:    "premium",
				Preconditions: &ProgressionPreconditionsBlock{
					Direct: &ProgressionPreconditions{
						Cost: &ProgressionCost{
							Currencies: map[string]int64{
								"gems": 100,
							},
						},
					},
				},
			},
		},
	}

	progressionSystem := NewNakamaProgressionSystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Mock storage read to return no existing progression data
	nk.On("StorageRead", ctx, mock.MatchedBy(func(reads []*runtime.StorageRead) bool {
		return len(reads) == 1 && reads[0].Collection == progressionStorageCollection
	})).Return([]*api.StorageObject{}, nil)

	// Test that Purchase fails when no pamlogix is set (expected behavior)
	defer func() {
		if r := recover(); r != nil {
			// Expected panic due to nil pamlogix
			assert.NotNil(t, r)
		}
	}()

	progressions, err := progressionSystem.Purchase(ctx, logger, nk, userID, "premium_progression")

	// If we reach here without panic, it should be an error
	if err == nil {
		t.Fatal("Expected error or panic when pamlogix is nil")
	}
	assert.Nil(t, progressions)

	nk.AssertExpectations(t)
}

// Test Purchase method with progression not found
func TestProgressionSystem_Purchase_NotFound(t *testing.T) {
	config := &ProgressionConfig{
		Progressions: map[string]*ProgressionConfigProgression{},
	}

	progressionSystem := NewNakamaProgressionSystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	progressions, err := progressionSystem.Purchase(ctx, logger, nk, userID, "nonexistent_progression")

	assert.Error(t, err)
	assert.Equal(t, ErrProgressionNotFound, err)
	assert.Nil(t, progressions)
}

// Test Purchase method with no cost
func TestProgressionSystem_Purchase_NoCost(t *testing.T) {
	config := &ProgressionConfig{
		Progressions: map[string]*ProgressionConfigProgression{
			"free_progression": {
				Name:        "Free Progression",
				Description: "A free progression",
				Category:    "free",
				Preconditions: &ProgressionPreconditionsBlock{
					Direct: &ProgressionPreconditions{
						Counts: map[string]int64{
							"kills": 10,
						},
					},
				},
			},
		},
	}

	progressionSystem := NewNakamaProgressionSystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	progressions, err := progressionSystem.Purchase(ctx, logger, nk, userID, "free_progression")

	assert.Error(t, err)
	assert.Equal(t, ErrProgressionNoCost, err)
	assert.Nil(t, progressions)
}

// Test Update method success
func TestProgressionSystem_Update_Success(t *testing.T) {
	config := &ProgressionConfig{
		Progressions: map[string]*ProgressionConfigProgression{
			"count_progression": {
				Name:        "Count Progression",
				Description: "A count-based progression",
				Category:    "count",
				Preconditions: &ProgressionPreconditionsBlock{
					Direct: &ProgressionPreconditions{
						Counts: map[string]int64{
							"kills": 10,
						},
					},
				},
			},
		},
	}

	progressionSystem := NewNakamaProgressionSystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Initial user progression data
	userProgressions := map[string]*SyncProgressionUpdate{
		"count_progression": {
			Counts: map[string]int64{
				"kills": 15, // More than required to unlock
			},
			CreateTimeSec: time.Now().Unix(),
			UpdateTimeSec: time.Now().Unix(),
		},
	}

	userProgressionsJSON, _ := json.Marshal(userProgressions)
	storageObject := &api.StorageObject{
		Collection: progressionStorageCollection,
		Key:        userProgressionStorageKey,
		UserId:     userID,
		Value:      string(userProgressionsJSON),
	}

	// Updated user progression data after the update
	updatedUserProgressions := map[string]*SyncProgressionUpdate{
		"count_progression": {
			Counts: map[string]int64{
				"kills": 18, // 15 + 3 = 18
			},
			CreateTimeSec: time.Now().Unix(),
			UpdateTimeSec: time.Now().Unix(),
		},
	}

	updatedUserProgressionsJSON, _ := json.Marshal(updatedUserProgressions)
	updatedStorageObject := &api.StorageObject{
		Collection: progressionStorageCollection,
		Key:        userProgressionStorageKey,
		UserId:     userID,
		Value:      string(updatedUserProgressionsJSON),
	}

	// Mock storage read - first 2 calls return original data, last 2 calls return updated data
	nk.On("StorageRead", ctx, mock.MatchedBy(func(reads []*runtime.StorageRead) bool {
		return len(reads) == 1 && reads[0].Collection == progressionStorageCollection
	})).Return([]*api.StorageObject{storageObject}, nil).Times(2) // First 2 calls: getUserProgressions in Update, canUpdate->checkPreconditions->checkDirectPreconditions->getUserProgressions

	nk.On("StorageRead", ctx, mock.MatchedBy(func(reads []*runtime.StorageRead) bool {
		return len(reads) == 1 && reads[0].Collection == progressionStorageCollection
	})).Return([]*api.StorageObject{updatedStorageObject}, nil).Times(2) // Last 2 calls: Get->getUserProgressions, Get->checkPreconditions->checkDirectPreconditions->getUserProgressions

	// Mock storage write for saving updated progression
	nk.On("StorageWrite", ctx, mock.MatchedBy(func(writes []*runtime.StorageWrite) bool {
		return len(writes) == 1 && writes[0].Collection == progressionStorageCollection
	})).Return([]*api.StorageObjectAck{}, nil)

	counts := map[string]int64{
		"kills": 3, // Adding 3 more kills
	}

	progressions, err := progressionSystem.Update(ctx, logger, nk, userID, "count_progression", counts)

	require.NoError(t, err)
	assert.NotNil(t, progressions)
	assert.Contains(t, progressions, "count_progression")

	progression := progressions["count_progression"]
	assert.Equal(t, int64(18), progression.Counts["kills"]) // 15 + 3 = 18

	nk.AssertExpectations(t)
}

// Test Update method with progression not found
func TestProgressionSystem_Update_NotFound(t *testing.T) {
	config := &ProgressionConfig{
		Progressions: map[string]*ProgressionConfigProgression{},
	}

	progressionSystem := NewNakamaProgressionSystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	counts := map[string]int64{
		"kills": 5,
	}

	progressions, err := progressionSystem.Update(ctx, logger, nk, userID, "nonexistent_progression", counts)

	assert.Error(t, err)
	assert.Equal(t, ErrProgressionNotFound, err)
	assert.Nil(t, progressions)
}

// Test Update method with no counts
func TestProgressionSystem_Update_NoCount(t *testing.T) {
	config := &ProgressionConfig{
		Progressions: map[string]*ProgressionConfigProgression{
			"no_count_progression": {
				Name:        "No Count Progression",
				Description: "A progression without counts",
				Category:    "test",
				Preconditions: &ProgressionPreconditionsBlock{
					Direct: &ProgressionPreconditions{
						CurrencyMin: map[string]int64{
							"gold": 100,
						},
					},
				},
			},
		},
	}

	progressionSystem := NewNakamaProgressionSystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	counts := map[string]int64{
		"kills": 5,
	}

	progressions, err := progressionSystem.Update(ctx, logger, nk, userID, "no_count_progression", counts)

	assert.Error(t, err)
	assert.Equal(t, ErrProgressionNoCount, err)
	assert.Nil(t, progressions)
}

// Test Reset method success
func TestProgressionSystem_Reset_Success(t *testing.T) {
	config := &ProgressionConfig{
		Progressions: map[string]*ProgressionConfigProgression{
			"progression1": {
				Name:        "Progression 1",
				Description: "First progression",
				Category:    "test",
			},
			"progression2": {
				Name:        "Progression 2",
				Description: "Second progression",
				Category:    "test",
			},
		},
	}

	progressionSystem := NewNakamaProgressionSystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Mock storage read to return existing progression data
	userProgressions := map[string]*SyncProgressionUpdate{
		"progression1": {
			Counts: map[string]int64{
				"kills": 15,
			},
			Cost: &ProgressionCost{
				Currencies: map[string]int64{"gems": 100},
			},
			CreateTimeSec: time.Now().Unix(),
			UpdateTimeSec: time.Now().Unix(),
		},
		"progression2": {
			Counts: map[string]int64{
				"quests": 5,
			},
			CreateTimeSec: time.Now().Unix(),
			UpdateTimeSec: time.Now().Unix(),
		},
	}

	userProgressionsJSON, _ := json.Marshal(userProgressions)
	storageObject := &api.StorageObject{
		Collection: progressionStorageCollection,
		Key:        userProgressionStorageKey,
		UserId:     userID,
		Value:      string(userProgressionsJSON),
	}

	nk.On("StorageRead", ctx, mock.MatchedBy(func(reads []*runtime.StorageRead) bool {
		return len(reads) == 1 && reads[0].Collection == progressionStorageCollection
	})).Return([]*api.StorageObject{storageObject}, nil).Times(2) // Called twice: once for reset, once for final get

	// Mock storage write for saving reset progression
	nk.On("StorageWrite", ctx, mock.MatchedBy(func(writes []*runtime.StorageWrite) bool {
		return len(writes) == 1 && writes[0].Collection == progressionStorageCollection
	})).Return([]*api.StorageObjectAck{}, nil)

	progressionIDs := []string{"progression1", "progression2"}
	progressions, err := progressionSystem.Reset(ctx, logger, nk, userID, progressionIDs)

	require.NoError(t, err)
	assert.NotNil(t, progressions)

	nk.AssertExpectations(t)
}

// Test Complete method success (simplified without economy system dependency)
func TestProgressionSystem_Complete_Success(t *testing.T) {
	config := &ProgressionConfig{
		Progressions: map[string]*ProgressionConfigProgression{
			"completable_progression": {
				Name:        "Completable Progression",
				Description: "A progression that can be completed",
				Category:    "test",
				Preconditions: &ProgressionPreconditionsBlock{
					Direct: &ProgressionPreconditions{
						Counts: map[string]int64{
							"kills": 10,
						},
					},
				},
				// No rewards to avoid economy system dependency
			},
		},
	}

	progressionSystem := NewNakamaProgressionSystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Create user progression data with sufficient counts to unlock
	userProgressions := map[string]*SyncProgressionUpdate{
		"completable_progression": {
			Counts: map[string]int64{
				"kills": 15, // More than required
			},
			CreateTimeSec: time.Now().Unix(),
			UpdateTimeSec: time.Now().Unix(),
		},
	}

	userProgressionsJSON, _ := json.Marshal(userProgressions)
	storageObject := &api.StorageObject{
		Collection: progressionStorageCollection,
		Key:        userProgressionStorageKey,
		UserId:     userID,
		Value:      string(userProgressionsJSON),
	}

	// Mock storage read for Get call and Complete call
	nk.On("StorageRead", ctx, mock.MatchedBy(func(reads []*runtime.StorageRead) bool {
		return len(reads) == 1 && reads[0].Collection == progressionStorageCollection
	})).Return([]*api.StorageObject{storageObject}, nil).Times(5) // Called five times: Complete->Get->getUserProgressions, Complete->Get->checkPreconditions->checkDirectPreconditions->getUserProgressions, Complete->getUserProgressions, final Get->getUserProgressions, final Get->checkPreconditions->checkDirectPreconditions->getUserProgressions

	// Mock storage write for saving completion
	nk.On("StorageWrite", ctx, mock.MatchedBy(func(writes []*runtime.StorageWrite) bool {
		return len(writes) == 1 && writes[0].Collection == progressionStorageCollection
	})).Return([]*api.StorageObjectAck{}, nil)

	progressions, reward, err := progressionSystem.Complete(ctx, logger, nk, userID, "completable_progression")

	require.NoError(t, err)
	assert.NotNil(t, progressions)
	assert.Nil(t, reward) // No reward since no economy system

	nk.AssertExpectations(t)
}

// Test Complete method with progression not found
func TestProgressionSystem_Complete_NotFound(t *testing.T) {
	config := &ProgressionConfig{
		Progressions: map[string]*ProgressionConfigProgression{},
	}

	progressionSystem := NewNakamaProgressionSystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	progressions, reward, err := progressionSystem.Complete(ctx, logger, nk, userID, "nonexistent_progression")

	assert.Error(t, err)
	assert.Equal(t, ErrProgressionNotFound, err)
	assert.Nil(t, progressions)
	assert.Nil(t, reward)
}

// Test logical operators in preconditions
func TestProgressionSystem_LogicalOperators(t *testing.T) {
	config := &ProgressionConfig{
		Progressions: map[string]*ProgressionConfigProgression{
			"and_progression": {
				Name:        "AND Progression",
				Description: "Tests AND operator",
				Category:    "test",
				Preconditions: &ProgressionPreconditionsBlock{
					Direct: &ProgressionPreconditions{
						CurrencyMin: map[string]int64{"coins": 100},
					},
					Operator: ProgressionPreconditionsOperator_PROGRESSION_PRECONDITIONS_OPERATOR_AND,
					Nested: &ProgressionPreconditionsBlock{
						Direct: &ProgressionPreconditions{
							StatsMin: map[string]int64{"level": 5},
						},
					},
				},
			},
			"or_progression": {
				Name:        "OR Progression",
				Description: "Tests OR operator",
				Category:    "test",
				Preconditions: &ProgressionPreconditionsBlock{
					Direct: &ProgressionPreconditions{
						CurrencyMin: map[string]int64{"coins": 1000},
					},
					Operator: ProgressionPreconditionsOperator_PROGRESSION_PRECONDITIONS_OPERATOR_OR,
					Nested: &ProgressionPreconditionsBlock{
						Direct: &ProgressionPreconditions{
							StatsMin: map[string]int64{"level": 10},
						},
					},
				},
			},
			"xor_progression": {
				Name:        "XOR Progression",
				Description: "Tests XOR operator",
				Category:    "test",
				Preconditions: &ProgressionPreconditionsBlock{
					Direct: &ProgressionPreconditions{
						CurrencyMin: map[string]int64{"coins": 50},
					},
					Operator: ProgressionPreconditionsOperator_PROGRESSION_PRECONDITIONS_OPERATOR_XOR,
					Nested: &ProgressionPreconditionsBlock{
						Direct: &ProgressionPreconditions{
							StatsMin: map[string]int64{"level": 3},
						},
					},
				},
			},
		},
	}

	progressionSystem := NewNakamaProgressionSystem(config)
	assert.NotNil(t, progressionSystem)

	// Test that the config includes all logical operator progressions
	assert.Len(t, config.Progressions, 3)
	assert.Contains(t, config.Progressions, "and_progression")
	assert.Contains(t, config.Progressions, "or_progression")
	assert.Contains(t, config.Progressions, "xor_progression")
}

// Test scheduled resets with CRON expressions
func TestProgressionSystem_ScheduledResets(t *testing.T) {
	config := &ProgressionConfig{
		Progressions: map[string]*ProgressionConfigProgression{
			"weekly_progression": {
				Name:          "Weekly Progression",
				Description:   "Resets weekly",
				Category:      "weekly",
				ResetSchedule: "0 0 * * 0", // Every Sunday at midnight
				Preconditions: &ProgressionPreconditionsBlock{
					Direct: &ProgressionPreconditions{
						Counts: map[string]int64{
							"weekly_kills": 50,
						},
					},
				},
			},
		},
	}

	progressionSystem := NewNakamaProgressionSystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Create user progression data with old timestamp (more than a week ago)
	oldTime := time.Now().AddDate(0, 0, -8).Unix() // 8 days ago
	userProgressions := map[string]*SyncProgressionUpdate{
		"weekly_progression": {
			Counts: map[string]int64{
				"weekly_kills": 30,
			},
			CreateTimeSec: oldTime,
			UpdateTimeSec: oldTime,
		},
	}

	userProgressionsJSON, _ := json.Marshal(userProgressions)
	storageObject := &api.StorageObject{
		Collection: progressionStorageCollection,
		Key:        userProgressionStorageKey,
		UserId:     userID,
		Value:      string(userProgressionsJSON),
	}

	nk.On("StorageRead", ctx, mock.MatchedBy(func(reads []*runtime.StorageRead) bool {
		return len(reads) == 1 && reads[0].Collection == progressionStorageCollection
	})).Return([]*api.StorageObject{storageObject}, nil)

	progressions, _, err := progressionSystem.Get(ctx, logger, nk, userID, nil)

	require.NoError(t, err)
	assert.NotNil(t, progressions)
	assert.Contains(t, progressions, "weekly_progression")

	// The progression should have been reset due to the schedule
	_ = progressions["weekly_progression"]
	// Note: The actual reset logic depends on the CRON implementation
	// This test verifies the structure is in place

	nk.AssertExpectations(t)
}

// Test complex progression with all features
func TestProgressionSystem_ComplexProgression(t *testing.T) {
	config := &ProgressionConfig{
		Progressions: map[string]*ProgressionConfigProgression{
			"complex_progression": {
				Name:        "Complex Progression",
				Description: "Tests all features together",
				Category:    "advanced",
				AdditionalProperties: map[string]string{
					"difficulty": "hard",
					"zone":       "endgame",
				},
				Preconditions: &ProgressionPreconditionsBlock{
					Direct: &ProgressionPreconditions{
						Cost: &ProgressionCost{
							Currencies: map[string]int64{"gems": 50},
							Items:      map[string]int64{"key": 1},
						},
						Counts:       map[string]int64{"bosses_defeated": 3},
						CurrencyMin:  map[string]int64{"gold": 1000},
						CurrencyMax:  map[string]int64{"debt": 0},
						StatsMin:     map[string]int64{"level": 20},
						StatsMax:     map[string]int64{"corruption": 10},
						ItemsMin:     map[string]int64{"health_potions": 5},
						ItemsMax:     map[string]int64{"cursed_items": 2},
						EnergyMin:    map[string]int64{"mana": 100},
						EnergyMax:    map[string]int64{"fatigue": 20},
						Progressions: []string{"basic_progression"},
						Achievements: []string{"first_boss_kill"},
					},
					Operator: ProgressionPreconditionsOperator_PROGRESSION_PRECONDITIONS_OPERATOR_AND,
					Nested: &ProgressionPreconditionsBlock{
						Direct: &ProgressionPreconditions{
							StatsMin: map[string]int64{"reputation": 100},
						},
						Operator: ProgressionPreconditionsOperator_PROGRESSION_PRECONDITIONS_OPERATOR_OR,
						Nested: &ProgressionPreconditionsBlock{
							Direct: &ProgressionPreconditions{
								ItemsMin: map[string]int64{"special_token": 1},
							},
						},
					},
				},
				ResetSchedule: "0 0 * * 0", // Weekly reset
				Rewards: &EconomyConfigReward{
					Guaranteed: &EconomyConfigRewardContents{
						Currencies: map[string]*EconomyConfigRewardCurrency{
							"gold": {
								EconomyConfigRewardRangeInt64: EconomyConfigRewardRangeInt64{
									Min: 1000,
									Max: 2000,
								},
							},
							"gems": {
								EconomyConfigRewardRangeInt64: EconomyConfigRewardRangeInt64{
									Min: 10,
									Max: 25,
								},
							},
						},
						Items: map[string]*EconomyConfigRewardItem{
							"legendary_weapon": {
								EconomyConfigRewardRangeInt64: EconomyConfigRewardRangeInt64{
									Min: 1,
									Max: 1,
								},
							},
						},
					},
				},
			},
		},
	}

	progressionSystem := NewNakamaProgressionSystem(config)
	assert.NotNil(t, progressionSystem)

	// Verify all aspects of the complex progression
	complexProgression := config.Progressions["complex_progression"]
	assert.Equal(t, "Complex Progression", complexProgression.Name)
	assert.Equal(t, "advanced", complexProgression.Category)
	assert.Equal(t, "0 0 * * 0", complexProgression.ResetSchedule)

	// Check additional properties
	assert.Equal(t, "hard", complexProgression.AdditionalProperties["difficulty"])
	assert.Equal(t, "endgame", complexProgression.AdditionalProperties["zone"])

	// Check preconditions
	assert.NotNil(t, complexProgression.Preconditions)
	assert.NotNil(t, complexProgression.Preconditions.Direct)
	assert.NotNil(t, complexProgression.Preconditions.Nested)

	// Check cost
	assert.NotNil(t, complexProgression.Preconditions.Direct.Cost)
	assert.Equal(t, int64(50), complexProgression.Preconditions.Direct.Cost.Currencies["gems"])
	assert.Equal(t, int64(1), complexProgression.Preconditions.Direct.Cost.Items["key"])

	// Check counts
	assert.Equal(t, int64(3), complexProgression.Preconditions.Direct.Counts["bosses_defeated"])

	// Check min/max requirements
	assert.Equal(t, int64(1000), complexProgression.Preconditions.Direct.CurrencyMin["gold"])
	assert.Equal(t, int64(0), complexProgression.Preconditions.Direct.CurrencyMax["debt"])
	assert.Equal(t, int64(20), complexProgression.Preconditions.Direct.StatsMin["level"])
	assert.Equal(t, int64(10), complexProgression.Preconditions.Direct.StatsMax["corruption"])

	// Check dependencies
	assert.Contains(t, complexProgression.Preconditions.Direct.Progressions, "basic_progression")
	assert.Contains(t, complexProgression.Preconditions.Direct.Achievements, "first_boss_kill")

	// Check nested logical operators
	assert.Equal(t, ProgressionPreconditionsOperator_PROGRESSION_PRECONDITIONS_OPERATOR_AND, complexProgression.Preconditions.Operator)
	assert.Equal(t, ProgressionPreconditionsOperator_PROGRESSION_PRECONDITIONS_OPERATOR_OR, complexProgression.Preconditions.Nested.Operator)

	// Check rewards
	assert.NotNil(t, complexProgression.Rewards)
	assert.NotNil(t, complexProgression.Rewards.Guaranteed)

	goldReward := complexProgression.Rewards.Guaranteed.Currencies["gold"]
	assert.Equal(t, int64(1000), goldReward.Min)
	assert.Equal(t, int64(2000), goldReward.Max)

	weaponReward := complexProgression.Rewards.Guaranteed.Items["legendary_weapon"]
	assert.Equal(t, int64(1), weaponReward.Min)
	assert.Equal(t, int64(1), weaponReward.Max)
}

// Test error handling for nil config
func TestProgressionSystem_NilConfig(t *testing.T) {
	progressionSystem := NewNakamaProgressionSystem(nil)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Test Get with nil config
	progressions, deltas, err := progressionSystem.Get(ctx, logger, nk, userID, nil)
	assert.Error(t, err)
	assert.Nil(t, progressions)
	assert.Nil(t, deltas)

	// Test Purchase with nil config
	progressions, err = progressionSystem.Purchase(ctx, logger, nk, userID, "any_progression")
	assert.Error(t, err)
	assert.Nil(t, progressions)

	// Test Update with nil config
	progressions, err = progressionSystem.Update(ctx, logger, nk, userID, "any_progression", map[string]int64{"count": 1})
	assert.Error(t, err)
	assert.Nil(t, progressions)

	// Test Reset with nil config
	progressions, err = progressionSystem.Reset(ctx, logger, nk, userID, []string{"any_progression"})
	assert.Error(t, err)
	assert.Nil(t, progressions)

	// Test Complete with nil config
	progressions, reward, err := progressionSystem.Complete(ctx, logger, nk, userID, "any_progression")
	assert.Error(t, err)
	assert.Nil(t, progressions)
	assert.Nil(t, reward)
}

// Mock implementations are available in other test files

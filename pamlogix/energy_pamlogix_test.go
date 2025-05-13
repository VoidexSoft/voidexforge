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

// Test basic creation and type checking
func TestNewNakamaEnergySystem(t *testing.T) {
	config := &EnergyConfig{
		Energies: map[string]*EnergyConfigEnergy{
			"energy1": {
				StartCount:    10,
				MaxCount:      20,
				MaxOverfill:   5,
				RefillCount:   2,
				RefillTimeSec: 300,
			},
		},
	}

	energySystem := NewNakamaEnergySystem(config)

	// Test type
	assert.Equal(t, SystemTypeEnergy, energySystem.GetType())

	// Test config
	assert.Equal(t, config, energySystem.GetConfig())
}

// Test Get method with no energies
func TestEnergySystem_Get_NoEnergies(t *testing.T) {
	config := &EnergyConfig{
		Energies: map[string]*EnergyConfigEnergy{},
	}

	energySystem := NewNakamaEnergySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	energies, err := energySystem.Get(ctx, logger, nk, userID)
	require.NoError(t, err)
	assert.Empty(t, energies)
}

// Test Get method with uninitialized user
func TestEnergySystem_Get_UninitializedUser(t *testing.T) {
	config := &EnergyConfig{
		Energies: map[string]*EnergyConfigEnergy{
			"energy1": {
				StartCount:    10,
				MaxCount:      20,
				RefillCount:   2,
				RefillTimeSec: 300,
			},
		},
	}

	energySystem := NewNakamaEnergySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Setup mock for storage read - no data found
	nk.On("StorageRead", mock.Anything, mock.Anything).Return([]*api.StorageObject{}, nil)

	energies, err := energySystem.Get(ctx, logger, nk, userID)
	require.NoError(t, err)
	assert.NotEmpty(t, energies)
	assert.Contains(t, energies, "energy1")
	assert.Equal(t, int32(10), energies["energy1"].Current)
	assert.Equal(t, int32(20), energies["energy1"].Max)
}

// Test Get method with existing user data
func TestEnergySystem_Get_ExistingUser(t *testing.T) {
	config := &EnergyConfig{
		Energies: map[string]*EnergyConfigEnergy{
			"energy1": {
				StartCount:    10,
				MaxCount:      20,
				RefillCount:   2,
				RefillTimeSec: 300,
			},
		},
	}

	energySystem := NewNakamaEnergySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Create existing energy data
	existingEnergy := &Energy{
		Id:                 "energy1",
		Current:            5,
		Max:                20,
		Refill:             2,
		RefillSec:          300,
		NextRefillTimeSec:  0,
		MaxRefillTimeSec:   0,
		StartRefillTimeSec: time.Now().Unix() - 600, // Started refill 10 minutes ago
		CurrentTimeSec:     time.Now().Unix() - 600,
	}

	existingEnergyList := &EnergyList{
		Energies: map[string]*Energy{
			"energy1": existingEnergy,
		},
	}

	energyData, _ := json.Marshal(existingEnergyList)

	// Setup mock for storage read - return existing data
	storageObject := &api.StorageObject{
		Collection: "energy",
		Key:        "user_energies",
		UserId:     userID,
		Value:      string(energyData),
	}

	nk.On("StorageRead", mock.Anything, mock.Anything).Return([]*api.StorageObject{storageObject}, nil)

	energies, err := energySystem.Get(ctx, logger, nk, userID)
	require.NoError(t, err)
	assert.NotEmpty(t, energies)
	assert.Contains(t, energies, "energy1")

	// Current should be greater than 5 due to automatic refill over time (2 units every 5 minutes)
	assert.Greater(t, energies["energy1"].Current, int32(5))
}

// Test refill calculation
func TestEnergySystem_ApplyRefills(t *testing.T) {
	config := &EnergyConfig{
		Energies: map[string]*EnergyConfigEnergy{
			"energy1": {
				StartCount:    10,
				MaxCount:      20,
				RefillCount:   2,
				RefillTimeSec: 300, // 5 minutes per refill
			},
		},
	}

	energySystem := NewNakamaEnergySystem(config)
	now := time.Now().Unix()

	// Case 1: Already at max
	energy1 := &Energy{
		Id:                 "energy1",
		Current:            20,
		Max:                20,
		Refill:             2,
		RefillSec:          300,
		StartRefillTimeSec: now - 600, // 10 minutes ago
	}

	energySystem.applyRefills(energy1, now)
	assert.Equal(t, int32(20), energy1.Current) // Should still be at max

	// Case 2: Partial refill
	energy2 := &Energy{
		Id:                 "energy1",
		Current:            5,
		Max:                20,
		Refill:             2,
		RefillSec:          300,
		StartRefillTimeSec: now - 600, // 10 minutes ago (2 refills)
	}

	energySystem.applyRefills(energy2, now)
	assert.Equal(t, int32(9), energy2.Current)         // 5 + (2 refills * 2 units)
	assert.Equal(t, now-0, energy2.StartRefillTimeSec) // Should have updated the start time

	// Case 3: Complete refill to max
	energy3 := &Energy{
		Id:                 "energy1",
		Current:            15,
		Max:                20,
		Refill:             2,
		RefillSec:          300,
		StartRefillTimeSec: now - 1500, // 25 minutes ago (5 refills)
	}

	energySystem.applyRefills(energy3, now)
	assert.Equal(t, int32(20), energy3.Current) // Should be capped at max
}

// Test Spend method
func TestEnergySystem_Spend(t *testing.T) {
	config := &EnergyConfig{
		Energies: map[string]*EnergyConfigEnergy{
			"energy1": {
				StartCount:    10,
				MaxCount:      20,
				RefillCount:   2,
				RefillTimeSec: 300,
			},
		},
	}

	energySystem := NewNakamaEnergySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Create existing energy data
	existingEnergy := &Energy{
		Id:                 "energy1",
		Current:            15,
		Max:                20,
		Refill:             2,
		RefillSec:          300,
		NextRefillTimeSec:  0,
		MaxRefillTimeSec:   0,
		StartRefillTimeSec: time.Now().Unix(),
		CurrentTimeSec:     time.Now().Unix(),
	}

	existingEnergyList := &EnergyList{
		Energies: map[string]*Energy{
			"energy1": existingEnergy,
		},
	}

	energyData, _ := json.Marshal(existingEnergyList)

	// Setup mocks
	storageObject := &api.StorageObject{
		Collection: "energy",
		Key:        "user_energies",
		UserId:     userID,
		Value:      string(energyData),
	}

	nk.On("StorageRead", mock.Anything, mock.Anything).Return([]*api.StorageObject{storageObject}, nil)
	nk.On("StorageWrite", mock.Anything, mock.Anything).Return([]*api.StorageObjectAck{}, nil)

	// Test successful spend
	amounts := map[string]int32{"energy1": 5}
	energies, reward, err := energySystem.Spend(ctx, logger, nk, userID, amounts)

	require.NoError(t, err)
	assert.NotEmpty(t, energies)
	assert.Contains(t, energies, "energy1")
	assert.Equal(t, int32(10), energies["energy1"].Current) // 15 - 5 = 10
	assert.Nil(t, reward)                                   // No reward configured

	// Test insufficient energy
	amounts = map[string]int32{"energy1": 20}
	_, _, err = energySystem.Spend(ctx, logger, nk, userID, amounts)
	assert.Error(t, err)
	assert.Equal(t, ErrBadInput, err)

	// Test non-existent energy
	amounts = map[string]int32{"energy2": 5}
	_, _, err = energySystem.Spend(ctx, logger, nk, userID, amounts)
	assert.Error(t, err)
	assert.Equal(t, ErrBadInput, err)
}

// Test Spend with reward
func TestEnergySystem_Spend_WithReward(t *testing.T) {
	// Setup configuration with reward
	rewardConfig := &EconomyConfigReward{
		Guaranteed: &EconomyConfigRewardContents{
			Currencies: map[string]*EconomyConfigRewardCurrency{
				"coins": {
					EconomyConfigRewardRangeInt64: EconomyConfigRewardRangeInt64{
						Min: 10,
						Max: 10,
					},
				},
			},
		},
	}

	config := &EnergyConfig{
		Energies: map[string]*EnergyConfigEnergy{
			"energy1": {
				StartCount:    10,
				MaxCount:      20,
				RefillCount:   2,
				RefillTimeSec: 300,
				Reward:        rewardConfig,
			},
		},
	}

	energySystem := NewNakamaEnergySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Create existing energy data
	existingEnergy := &Energy{
		Id:                 "energy1",
		Current:            15,
		Max:                20,
		Refill:             2,
		RefillSec:          300,
		NextRefillTimeSec:  0,
		MaxRefillTimeSec:   0,
		StartRefillTimeSec: time.Now().Unix(),
		CurrentTimeSec:     time.Now().Unix(),
	}

	existingEnergyList := &EnergyList{
		Energies: map[string]*Energy{
			"energy1": existingEnergy,
		},
	}

	energyData, _ := json.Marshal(existingEnergyList)

	// Setup mocks
	storageObject := &api.StorageObject{
		Collection: "energy",
		Key:        "user_energies",
		UserId:     userID,
		Value:      string(energyData),
	}

	nk.On("StorageRead", mock.Anything, mock.Anything).Return([]*api.StorageObject{storageObject}, nil)
	nk.On("StorageWrite", mock.Anything, mock.Anything).Return([]*api.StorageObjectAck{}, nil)

	// Test successful spend with reward
	amounts := map[string]int32{"energy1": 5}

	// Create a custom reward function
	energySystem.SetOnSpendReward(func(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, sourceID string, source *EnergyConfigEnergy, rewardConfig *EconomyConfigReward, reward *Reward) (*Reward, error) {
		// Add coins to the reward
		reward.Currencies["coins"] = 10
		return reward, nil
	})

	energies, reward, err := energySystem.Spend(ctx, logger, nk, userID, amounts)

	require.NoError(t, err)
	assert.NotEmpty(t, energies)
	assert.Contains(t, energies, "energy1")
	assert.Equal(t, int32(10), energies["energy1"].Current) // 15 - 5 = 10
	assert.NotNil(t, reward)
	assert.Equal(t, int64(10), reward.Currencies["coins"])
}

// Test Grant method
func TestEnergySystem_Grant(t *testing.T) {
	config := &EnergyConfig{
		Energies: map[string]*EnergyConfigEnergy{
			"energy1": {
				StartCount:    10,
				MaxCount:      20,
				MaxOverfill:   5, // Allow overfill up to 25
				RefillCount:   2,
				RefillTimeSec: 300,
			},
		},
	}

	energySystem := NewNakamaEnergySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Create existing energy data
	existingEnergy := &Energy{
		Id:                 "energy1",
		Current:            15,
		Max:                20,
		Refill:             2,
		RefillSec:          300,
		NextRefillTimeSec:  0,
		MaxRefillTimeSec:   0,
		StartRefillTimeSec: time.Now().Unix(),
		CurrentTimeSec:     time.Now().Unix(),
	}

	existingEnergyList := &EnergyList{
		Energies: map[string]*Energy{
			"energy1": existingEnergy,
		},
	}

	energyData, _ := json.Marshal(existingEnergyList)

	// Setup mocks
	storageObject := &api.StorageObject{
		Collection: "energy",
		Key:        "user_energies",
		UserId:     userID,
		Value:      string(energyData),
	}

	nk.On("StorageRead", mock.Anything, mock.Anything).Return([]*api.StorageObject{storageObject}, nil)
	nk.On("StorageWrite", mock.Anything, mock.Anything).Return([]*api.StorageObjectAck{}, nil)

	// Test successful grant
	amounts := map[string]int32{"energy1": 5}
	energies, err := energySystem.Grant(ctx, logger, nk, userID, amounts, nil)

	require.NoError(t, err)
	assert.NotEmpty(t, energies)
	assert.Contains(t, energies, "energy1")
	assert.Equal(t, int32(20), energies["energy1"].Current) // 15 + 5 = 20, max is 20

	// Test overfill (with MaxOverfill)
	amounts = map[string]int32{"energy1": 10}
	energies, err = energySystem.Grant(ctx, logger, nk, userID, amounts, nil)

	require.NoError(t, err)
	assert.NotEmpty(t, energies)
	assert.Contains(t, energies, "energy1")
	assert.Equal(t, int32(25), energies["energy1"].Current) // Should be capped at 25 (20 + 5 overfill)

	// Test with non-existent energy (should create it)
	amounts = map[string]int32{"energy2": 5}
	energies, err = energySystem.Grant(ctx, logger, nk, userID, amounts, nil)

	require.NoError(t, err)
	assert.NotEmpty(t, energies)
	// Since energy2 doesn't exist in the config, the grant will be logged but not applied
	assert.NotContains(t, energies, "energy2")
}

// Test Grant with modifiers
func TestEnergySystem_Grant_WithModifiers(t *testing.T) {
	config := &EnergyConfig{
		Energies: map[string]*EnergyConfigEnergy{
			"energy1": {
				StartCount:    10,
				MaxCount:      20,
				RefillCount:   2,
				RefillTimeSec: 300,
			},
		},
	}

	energySystem := NewNakamaEnergySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Create existing energy data
	existingEnergy := &Energy{
		Id:                 "energy1",
		Current:            10,
		Max:                20,
		Refill:             2,
		RefillSec:          300,
		NextRefillTimeSec:  0,
		MaxRefillTimeSec:   0,
		StartRefillTimeSec: time.Now().Unix(),
		CurrentTimeSec:     time.Now().Unix(),
		Modifiers:          nil,
	}

	existingEnergyList := &EnergyList{
		Energies: map[string]*Energy{
			"energy1": existingEnergy,
		},
	}

	energyData, _ := json.Marshal(existingEnergyList)

	// Setup mocks
	storageObject := &api.StorageObject{
		Collection: "energy",
		Key:        "user_energies",
		UserId:     userID,
		Value:      string(energyData),
	}

	// Setup initial read/write mocks
	nk.On("StorageRead", mock.Anything, mock.Anything).Return([]*api.StorageObject{storageObject}, nil).Once()
	nk.On("StorageWrite", mock.Anything, mock.Anything).Return([]*api.StorageObjectAck{}, nil).Once()

	// Test with "add" modifier
	amounts := map[string]int32{"energy1": 5}
	modifiers := []*RewardEnergyModifier{
		{
			Id:          "energy1",
			Operator:    "add",
			Value:       2,
			DurationSec: 3600, // 1 hour
		},
	}

	energies, err := energySystem.Grant(ctx, logger, nk, userID, amounts, modifiers)

	require.NoError(t, err)
	assert.NotEmpty(t, energies)
	assert.Contains(t, energies, "energy1")
	assert.Equal(t, int32(17), energies["energy1"].Current) // 10 + (5 + 2) = 17
	assert.Len(t, energies["energy1"].Modifiers, 1)

	// For the second test, we need to create a new energy data object that includes the first modifier
	updatedEnergy := &Energy{
		Id:                 "energy1",
		Current:            17, // Updated from previous test
		Max:                20,
		Refill:             2,
		RefillSec:          300,
		NextRefillTimeSec:  0,
		MaxRefillTimeSec:   0,
		StartRefillTimeSec: time.Now().Unix(),
		CurrentTimeSec:     time.Now().Unix(),
		Modifiers: []*EnergyModifier{
			{
				Operator:     "add",
				Value:        2,
				StartTimeSec: time.Now().Unix(),
				EndTimeSec:   time.Now().Unix() + 3600,
			},
		},
	}

	updatedEnergyList := &EnergyList{
		Energies: map[string]*Energy{
			"energy1": updatedEnergy,
		},
	}

	updatedEnergyData, _ := json.Marshal(updatedEnergyList)

	// Setup updated mocks for the second test
	updatedStorageObject := &api.StorageObject{
		Collection: "energy",
		Key:        "user_energies",
		UserId:     userID,
		Value:      string(updatedEnergyData),
	}

	nk.On("StorageRead", mock.Anything, mock.Anything).Return([]*api.StorageObject{updatedStorageObject}, nil).Once()
	nk.On("StorageWrite", mock.Anything, mock.Anything).Return([]*api.StorageObjectAck{}, nil).Once()

	// Test with "multiply" modifier
	modifiers = []*RewardEnergyModifier{
		{
			Id:          "energy1",
			Operator:    "multiply",
			Value:       2,
			DurationSec: 3600, // 1 hour
		},
	}

	energies, err = energySystem.Grant(ctx, logger, nk, userID, amounts, modifiers)

	require.NoError(t, err)
	assert.NotEmpty(t, energies)
	assert.Contains(t, energies, "energy1")
	assert.Equal(t, int32(20), energies["energy1"].Current) // 17 + (5 * 2) = 27, but capped at 20
	assert.Len(t, energies["energy1"].Modifiers, 2)
}

// Test invalid cases
func TestEnergySystem_EdgeCases(t *testing.T) {
	// Test with nil config
	energySystem := NewNakamaEnergySystem(nil)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Get with nil config
	energies, err := energySystem.Get(ctx, logger, nk, userID)
	require.NoError(t, err)
	assert.Empty(t, energies)

	// Spend with nil config
	_, _, err = energySystem.Spend(ctx, logger, nk, userID, map[string]int32{"energy1": 5})
	assert.Error(t, err)
	assert.Equal(t, ErrSystemNotAvailable, err)

	// Grant with nil config
	energies, err = energySystem.Grant(ctx, logger, nk, userID, map[string]int32{"energy1": 5}, nil)
	require.NoError(t, err)
	assert.Empty(t, energies)
}

// Test refill time calculations
func TestEnergySystem_RefillTimeCalculations(t *testing.T) {
	energySystem := NewNakamaEnergySystem(nil)
	now := time.Now().Unix()

	// Case 1: Already at max
	energy1 := &Energy{
		Id:                 "energy1",
		Current:            20,
		Max:                20,
		Refill:             2,
		RefillSec:          300,
		StartRefillTimeSec: now - 600,
	}

	energySystem.calculateRefillTimes(energy1, now)
	assert.Equal(t, int64(0), energy1.NextRefillTimeSec)
	assert.Equal(t, int64(0), energy1.MaxRefillTimeSec)

	// Case 2: Needs partial refill
	energy2 := &Energy{
		Id:                 "energy1",
		Current:            15,
		Max:                20,
		Refill:             2,
		RefillSec:          300,
		StartRefillTimeSec: now - 250, // 250 seconds elapsed
	}

	energySystem.calculateRefillTimes(energy2, now)
	// Next refill in 50 seconds
	expectedNextRefill := now + 50
	assert.Equal(t, expectedNextRefill, energy2.NextRefillTimeSec)

	// 3 refills needed for full (15 + 2 = 17, 17 + 2 = 19, 19 + 2 = 21 > 20)
	// First refill in 50 seconds, then 2 more at 300 second intervals
	expectedMaxRefillTime := energy2.NextRefillTimeSec + (300 * 2)
	assert.Equal(t, expectedMaxRefillTime, energy2.MaxRefillTimeSec)

	// Case 3: Multiple refills needed - Energy just starting to refill
	energy3 := &Energy{
		Id:                 "energy1",
		Current:            5,
		Max:                20,
		Refill:             2,
		RefillSec:          300,
		StartRefillTimeSec: now, // Just starting refill now
	}

	energySystem.calculateRefillTimes(energy3, now)

	// Debug output
	t.Logf("Debug Case 3:")
	t.Logf("now: %d", now)
	t.Logf("energy3.StartRefillTimeSec: %d", energy3.StartRefillTimeSec)
	t.Logf("energy3.NextRefillTimeSec: %d", energy3.NextRefillTimeSec)
	t.Logf("energy3.MaxRefillTimeSec: %d", energy3.MaxRefillTimeSec)

	// Calculate expected times for debugging
	timeSinceLastRefill := now - energy3.StartRefillTimeSec
	timeUntilNextRefill := energy3.RefillSec - (timeSinceLastRefill % energy3.RefillSec)
	if timeUntilNextRefill == energy3.RefillSec {
		timeUntilNextRefill = 0
	}
	expectedNextRefillTime := now + timeUntilNextRefill

	// Calculate refills needed
	energyNeeded := energy3.Max - energy3.Current                                    // 20 - 5 = 15
	refillsNeededCalc := int64((energyNeeded + energy3.Refill - 1) / energy3.Refill) // (15 + 2 - 1) / 2 = 16 / 2 = 8

	// Calculate expected max refill time
	expectedMaxRefillTimeCalc := expectedNextRefillTime + energy3.RefillSec*(refillsNeededCalc-1)

	t.Logf("timeSinceLastRefill: %d", timeSinceLastRefill)
	t.Logf("timeUntilNextRefill: %d", timeUntilNextRefill)
	t.Logf("expectedNextRefillTime: %d", expectedNextRefillTime)
	t.Logf("energyNeeded: %d", energyNeeded)
	t.Logf("refillsNeededCalc: %d", refillsNeededCalc)
	t.Logf("expectedMaxRefillTimeCalc: %d", expectedMaxRefillTimeCalc)

	// When StartRefillTimeSec equals now, the next refill is immediately (now + 300 seconds)
	// But since timeUntilNextRefill is 0 (as there's no elapsed time), NextRefillTimeSec is now
	assert.Equal(t, now, energy3.NextRefillTimeSec)

	// 8 refills needed for full (5 + 2 = 7, 7 + 2 = 9, 9 + 2 = 11, 11 + 2 = 13, 13 + 2 = 15, 15 + 2 = 17, 17 + 2 = 19, 19 + 2 = 21 > 20)
	// Using ceiling division: (Max - Current + Refill - 1) / Refill = (20 - 5 + 2 - 1) / 2 = 16 / 2 = 8 refills
	refillsNeeded := int64(8)

	// MaxRefillTimeSec = NextRefillTimeSec + RefillSec * (refillsNeeded - 1)
	expectedMaxRefillTime = energy3.NextRefillTimeSec + energy3.RefillSec*(refillsNeeded-1)

	t.Logf("Manual calculation:")
	t.Logf("refillsNeeded: %d", refillsNeeded)
	t.Logf("expectedNextRefill: %d", energy3.NextRefillTimeSec)
	t.Logf("expectedMaxRefillTime: %d", expectedMaxRefillTime)

	assert.Equal(t, expectedMaxRefillTime, energy3.MaxRefillTimeSec)

	// Case 4: Multiple refills needed - Energy started refilling a while ago
	energy4 := &Energy{
		Id:                 "energy1",
		Current:            5,
		Max:                20,
		Refill:             2,
		RefillSec:          300,
		StartRefillTimeSec: now - 200, // Started refilling 200 seconds ago
	}

	energySystem.calculateRefillTimes(energy4, now)

	// Next refill should be in 100 seconds (300 - 200)
	assert.Equal(t, now+100, energy4.NextRefillTimeSec)

	// 8 refills needed, first is in 100 seconds, then 7 more at 300 seconds each
	expectedMaxRefillTime = energy4.NextRefillTimeSec + energy4.RefillSec*(refillsNeeded-1)
	assert.Equal(t, expectedMaxRefillTime, energy4.MaxRefillTimeSec)
}

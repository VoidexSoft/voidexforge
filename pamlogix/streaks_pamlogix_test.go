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

// Mock logger for tests
type mockStreaksLogger struct{}

func (l *mockStreaksLogger) Debug(format string, v ...interface{})                   {}
func (l *mockStreaksLogger) Info(format string, v ...interface{})                    {}
func (l *mockStreaksLogger) Warn(format string, v ...interface{})                    {}
func (l *mockStreaksLogger) Error(format string, v ...interface{})                   {}
func (l *mockStreaksLogger) WithField(key string, value interface{}) runtime.Logger  { return l }
func (l *mockStreaksLogger) WithFields(fields map[string]interface{}) runtime.Logger { return l }
func (l *mockStreaksLogger) Fields() map[string]interface{}                          { return map[string]interface{}{} }

// Test basic system creation
func TestNakamaStreaksSystem_Creation(t *testing.T) {
	config := &StreaksConfig{
		Streaks: map[string]*StreaksConfigStreak{
			"daily_login": {
				Name:        "Daily Login",
				Description: "Login every day",
				Count:       0,
				MaxCount:    30,
			},
		},
	}

	system := NewNakamaStreaksSystem(config)
	assert.NotNil(t, system)
	assert.Equal(t, SystemTypeStreaks, system.GetType())
	assert.Equal(t, config, system.GetConfig())
}

// Test List method with no existing data
func TestNakamaStreaksSystem_List_NewUser(t *testing.T) {
	config := &StreaksConfig{
		Streaks: map[string]*StreaksConfigStreak{
			"daily_login": {
				Name:        "Daily Login",
				Description: "Login every day",
				Count:       0,
				MaxCount:    30,
				Rewards: []*StreaksConfigStreakReward{
					{
						CountMin: 1,
						CountMax: 5,
						Reward: &EconomyConfigReward{
							Guaranteed: &EconomyConfigRewardContents{
								Currencies: map[string]*EconomyConfigRewardCurrency{
									"coins": {
										EconomyConfigRewardRangeInt64: EconomyConfigRewardRangeInt64{
											Min: 100,
											Max: 100,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	system := NewNakamaStreaksSystem(config)
	logger := &mockStreaksLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "test_user"

	// Mock storage read - no existing data
	nk.On("StorageRead", ctx, []*runtime.StorageRead{
		{Collection: streaksStorageCollection, Key: userStreaksStorageKey, UserID: userID},
	}).Return([]*api.StorageObject{}, nil)

	streaks, err := system.List(ctx, logger, nk, userID)
	require.NoError(t, err)
	assert.NotEmpty(t, streaks)
	assert.Contains(t, streaks, "daily_login")

	streak := streaks["daily_login"]
	assert.Equal(t, "daily_login", streak.Id)
	assert.Equal(t, "Daily Login", streak.Name)
	assert.Equal(t, "Login every day", streak.Description)
	assert.Equal(t, int64(0), streak.Count)
	assert.Equal(t, int64(30), streak.MaxCount)
	assert.True(t, streak.CanUpdate)
	assert.False(t, streak.CanClaim)
	assert.Len(t, streak.Rewards, 1)

	nk.AssertExpectations(t)
}

// Test List method with existing data
func TestNakamaStreaksSystem_List_ExistingUser(t *testing.T) {
	config := &StreaksConfig{
		Streaks: map[string]*StreaksConfigStreak{
			"daily_login": {
				Name:     "Daily Login",
				Count:    0,
				MaxCount: 30,
				Rewards: []*StreaksConfigStreakReward{
					{
						CountMin: 1,
						CountMax: 5,
						Reward: &EconomyConfigReward{
							Guaranteed: &EconomyConfigRewardContents{
								Currencies: map[string]*EconomyConfigRewardCurrency{
									"coins": {
										EconomyConfigRewardRangeInt64: EconomyConfigRewardRangeInt64{
											Min: 100,
											Max: 100,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	system := NewNakamaStreaksSystem(config)
	logger := &mockStreaksLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "test_user"

	// Create existing streak data
	existingStreaks := &SyncStreaks{
		Updates: map[string]*SyncStreakUpdate{
			"daily_login": {
				Count:             5,
				CountCurrentReset: 2,
				ClaimCount:        3,
				CreateTimeSec:     time.Now().Unix() - 86400,
				UpdateTimeSec:     time.Now().Unix() - 3600,
				ClaimTimeSec:      time.Now().Unix() - 7200,
				ClaimedRewards:    []*StreakReward{},
			},
		},
		Resets: []string{},
	}

	streaksData, _ := json.Marshal(existingStreaks)
	storageObject := &api.StorageObject{
		Collection: streaksStorageCollection,
		Key:        userStreaksStorageKey,
		UserId:     userID,
		Value:      string(streaksData),
	}

	nk.On("StorageRead", ctx, []*runtime.StorageRead{
		{Collection: streaksStorageCollection, Key: userStreaksStorageKey, UserID: userID},
	}).Return([]*api.StorageObject{storageObject}, nil)

	streaks, err := system.List(ctx, logger, nk, userID)
	require.NoError(t, err)
	assert.NotEmpty(t, streaks)
	assert.Contains(t, streaks, "daily_login")

	streak := streaks["daily_login"]
	assert.Equal(t, int64(5), streak.Count)
	assert.Equal(t, int64(2), streak.CountCurrentReset)
	assert.Equal(t, int64(3), streak.ClaimCount)
	assert.True(t, streak.CanUpdate)
	assert.True(t, streak.CanClaim) // Should be claimable since count > claim_count

	nk.AssertExpectations(t)
}

// Test Update method
func TestNakamaStreaksSystem_Update(t *testing.T) {
	config := &StreaksConfig{
		Streaks: map[string]*StreaksConfigStreak{
			"daily_login": {
				Name:     "Daily Login",
				Count:    0,
				MaxCount: 30,
			},
		},
	}

	system := NewNakamaStreaksSystem(config)
	logger := &mockStreaksLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "test_user"

	// Mock storage read - no existing data
	nk.On("StorageRead", ctx, []*runtime.StorageRead{
		{Collection: streaksStorageCollection, Key: userStreaksStorageKey, UserID: userID},
	}).Return([]*api.StorageObject{}, nil)

	// Mock storage write
	nk.On("StorageWrite", ctx, mock.MatchedBy(func(writes []*runtime.StorageWrite) bool {
		return len(writes) == 1 && writes[0].Collection == streaksStorageCollection
	})).Return([]*api.StorageObjectAck{}, nil)

	updates := map[string]int64{"daily_login": 1}
	streaks, err := system.Update(ctx, logger, nk, userID, updates)

	require.NoError(t, err)
	assert.NotEmpty(t, streaks)
	assert.Contains(t, streaks, "daily_login")

	streak := streaks["daily_login"]
	assert.Equal(t, int64(1), streak.Count)
	assert.Equal(t, int64(1), streak.CountCurrentReset)

	nk.AssertExpectations(t)
}

// Test Update with max count limits
func TestNakamaStreaksSystem_Update_MaxCountLimits(t *testing.T) {
	config := &StreaksConfig{
		Streaks: map[string]*StreaksConfigStreak{
			"daily_login": {
				Name:                 "Daily Login",
				Count:                0,
				MaxCount:             10,
				MaxCountCurrentReset: 3,
			},
		},
	}

	system := NewNakamaStreaksSystem(config)
	logger := &mockStreaksLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "test_user"

	// Create existing streak data near limits
	existingStreaks := &SyncStreaks{
		Updates: map[string]*SyncStreakUpdate{
			"daily_login": {
				Count:             9,
				CountCurrentReset: 2,
				CreateTimeSec:     time.Now().Unix(),
				UpdateTimeSec:     time.Now().Unix(),
			},
		},
		Resets: []string{},
	}

	streaksData, _ := json.Marshal(existingStreaks)
	storageObject := &api.StorageObject{
		Collection: streaksStorageCollection,
		Key:        userStreaksStorageKey,
		UserId:     userID,
		Value:      string(streaksData),
	}

	nk.On("StorageRead", ctx, []*runtime.StorageRead{
		{Collection: streaksStorageCollection, Key: userStreaksStorageKey, UserID: userID},
	}).Return([]*api.StorageObject{storageObject}, nil)

	nk.On("StorageWrite", ctx, mock.Anything).Return([]*api.StorageObjectAck{}, nil)

	// Update with amount that would exceed limits
	updates := map[string]int64{"daily_login": 5}
	streaks, err := system.Update(ctx, logger, nk, userID, updates)

	require.NoError(t, err)
	streak := streaks["daily_login"]
	assert.Equal(t, int64(10), streak.Count)            // Capped at MaxCount
	assert.Equal(t, int64(3), streak.CountCurrentReset) // Capped at MaxCountCurrentReset

	nk.AssertExpectations(t)
}

// Test Claim method
func TestNakamaStreaksSystem_Claim(t *testing.T) {
	config := &StreaksConfig{
		Streaks: map[string]*StreaksConfigStreak{
			"daily_login": {
				Name:     "Daily Login",
				Count:    0,
				MaxCount: 30,
				Rewards: []*StreaksConfigStreakReward{
					{
						CountMin: 1,
						CountMax: 5,
						Reward: &EconomyConfigReward{
							Guaranteed: &EconomyConfigRewardContents{
								Currencies: map[string]*EconomyConfigRewardCurrency{
									"coins": {
										EconomyConfigRewardRangeInt64: EconomyConfigRewardRangeInt64{
											Min: 100,
											Max: 100,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	system := NewNakamaStreaksSystem(config)

	// Mock economy system
	mockEconomy := &mockEconomySystem{}
	mockPamlogix := &mockPamlogix{economy: mockEconomy}
	system.SetPamlogix(mockPamlogix)

	logger := &mockStreaksLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "test_user"

	// Create existing streak data with claimable progress
	existingStreaks := &SyncStreaks{
		Updates: map[string]*SyncStreakUpdate{
			"daily_login": {
				Count:             5,
				CountCurrentReset: 2,
				ClaimCount:        0,
				CreateTimeSec:     time.Now().Unix(),
				UpdateTimeSec:     time.Now().Unix(),
				ClaimedRewards:    []*StreakReward{},
			},
		},
		Resets: []string{},
	}

	streaksData, _ := json.Marshal(existingStreaks)
	storageObject := &api.StorageObject{
		Collection: streaksStorageCollection,
		Key:        userStreaksStorageKey,
		UserId:     userID,
		Value:      string(streaksData),
	}

	nk.On("StorageRead", ctx, []*runtime.StorageRead{
		{Collection: streaksStorageCollection, Key: userStreaksStorageKey, UserID: userID},
	}).Return([]*api.StorageObject{storageObject}, nil)

	nk.On("StorageWrite", ctx, mock.Anything).Return([]*api.StorageObjectAck{}, nil)

	streakIDs := []string{"daily_login"}
	streaks, err := system.Claim(ctx, logger, nk, userID, streakIDs)

	require.NoError(t, err)
	assert.NotEmpty(t, streaks)

	if len(streaks) > 0 {
		streak := streaks["daily_login"]
		assert.Equal(t, int64(5), streak.ClaimCount)
		assert.Len(t, streak.ClaimedRewards, 1)
		// Verify that the reward was processed by checking the lastRolled field
		assert.NotNil(t, mockEconomy.lastRolled)
	}

	nk.AssertExpectations(t)
}

// Test Reset method
func TestNakamaStreaksSystem_Reset(t *testing.T) {
	config := &StreaksConfig{
		Streaks: map[string]*StreaksConfigStreak{
			"daily_login": {
				Name:     "Daily Login",
				Count:    0,
				MaxCount: 30,
			},
		},
	}

	system := NewNakamaStreaksSystem(config)
	logger := &mockStreaksLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "test_user"

	// Create existing streak data
	existingStreaks := &SyncStreaks{
		Updates: map[string]*SyncStreakUpdate{
			"daily_login": {
				Count:             10,
				CountCurrentReset: 5,
				ClaimCount:        8,
				CreateTimeSec:     time.Now().Unix(),
				UpdateTimeSec:     time.Now().Unix(),
				ClaimedRewards:    []*StreakReward{},
			},
		},
		Resets: []string{},
	}

	streaksData, _ := json.Marshal(existingStreaks)
	storageObject := &api.StorageObject{
		Collection: streaksStorageCollection,
		Key:        userStreaksStorageKey,
		UserId:     userID,
		Value:      string(streaksData),
	}

	nk.On("StorageRead", ctx, []*runtime.StorageRead{
		{Collection: streaksStorageCollection, Key: userStreaksStorageKey, UserID: userID},
	}).Return([]*api.StorageObject{storageObject}, nil)

	nk.On("StorageWrite", ctx, mock.Anything).Return([]*api.StorageObjectAck{}, nil)

	streakIDs := []string{"daily_login"}
	streaks, err := system.Reset(ctx, logger, nk, userID, streakIDs)

	require.NoError(t, err)
	assert.NotEmpty(t, streaks)
	streak := streaks["daily_login"]
	assert.Equal(t, int64(0), streak.Count)
	assert.Equal(t, int64(0), streak.CountCurrentReset)
	assert.Equal(t, int64(0), streak.ClaimCount)
	assert.Empty(t, streak.ClaimedRewards)

	nk.AssertExpectations(t)
}

// Test disabled streaks are filtered out
func TestNakamaStreaksSystem_List_DisabledStreaks(t *testing.T) {
	config := &StreaksConfig{
		Streaks: map[string]*StreaksConfigStreak{
			"daily_login": {
				Name:     "Daily Login",
				Count:    0,
				MaxCount: 30,
				Disabled: false,
			},
			"weekly_challenge": {
				Name:     "Weekly Challenge",
				Count:    0,
				MaxCount: 10,
				Disabled: true, // This should be filtered out
			},
		},
	}

	system := NewNakamaStreaksSystem(config)
	logger := &mockStreaksLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "test_user"

	nk.On("StorageRead", ctx, mock.Anything).Return([]*api.StorageObject{}, nil)

	streaks, err := system.List(ctx, logger, nk, userID)
	require.NoError(t, err)
	assert.Contains(t, streaks, "daily_login")
	assert.NotContains(t, streaks, "weekly_challenge")

	nk.AssertExpectations(t)
}

// Test time window filtering
func TestNakamaStreaksSystem_List_TimeWindows(t *testing.T) {
	now := time.Now().Unix()
	config := &StreaksConfig{
		Streaks: map[string]*StreaksConfigStreak{
			"current_event": {
				Name:         "Current Event",
				StartTimeSec: now - 3600, // Started 1 hour ago
				EndTimeSec:   now + 3600, // Ends in 1 hour
			},
			"future_event": {
				Name:         "Future Event",
				StartTimeSec: now + 7200,  // Starts in 2 hours
				EndTimeSec:   now + 10800, // Ends in 3 hours
			},
			"past_event": {
				Name:         "Past Event",
				StartTimeSec: now - 7200, // Started 2 hours ago
				EndTimeSec:   now - 3600, // Ended 1 hour ago
			},
		},
	}

	system := NewNakamaStreaksSystem(config)
	logger := &mockStreaksLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "test_user"

	nk.On("StorageRead", ctx, mock.Anything).Return([]*api.StorageObject{}, nil)

	streaks, err := system.List(ctx, logger, nk, userID)
	require.NoError(t, err)

	// Only current_event should be included
	assert.Contains(t, streaks, "current_event")
	assert.NotContains(t, streaks, "future_event")
	assert.NotContains(t, streaks, "past_event")

	nk.AssertExpectations(t)
}

// Test cron reset functionality
func TestNakamaStreaksSystem_CronReset(t *testing.T) {
	config := &StreaksConfig{
		Streaks: map[string]*StreaksConfigStreak{
			"daily_login": {
				Name:          "Daily Login",
				Count:         0,
				MaxCount:      30,
				ResetCronexpr: "0 0 * * *", // Daily at midnight
			},
		},
	}

	system := NewNakamaStreaksSystem(config)
	logger := &mockStreaksLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "test_user"

	// Create streak data that should be reset
	yesterday := time.Now().AddDate(0, 0, -1)
	existingStreaks := &SyncStreaks{
		Updates: map[string]*SyncStreakUpdate{
			"daily_login": {
				Count:             5,
				CountCurrentReset: 3,
				CreateTimeSec:     yesterday.Unix(),
				UpdateTimeSec:     yesterday.Unix(),
			},
		},
		Resets: []string{},
	}

	streaksData, _ := json.Marshal(existingStreaks)
	storageObject := &api.StorageObject{
		Collection: streaksStorageCollection,
		Key:        userStreaksStorageKey,
		UserId:     userID,
		Value:      string(streaksData),
	}

	nk.On("StorageRead", ctx, []*runtime.StorageRead{
		{Collection: streaksStorageCollection, Key: userStreaksStorageKey, UserID: userID},
	}).Return([]*api.StorageObject{storageObject}, nil)

	streaks, err := system.List(ctx, logger, nk, userID)
	require.NoError(t, err)

	streak := streaks["daily_login"]
	// CountCurrentReset should be reset to 0 due to cron schedule
	assert.Equal(t, int64(0), streak.CountCurrentReset)

	nk.AssertExpectations(t)
}

// Test error handling - invalid config
func TestNakamaStreaksSystem_List_NoConfig(t *testing.T) {
	system := NewNakamaStreaksSystem(nil)
	logger := &mockStreaksLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "test_user"

	streaks, err := system.List(ctx, logger, nk, userID)
	assert.Error(t, err)
	assert.Nil(t, streaks)
	assert.Contains(t, err.Error(), "streaks config not loaded")
}

// Test error handling - storage read failure
func TestNakamaStreaksSystem_List_StorageError(t *testing.T) {
	config := &StreaksConfig{
		Streaks: map[string]*StreaksConfigStreak{
			"daily_login": {Name: "Daily Login"},
		},
	}

	system := NewNakamaStreaksSystem(config)
	logger := &mockStreaksLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "test_user"

	// Mock storage read failure
	nk.On("StorageRead", ctx, mock.Anything).Return([]*api.StorageObject{}, assert.AnError)

	streaks, err := system.List(ctx, logger, nk, userID)
	assert.Error(t, err)
	assert.Nil(t, streaks)

	nk.AssertExpectations(t)
}

// Test SetOnClaimReward callback
func TestNakamaStreaksSystem_SetOnClaimReward(t *testing.T) {
	config := &StreaksConfig{}
	system := NewNakamaStreaksSystem(config)

	callbackCalled := false
	callback := func(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, sourceID string, source *StreaksConfigStreak, rewardConfig *EconomyConfigReward, reward *Reward) (*Reward, error) {
		callbackCalled = true
		return reward, nil
	}

	system.SetOnClaimReward(callback)
	assert.NotNil(t, system.onClaimReward)

	// Test that callback is stored (we can't easily test execution without a full integration test)
	assert.False(t, callbackCalled) // Not called yet
}

// Test Update with non-existent streak ID
func TestNakamaStreaksSystem_Update_NonExistentStreak(t *testing.T) {
	config := &StreaksConfig{
		Streaks: map[string]*StreaksConfigStreak{
			"daily_login": {Name: "Daily Login"},
		},
	}

	system := NewNakamaStreaksSystem(config)
	logger := &mockStreaksLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "test_user"

	nk.On("StorageRead", ctx, mock.Anything).Return([]*api.StorageObject{}, nil)

	updates := map[string]int64{"non_existent": 1}
	streaks, err := system.Update(ctx, logger, nk, userID, updates)

	require.NoError(t, err)
	assert.Empty(t, streaks) // No streaks should be returned for non-existent IDs

	nk.AssertExpectations(t)
}

// Test Claim with no available rewards
func TestNakamaStreaksSystem_Claim_NoAvailableRewards(t *testing.T) {
	config := &StreaksConfig{
		Streaks: map[string]*StreaksConfigStreak{
			"daily_login": {
				Name:     "Daily Login",
				Count:    0,
				MaxCount: 30,
				// No rewards configured
			},
		},
	}

	system := NewNakamaStreaksSystem(config)

	// Mock economy system
	mockEconomy := &mockEconomySystem{}
	mockPamlogix := &mockPamlogix{economy: mockEconomy}
	system.SetPamlogix(mockPamlogix)

	logger := &mockStreaksLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "test_user"

	// Create existing streak data
	existingStreaks := &SyncStreaks{
		Updates: map[string]*SyncStreakUpdate{
			"daily_login": {
				Count:         5,
				ClaimCount:    0,
				CreateTimeSec: time.Now().Unix(),
				UpdateTimeSec: time.Now().Unix(),
			},
		},
		Resets: []string{},
	}

	streaksData, _ := json.Marshal(existingStreaks)
	storageObject := &api.StorageObject{
		Collection: streaksStorageCollection,
		Key:        userStreaksStorageKey,
		UserId:     userID,
		Value:      string(streaksData),
	}

	nk.On("StorageRead", ctx, []*runtime.StorageRead{
		{Collection: streaksStorageCollection, Key: userStreaksStorageKey, UserID: userID},
	}).Return([]*api.StorageObject{storageObject}, nil)

	streakIDs := []string{"daily_login"}
	streaks, err := system.Claim(ctx, logger, nk, userID, streakIDs)

	require.NoError(t, err)
	// When there are no available rewards, the streak should not be returned
	assert.Empty(t, streaks)

	nk.AssertExpectations(t)
}

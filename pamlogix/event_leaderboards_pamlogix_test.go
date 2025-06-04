package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/heroiclabs/nakama-common/api"
	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Test configuration for event leaderboards
func getTestEventLeaderboardsConfig() *EventLeaderboardsConfig {
	now := time.Now().Unix()
	return &EventLeaderboardsConfig{
		EventLeaderboards: map[string]*EventLeaderboardsConfigLeaderboard{
			"test_event": {
				Name:         "Test Event",
				Description:  "A test event leaderboard",
				Category:     "test",
				Ascending:    false,
				Operator:     "best",
				CohortSize:   10,
				Tiers:        3,
				StartTimeSec: now - 3600, // Started 1 hour ago
				EndTimeSec:   now + 3600, // Ends in 1 hour
				MaxRerolls:   3,
				RerollCost: &EconomyConfigReward{
					Guaranteed: &EconomyConfigRewardContents{
						Currencies: map[string]*EconomyConfigRewardCurrency{
							"gems": {EconomyConfigRewardRangeInt64{Min: 50, Max: 50, Multiple: 1}},
						},
					},
				},
				ParticipationCost: &EconomyConfigReward{
					Guaranteed: &EconomyConfigRewardContents{
						Currencies: map[string]*EconomyConfigRewardCurrency{
							"coins": {EconomyConfigRewardRangeInt64{Min: 100, Max: 100, Multiple: 1}},
						},
					},
				},
				TargetScore: 1000,
				WinnerCount: 3,
				RewardTiers: map[string][]*EventLeaderboardsConfigLeaderboardRewardTier{
					"0": {
						{
							Name:    "Winner",
							RankMin: 1,
							RankMax: 3,
							Reward: &EconomyConfigReward{
								Guaranteed: &EconomyConfigRewardContents{
									Currencies: map[string]*EconomyConfigRewardCurrency{
										"coins": {EconomyConfigRewardRangeInt64{Min: 1000, Max: 1000, Multiple: 1}},
									},
								},
							},
							TierChange: 1,
						},
					},
				},
				ChangeZones: map[string]*EventLeaderboardsConfigChangeZone{
					"0": {
						Promotion:  0.2,
						Demotion:   0.3,
						DemoteIdle: true,
					},
				},
				MaxIdleTierDrop: 1,
			},
			"ended_event": {
				Name:         "Ended Event",
				Description:  "An ended event for testing claims",
				Category:     "test",
				StartTimeSec: now - 7200, // Started 2 hours ago
				EndTimeSec:   now - 3600, // Ended 1 hour ago
				CohortSize:   5,
				RewardTiers: map[string][]*EventLeaderboardsConfigLeaderboardRewardTier{
					"0": {
						{
							Name:    "Winner",
							RankMin: 1,
							RankMax: 1,
							Reward: &EconomyConfigReward{
								Guaranteed: &EconomyConfigRewardContents{
									Currencies: map[string]*EconomyConfigRewardCurrency{
										"coins": {EconomyConfigRewardRangeInt64{Min: 500, Max: 500, Multiple: 1}},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Test helper to create a mock Pamlogix instance
func createTestMockPamlogix(t *testing.T) *MockPamlogix {
	mockPamlogix := &MockPamlogix{}

	// Create mock economy system
	mockEconomy := &MockEconomySystem{}
	mockPamlogix.On("GetEconomySystem").Return(mockEconomy)

	// Set up common mock expectations for economy system
	mockEconomy.On("RewardConvert", mock.Anything).Return(&EconomyConfigReward{})
	mockEconomy.On("UnmarshalWallet", mock.Anything).Return(map[string]int64{"coins": 200, "gems": 100}, nil)

	// Add mock expectation for Grant method (used for deducting participation costs)
	mockEconomy.On("Grant",
		mock.Anything, // context (use Anything to avoid type issues)
		mock.AnythingOfType("*pamlogix.mockLogger"),
		mock.AnythingOfType("*pamlogix.MockNakamaModule"),
		mock.AnythingOfType("string"),                     // userId
		mock.AnythingOfType("map[string]int64"),           // currencies (negative for deduction)
		mock.AnythingOfType("map[string]int64"),           // items
		mock.AnythingOfType("[]*pamlogix.RewardModifier"), // modifiers
		mock.AnythingOfType("map[string]interface {}"),    // metadata
	).Return(map[string]int64{"coins": 100}, []*ActiveRewardModifier{}, int64(1234567890), nil)

	// Add mock expectation for RewardRoll method (used for claiming rewards)
	mockEconomy.On("RewardRoll",
		mock.Anything, // context
		mock.AnythingOfType("*pamlogix.mockLogger"),
		mock.AnythingOfType("*pamlogix.MockNakamaModule"),
		mock.AnythingOfType("string"),                        // userId
		mock.AnythingOfType("*pamlogix.EconomyConfigReward"), // rewardConfig
	).Return(&Reward{
		Currencies: map[string]int64{"coins": 500},
		Items:      map[string]int64{},
	}, nil)

	// Add mock expectation for RewardGrant method (used for granting claimed rewards)
	mockEconomy.On("RewardGrant",
		mock.Anything, // context
		mock.AnythingOfType("*pamlogix.mockLogger"),
		mock.AnythingOfType("*pamlogix.MockNakamaModule"),
		mock.AnythingOfType("string"),                  // userId
		mock.AnythingOfType("*pamlogix.Reward"),        // reward
		mock.AnythingOfType("map[string]interface {}"), // metadata
		mock.AnythingOfType("bool"),                    // ignoreLimits
	).Return(map[string]*InventoryItem{}, map[string]*InventoryItem{}, map[string]int64{}, nil)

	// Create mock inventory system
	mockInventory := NewNakamaInventorySystem(&InventoryConfig{})
	mockPamlogix.inventorySystem = mockInventory

	return mockPamlogix
}

func TestListEventLeaderboard_ActiveEvents(t *testing.T) {
	config := getTestEventLeaderboardsConfig()
	system := NewNakamaEventLeaderboardsSystem(config)
	mockPamlogix := createTestMockPamlogix(t)
	system.SetPamlogix(mockPamlogix)

	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Mock storage read for user state (empty initially)
	nk.On("StorageRead", ctx, mock.Anything).Return([]*api.StorageObject{}, nil)

	events, err := system.ListEventLeaderboard(ctx, logger, nk, userID, false, nil)
	require.NoError(t, err)
	assert.Len(t, events, 2) // Both test_event and ended_event should be returned

	// Check active event
	var activeEvent *EventLeaderboard
	for _, event := range events {
		if event.Id == "test_event" {
			activeEvent = event
			break
		}
	}
	require.NotNil(t, activeEvent)
	assert.True(t, activeEvent.IsActive)
	assert.True(t, activeEvent.CanRoll)
	assert.False(t, activeEvent.CanClaim)

	nk.AssertExpectations(t)
}

func TestRollEventLeaderboard_FirstTime(t *testing.T) {
	config := getTestEventLeaderboardsConfig()
	system := NewNakamaEventLeaderboardsSystem(config)
	mockPamlogix := createTestMockPamlogix(t)
	system.SetPamlogix(mockPamlogix)

	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Mock storage read for user state (empty initially)
	nk.On("StorageRead", ctx, mock.Anything).Return([]*api.StorageObject{}, nil)

	// Mock storage list for finding available cohorts (empty initially)
	nk.On("StorageList", ctx, "", "", eventLeaderboardsStorageCollection, 100, "").Return([]*api.StorageObject{}, "", nil)

	// Mock account get for affordability check
	account := &api.Account{
		Wallet: `{"coins": 200, "gems": 100}`,
	}
	nk.On("AccountGetId", ctx, userID).Return(account, nil)

	// Mock storage write for user state
	nk.On("StorageWrite", ctx, mock.Anything).Return([]*api.StorageObjectAck{}, nil)

	// Mock storage write for cohort creation
	nk.On("StorageWrite", ctx, mock.MatchedBy(func(writes []*runtime.StorageWrite) bool {
		return len(writes) == 1 && writes[0].Collection == eventLeaderboardsStorageCollection
	})).Return([]*api.StorageObjectAck{}, nil)

	// Mock leaderboard creation
	nk.On("LeaderboardCreate", ctx, mock.AnythingOfType("string"), false, "desc", "best", "", mock.Anything, false).Return(nil)

	// Mock leaderboard records list (empty for new cohort)
	nk.On("LeaderboardRecordsList", ctx, mock.AnythingOfType("string"), mock.Anything, 100, "", int64(0)).Return(
		[]*api.LeaderboardRecord{}, []*api.LeaderboardRecord{}, "", "", nil)

	eventLeaderboard, err := system.RollEventLeaderboard(ctx, logger, nk, userID, "test_event", nil, nil)
	require.NoError(t, err)
	assert.NotNil(t, eventLeaderboard)
	assert.Equal(t, "test_event", eventLeaderboard.Id)
	assert.NotEmpty(t, eventLeaderboard.CohortId)
	assert.Equal(t, int32(0), eventLeaderboard.Tier)

	nk.AssertExpectations(t)
}

func TestRollEventLeaderboard_Reroll(t *testing.T) {
	config := getTestEventLeaderboardsConfig()
	system := NewNakamaEventLeaderboardsSystem(config)
	mockPamlogix := createTestMockPamlogix(t)
	system.SetPamlogix(mockPamlogix)

	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Mock existing user state with a cohort
	existingState := &EventLeaderboardUserState{
		EventLeaderboards: map[string]*EventLeaderboardUserEventState{
			"test_event": {
				CohortID:    "existing_cohort",
				Tier:        1,
				RerollCount: 1,
			},
		},
	}
	stateData, _ := json.Marshal(existingState)
	nk.On("StorageRead", ctx, mock.Anything).Return([]*api.StorageObject{
		{Value: string(stateData)},
	}, nil)

	// Mock storage list for finding available cohorts (empty initially)
	nk.On("StorageList", ctx, "", "", eventLeaderboardsStorageCollection, 100, "").Return([]*api.StorageObject{}, "", nil)

	// Mock account get for affordability check
	account := &api.Account{
		Wallet: `{"coins": 200, "gems": 100}`,
	}
	nk.On("AccountGetId", ctx, userID).Return(account, nil)

	// Mock storage write for user state
	nk.On("StorageWrite", ctx, mock.Anything).Return([]*api.StorageObjectAck{}, nil)

	// Mock storage write for cohort creation
	nk.On("StorageWrite", ctx, mock.MatchedBy(func(writes []*runtime.StorageWrite) bool {
		return len(writes) == 1 && writes[0].Collection == eventLeaderboardsStorageCollection
	})).Return([]*api.StorageObjectAck{}, nil)

	// Mock leaderboard creation
	nk.On("LeaderboardCreate", ctx, mock.AnythingOfType("string"), false, "desc", "best", "", mock.Anything, false).Return(nil)

	// Mock leaderboard records list (empty for new cohort)
	nk.On("LeaderboardRecordsList", ctx, mock.AnythingOfType("string"), mock.Anything, 100, "", int64(0)).Return(
		[]*api.LeaderboardRecord{}, []*api.LeaderboardRecord{}, "", "", nil)

	eventLeaderboard, err := system.RollEventLeaderboard(ctx, logger, nk, userID, "test_event", nil, nil)
	require.NoError(t, err)
	assert.NotNil(t, eventLeaderboard)
	assert.NotEqual(t, "existing_cohort", eventLeaderboard.CohortId) // Should get new cohort

	nk.AssertExpectations(t)
}

func TestRollEventLeaderboard_RerollLimitExceeded(t *testing.T) {
	config := getTestEventLeaderboardsConfig()
	system := NewNakamaEventLeaderboardsSystem(config)
	mockPamlogix := createTestMockPamlogix(t)
	system.SetPamlogix(mockPamlogix)

	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Mock existing user state with max rerolls used
	existingState := &EventLeaderboardUserState{
		EventLeaderboards: map[string]*EventLeaderboardUserEventState{
			"test_event": {
				CohortID:    "existing_cohort",
				Tier:        1,
				RerollCount: 3, // Max rerolls reached
			},
		},
	}
	stateData, _ := json.Marshal(existingState)
	nk.On("StorageRead", ctx, mock.Anything).Return([]*api.StorageObject{
		{Value: string(stateData)},
	}, nil)

	eventLeaderboard, err := system.RollEventLeaderboard(ctx, logger, nk, userID, "test_event", nil, nil)
	assert.Error(t, err)
	assert.Equal(t, ErrBadInput, err)
	assert.Nil(t, eventLeaderboard)

	nk.AssertExpectations(t)
}

func TestUpdateEventLeaderboard_TargetScoreAchievement(t *testing.T) {
	config := getTestEventLeaderboardsConfig()
	system := NewNakamaEventLeaderboardsSystem(config)
	mockPamlogix := createTestMockPamlogix(t)
	system.SetPamlogix(mockPamlogix)

	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Mock existing user state with a cohort
	existingState := &EventLeaderboardUserState{
		EventLeaderboards: map[string]*EventLeaderboardUserEventState{
			"test_event": {
				CohortID: "test_cohort",
				Tier:     0,
			},
		},
	}
	stateData, _ := json.Marshal(existingState)
	nk.On("StorageRead", ctx, mock.Anything).Return([]*api.StorageObject{
		{Value: string(stateData)},
	}, nil)

	// Mock leaderboard record write
	record := &api.LeaderboardRecord{
		LeaderboardId: "backing_test_event_test_cohort",
		OwnerId:       userID,
		Username:      wrapperspb.String("testuser"),
		Score:         1000,
		Subscore:      0,
		Rank:          1,
	}
	nk.On("LeaderboardRecordWrite", ctx, mock.AnythingOfType("string"), userID, "testuser", int64(1000), int64(0), mock.Anything, mock.Anything).Return(record, nil)

	// Mock storage write for user state update
	nk.On("StorageWrite", ctx, mock.Anything).Return([]*api.StorageObjectAck{}, nil)

	// Mock leaderboard records list for building response
	nk.On("LeaderboardRecordsList", ctx, mock.AnythingOfType("string"), mock.Anything, 100, "", int64(0)).Return(
		[]*api.LeaderboardRecord{record}, []*api.LeaderboardRecord{}, "", "", nil)

	eventLeaderboard, err := system.UpdateEventLeaderboard(ctx, logger, nk, userID, "testuser", "test_event", 1000, 0, nil)
	require.NoError(t, err)
	assert.NotNil(t, eventLeaderboard)

	nk.AssertExpectations(t)
}

func TestClaimEventLeaderboard_Success(t *testing.T) {
	config := getTestEventLeaderboardsConfig()
	system := NewNakamaEventLeaderboardsSystem(config)
	mockPamlogix := createTestMockPamlogix(t)
	system.SetPamlogix(mockPamlogix)

	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Mock existing user state with a cohort in ended event
	existingState := &EventLeaderboardUserState{
		EventLeaderboards: map[string]*EventLeaderboardUserEventState{
			"ended_event": {
				CohortID: "test_cohort",
				Tier:     0,
			},
		},
	}
	stateData, _ := json.Marshal(existingState)
	nk.On("StorageRead", ctx, mock.Anything).Return([]*api.StorageObject{
		{Value: string(stateData)},
	}, nil)

	// Mock leaderboard records list for rank checking
	record := &api.LeaderboardRecord{
		LeaderboardId: "backing_ended_event_test_cohort",
		OwnerId:       userID,
		Username:      wrapperspb.String("testuser"),
		Score:         1000,
		Subscore:      0,
		Rank:          1,
		CreateTime:    timestamppb.Now(),
		UpdateTime:    timestamppb.Now(),
	}
	nk.On("LeaderboardRecordsList", ctx, mock.AnythingOfType("string"), []string{userID}, 1, "", int64(0)).Return(
		[]*api.LeaderboardRecord{record}, []*api.LeaderboardRecord{}, "", "", nil)

	// Mock storage write for user state update
	nk.On("StorageWrite", ctx, mock.Anything).Return([]*api.StorageObjectAck{}, nil)

	// Mock leaderboard records list for building response
	nk.On("LeaderboardRecordsList", ctx, mock.AnythingOfType("string"), mock.Anything, 100, "", int64(0)).Return(
		[]*api.LeaderboardRecord{record}, []*api.LeaderboardRecord{}, "", "", nil)

	eventLeaderboard, err := system.ClaimEventLeaderboard(ctx, logger, nk, userID, "ended_event")
	require.NoError(t, err)
	assert.NotNil(t, eventLeaderboard)
	assert.True(t, eventLeaderboard.ClaimTimeSec > 0)

	nk.AssertExpectations(t)
}

func TestProcessEventEnd_TierChanges(t *testing.T) {
	config := getTestEventLeaderboardsConfig()
	system := NewNakamaEventLeaderboardsSystem(config)
	mockPamlogix := createTestMockPamlogix(t)
	system.SetPamlogix(mockPamlogix)

	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()

	// Set event as ended
	config.EventLeaderboards["test_event"].EndTimeSec = time.Now().Unix() - 100

	// Mock storage list for finding cohorts
	cohortState := &EventLeaderboardCohortState{
		ID:                 "test_cohort",
		EventLeaderboardID: "test_event",
		Tier:               0,
		UserIDs:            []string{"user1", "user2", "user3", "user4", "user5"},
	}
	cohortData, _ := json.Marshal(cohortState)
	nk.On("StorageList", ctx, "", "", eventLeaderboardsStorageCollection, 100, "").Return([]*api.StorageObject{
		{
			Key:   "cohort_test_cohort",
			Value: string(cohortData),
		},
	}, "", nil)

	// Mock leaderboard records for tier change calculation
	records := []*api.LeaderboardRecord{
		{OwnerId: "user1", Score: 1000, Rank: 1}, // Should be promoted (top 20%)
		{OwnerId: "user2", Score: 800, Rank: 2},
		{OwnerId: "user3", Score: 600, Rank: 3},
		{OwnerId: "user4", Score: 200, Rank: 4}, // Should be demoted (bottom 30%)
		{OwnerId: "user5", Score: 0, Rank: 5},   // Should be demoted (bottom 30% + idle)
	}
	nk.On("LeaderboardRecordsList", ctx, mock.AnythingOfType("string"), mock.Anything, 100, "", int64(0)).Return(
		records, []*api.LeaderboardRecord{}, "", "", nil)

	// Mock storage read/write for user state updates
	// Based on the change zone config: promotion 0.2 (20%), demotion 0.3 (30%)
	// With 5 users: promotion = 1 user (user1 at index 0), demotion = 1 user (user5 at index 4)
	// user5 has score 0, so gets demoted for being idle
	// Let's include all users that might get tier changes to be safe
	usersWithTierChanges := []string{"user1", "user5"} // Only these users get tier changes
	for _, userID := range usersWithTierChanges {
		userState := &EventLeaderboardUserState{
			EventLeaderboards: map[string]*EventLeaderboardUserEventState{
				"test_event": {
					CohortID: "test_cohort",
					Tier:     0,
				},
			},
		}
		userData, _ := json.Marshal(userState)
		nk.On("StorageRead", ctx, mock.MatchedBy(func(reads []*runtime.StorageRead) bool {
			return len(reads) == 1 && reads[0].UserID == userID
		})).Return([]*api.StorageObject{
			{Value: string(userData)},
		}, nil)

		nk.On("StorageWrite", ctx, mock.MatchedBy(func(writes []*runtime.StorageWrite) bool {
			return len(writes) == 1 && writes[0].UserID == userID
		})).Return([]*api.StorageObjectAck{}, nil)
	}

	err := system.ProcessEventEnd(ctx, logger, nk, "test_event")
	require.NoError(t, err)

	nk.AssertExpectations(t)
}

func TestGetEventLeaderboard_WithScores(t *testing.T) {
	config := getTestEventLeaderboardsConfig()
	system := NewNakamaEventLeaderboardsSystem(config)
	mockPamlogix := createTestMockPamlogix(t)
	system.SetPamlogix(mockPamlogix)

	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Mock existing user state with a cohort
	existingState := &EventLeaderboardUserState{
		EventLeaderboards: map[string]*EventLeaderboardUserEventState{
			"test_event": {
				CohortID: "test_cohort",
				Tier:     0,
			},
		},
	}
	stateData, _ := json.Marshal(existingState)
	nk.On("StorageRead", ctx, mock.Anything).Return([]*api.StorageObject{
		{Value: string(stateData)},
	}, nil)

	// Mock leaderboard records list
	records := []*api.LeaderboardRecord{
		{
			OwnerId:    userID,
			Username:   wrapperspb.String("testuser"),
			Score:      1000,
			Subscore:   0,
			Rank:       1,
			CreateTime: timestamppb.Now(),
			UpdateTime: timestamppb.Now(),
		},
		{
			OwnerId:    "user2",
			Username:   wrapperspb.String("testuser2"),
			Score:      800,
			Subscore:   0,
			Rank:       2,
			CreateTime: timestamppb.Now(),
			UpdateTime: timestamppb.Now(),
		},
	}
	nk.On("LeaderboardRecordsList", ctx, mock.AnythingOfType("string"), mock.Anything, 100, "", int64(0)).Return(
		records, []*api.LeaderboardRecord{}, "", "", nil)

	eventLeaderboard, err := system.GetEventLeaderboard(ctx, logger, nk, userID, "test_event")
	require.NoError(t, err)
	assert.NotNil(t, eventLeaderboard)
	assert.Equal(t, "test_event", eventLeaderboard.Id)
	assert.Len(t, eventLeaderboard.Scores, 2)
	assert.Equal(t, int64(2), eventLeaderboard.Count)

	nk.AssertExpectations(t)
}

func TestDebugFill_AddsDummyUsers(t *testing.T) {
	config := getTestEventLeaderboardsConfig()
	system := NewNakamaEventLeaderboardsSystem(config)
	mockPamlogix := createTestMockPamlogix(t)
	system.SetPamlogix(mockPamlogix)

	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Mock existing user state with a cohort
	existingState := &EventLeaderboardUserState{
		EventLeaderboards: map[string]*EventLeaderboardUserEventState{
			"test_event": {
				CohortID: "test_cohort",
				Tier:     0,
			},
		},
	}
	stateData, _ := json.Marshal(existingState)
	nk.On("StorageRead", ctx, mock.Anything).Return([]*api.StorageObject{
		{Value: string(stateData)},
	}, nil)

	// Mock existing records (only 1 user)
	existingRecord := &api.LeaderboardRecord{
		OwnerId:  userID,
		Username: wrapperspb.String("testuser"),
		Score:    1000,
		Rank:     1,
	}
	nk.On("LeaderboardRecordsList", ctx, mock.AnythingOfType("string"), mock.Anything, 100, "", int64(0)).Return(
		[]*api.LeaderboardRecord{existingRecord}, []*api.LeaderboardRecord{}, "", "", nil).Once()

	// Mock leaderboard record writes for dummy users
	for i := 1; i < 5; i++ { // Adding 4 more users to reach target of 5
		nk.On("LeaderboardRecordWrite", ctx, mock.AnythingOfType("string"),
			mock.MatchedBy(func(dummyUserID string) bool {
				return dummyUserID != userID // Should be dummy user ID
			}),
			mock.AnythingOfType("string"), int64(0), int64(0), mock.Anything, mock.Anything).Return(
			&api.LeaderboardRecord{}, nil).Once()
	}

	// Mock final records list with all users
	finalRecords := []*api.LeaderboardRecord{existingRecord}
	for i := 1; i < 5; i++ {
		finalRecords = append(finalRecords, &api.LeaderboardRecord{
			OwnerId:  "dummy_test_cohort_" + string(rune(i)),
			Username: wrapperspb.String("Bot" + string(rune(i+1))),
			Score:    0,
			Rank:     int64(i + 1),
		})
	}
	nk.On("LeaderboardRecordsList", ctx, mock.AnythingOfType("string"), mock.Anything, 100, "", int64(0)).Return(
		finalRecords, []*api.LeaderboardRecord{}, "", "", nil).Once()

	eventLeaderboard, err := system.DebugFill(ctx, logger, nk, userID, "test_event", 5)
	require.NoError(t, err)
	assert.NotNil(t, eventLeaderboard)
	assert.Equal(t, int64(5), eventLeaderboard.Count)

	nk.AssertExpectations(t)
}

// Mock economy system for testing
type MockEconomySystem struct {
	mock.Mock
}

func (m *MockEconomySystem) GetType() SystemType {
	return SystemTypeEconomy
}

func (m *MockEconomySystem) GetConfig() any {
	args := m.Called()
	return args.Get(0)
}

func (m *MockEconomySystem) RewardRoll(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, rewardConfig *EconomyConfigReward) (*Reward, error) {
	args := m.Called(ctx, logger, nk, userID, rewardConfig)
	return args.Get(0).(*Reward), args.Error(1)
}

func (m *MockEconomySystem) RewardGrant(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, reward *Reward, metadata map[string]interface{}, ignoreLimits bool) (newItems map[string]*InventoryItem, updatedItems map[string]*InventoryItem, notGrantedItemIDs map[string]int64, err error) {
	args := m.Called(ctx, logger, nk, userID, reward, metadata, ignoreLimits)
	return args.Get(0).(map[string]*InventoryItem), args.Get(1).(map[string]*InventoryItem), args.Get(2).(map[string]int64), args.Error(3)
}

func (m *MockEconomySystem) RewardConvert(contents *AvailableRewards) *EconomyConfigReward {
	args := m.Called(contents)
	return args.Get(0).(*EconomyConfigReward)
}

func (m *MockEconomySystem) UnmarshalWallet(account *api.Account) (map[string]int64, error) {
	args := m.Called(account)
	return args.Get(0).(map[string]int64), args.Error(1)
}

func (m *MockEconomySystem) Grant(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, currencies map[string]int64, items map[string]int64, modifiers []*RewardModifier, metadata map[string]interface{}) (updatedWallet map[string]int64, rewardModifiers []*ActiveRewardModifier, timestamp int64, err error) {
	args := m.Called(ctx, logger, nk, userID, currencies, items, modifiers, metadata)
	return args.Get(0).(map[string]int64), args.Get(1).([]*ActiveRewardModifier), args.Get(2).(int64), args.Error(3)
}

// Stub implementations for other required interface methods
func (m *MockEconomySystem) RewardCreate() *EconomyConfigReward { return nil }
func (m *MockEconomySystem) DonationClaim(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, donationClaims map[string]*EconomyDonationClaimRequestDetails) (*EconomyDonationsList, error) {
	return nil, nil
}
func (m *MockEconomySystem) DonationGet(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userIDs []string) (*EconomyDonationsByUserList, error) {
	return nil, nil
}
func (m *MockEconomySystem) DonationGive(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, donationID, fromUserID string) (*EconomyDonation, map[string]int64, *Inventory, []*ActiveRewardModifier, *Reward, int64, error) {
	return nil, nil, nil, nil, nil, 0, nil
}
func (m *MockEconomySystem) DonationRequest(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, donationID string) (*EconomyDonation, bool, error) {
	return nil, false, nil
}
func (m *MockEconomySystem) List(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (map[string]*EconomyConfigStoreItem, map[string]*EconomyConfigPlacement, []*ActiveRewardModifier, int64, error) {
	return nil, nil, nil, 0, nil
}
func (m *MockEconomySystem) PurchaseIntent(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, itemID string, store EconomyStoreType, sku string) error {
	return nil
}
func (m *MockEconomySystem) PurchaseItem(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, userID, itemID string, store EconomyStoreType, receipt string) (map[string]int64, *Inventory, *Reward, bool, error) {
	return nil, nil, nil, false, nil
}
func (m *MockEconomySystem) PurchaseRestore(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, store EconomyStoreType, receipts []string) error {
	return nil
}
func (m *MockEconomySystem) PlacementStatus(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, rewardID, placementID string, retryCount int) (*EconomyPlacementStatus, error) {
	return nil, nil
}
func (m *MockEconomySystem) PlacementStart(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, placementID string, metadata map[string]string) (*EconomyPlacementStatus, error) {
	return nil, nil
}
func (m *MockEconomySystem) PlacementSuccess(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, rewardID, placementID string) (*Reward, map[string]string, error) {
	return nil, nil, nil
}
func (m *MockEconomySystem) PlacementFail(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, rewardID, placementID string) (map[string]string, error) {
	return nil, nil
}
func (m *MockEconomySystem) SetOnDonationClaimReward(fn OnReward[*EconomyConfigDonation])       {}
func (m *MockEconomySystem) SetOnDonationContributorReward(fn OnReward[*EconomyConfigDonation]) {}
func (m *MockEconomySystem) SetOnPlacementReward(fn OnReward[*EconomyPlacementInfo])            {}
func (m *MockEconomySystem) SetOnStoreItemReward(fn OnReward[*EconomyConfigStoreItem])          {}

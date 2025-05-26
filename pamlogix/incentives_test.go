package pamlogix

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/heroiclabs/nakama-common/api"
	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Mock implementations for testing
type mockIncentivesLogger struct{}

func (m *mockIncentivesLogger) Debug(format string, v ...interface{}) {}
func (m *mockIncentivesLogger) Info(format string, v ...interface{})  {}
func (m *mockIncentivesLogger) Warn(format string, v ...interface{})  {}
func (m *mockIncentivesLogger) Error(format string, v ...interface{}) {}
func (m *mockIncentivesLogger) WithField(key string, v interface{}) runtime.Logger {
	return m
}
func (m *mockIncentivesLogger) WithFields(fields map[string]interface{}) runtime.Logger {
	return m
}
func (m *mockIncentivesLogger) Fields() map[string]interface{} { return nil }

type mockIncentivesNakama struct {
	*MockNakamaModule
	storage   map[string]string
	accounts  map[string]*api.Account
	failRead  bool
	failWrite bool
}

func newMockIncentivesNakama() *mockIncentivesNakama {
	return &mockIncentivesNakama{
		MockNakamaModule: NewMockNakama(nil),
		storage:          make(map[string]string),
		accounts:         make(map[string]*api.Account),
	}
}

func (m *mockIncentivesNakama) StorageRead(ctx context.Context, reads []*runtime.StorageRead) ([]*api.StorageObject, error) {
	if m.failRead {
		return nil, runtime.NewError("mock read error", 13)
	}

	var result []*api.StorageObject
	for _, r := range reads {
		key := r.UserID + ":" + r.Collection + ":" + r.Key
		if val, ok := m.storage[key]; ok {
			result = append(result, &api.StorageObject{
				Collection: r.Collection,
				Key:        r.Key,
				UserId:     r.UserID,
				Value:      val,
				Version:    "v1",
			})
		}
	}
	return result, nil
}

func (m *mockIncentivesNakama) StorageWrite(ctx context.Context, writes []*runtime.StorageWrite) ([]*api.StorageObjectAck, error) {
	if m.failWrite {
		return nil, runtime.NewError("mock write error", 13)
	}

	var acks []*api.StorageObjectAck
	for _, w := range writes {
		key := w.UserID + ":" + w.Collection + ":" + w.Key
		m.storage[key] = w.Value
		acks = append(acks, &api.StorageObjectAck{
			Collection: w.Collection,
			Key:        w.Key,
			UserId:     w.UserID,
			Version:    "v1",
		})
	}
	return acks, nil
}

func (m *mockIncentivesNakama) StorageList(ctx context.Context, collection, userID, key string, limit int, cursor string) ([]*api.StorageObject, string, error) {
	var result []*api.StorageObject
	for storageKey, value := range m.storage {
		if collection != "" && !incentivesContains(storageKey, collection) {
			continue
		}
		if key != "" && !incentivesContains(storageKey, key) {
			continue
		}

		parts := incentivesSplitStorageKey(storageKey)
		if len(parts) >= 3 {
			result = append(result, &api.StorageObject{
				Collection: parts[1],
				Key:        parts[2],
				UserId:     parts[0],
				Value:      value,
				Version:    "v1",
			})
		}
	}
	return result, "", nil
}

func (m *mockIncentivesNakama) AccountGetId(ctx context.Context, userID string) (*api.Account, error) {
	if account, exists := m.accounts[userID]; exists {
		return account, nil
	}

	// Create a default account
	now := timestamppb.Now()
	account := &api.Account{
		User: &api.User{
			Id:         userID,
			Username:   "test_user_" + userID,
			CreateTime: now,
			UpdateTime: now,
		},
	}
	m.accounts[userID] = account
	return account, nil
}

func (m *mockIncentivesNakama) WalletUpdate(ctx context.Context, userID string, changeset map[string]int64, metadata map[string]interface{}, updateLedger bool) (updated map[string]int64, previous map[string]int64, err error) {
	// Simple mock implementation that returns the changeset as the updated wallet
	return changeset, make(map[string]int64), nil
}

// Helper functions
func incentivesContains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func incentivesSplitStorageKey(key string) []string {
	return strings.Split(key, ":")
}

// Test fixtures
func createTestConfig() *IncentivesConfig {
	return &IncentivesConfig{
		Incentives: map[string]*IncentivesConfigIncentive{
			"invite": {
				Type:               IncentiveType_INCENTIVE_TYPE_INVITE,
				Name:               "Invite Friend",
				Description:        "Invite a friend to get rewards",
				MaxClaims:          10,
				MaxConcurrent:      3,
				ExpiryDurationSec:  86400,  // 24 hours
				MaxRecipientAgeSec: 604800, // 7 days
				MaxGlobalClaims:    5,
				RecipientReward: &EconomyConfigReward{
					Guaranteed: &EconomyConfigRewardContents{
						Currencies: map[string]*EconomyConfigRewardCurrency{
							"gems": {EconomyConfigRewardRangeInt64{Min: 100, Max: 100}},
						},
					},
				},
				SenderReward: &EconomyConfigReward{
					Guaranteed: &EconomyConfigRewardContents{
						Currencies: map[string]*EconomyConfigRewardCurrency{
							"gems": {EconomyConfigRewardRangeInt64{Min: 50, Max: 50}},
						},
					},
				},
				AdditionalProperties: map[string]interface{}{
					"category": "social",
					"priority": 1,
				},
			},
			"daily_bonus": {
				Type:              IncentiveType_INCENTIVE_TYPE_INVITE,
				Name:              "Daily Bonus",
				Description:       "Daily login bonus",
				MaxClaims:         1,
				MaxConcurrent:     1,
				ExpiryDurationSec: 3600, // 1 hour
				RecipientReward: &EconomyConfigReward{
					Guaranteed: &EconomyConfigRewardContents{
						Currencies: map[string]*EconomyConfigRewardCurrency{
							"coins": {EconomyConfigRewardRangeInt64{Min: 500, Max: 500}},
						},
					},
				},
			},
		},
	}
}

func createMockPamlogix() *pamlogixImpl {
	economySystem := NewNakamaEconomySystem(&EconomyConfig{})
	pamlogix := &pamlogixImpl{
		systems: make(map[SystemType]System),
	}
	pamlogix.systems[SystemTypeEconomy] = economySystem
	return pamlogix
}

// Basic functionality tests
func TestNakamaIncentivesSystem_Creation(t *testing.T) {
	config := createTestConfig()
	system := NewNakamaIncentivesSystem(config)

	assert.NotNil(t, system)
	assert.Equal(t, SystemTypeIncentives, system.GetType())
	assert.Equal(t, config, system.GetConfig())
	assert.Nil(t, system.pamlogix)
	assert.Nil(t, system.onSenderReward)
	assert.Nil(t, system.onRecipientReward)
}

func TestNakamaIncentivesSystem_SetPamlogix(t *testing.T) {
	config := &IncentivesConfig{}
	system := NewNakamaIncentivesSystem(config)

	mockPamlogix := createMockPamlogix()
	system.SetPamlogix(mockPamlogix)

	assert.Equal(t, mockPamlogix, system.pamlogix)
}

func TestNakamaIncentivesSystem_SetRewardCallbacks(t *testing.T) {
	config := &IncentivesConfig{}
	system := NewNakamaIncentivesSystem(config)

	// Test setting sender reward callback
	senderCallback := func(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, sourceID string, source *IncentivesConfigIncentive, rewardConfig *EconomyConfigReward, reward *Reward) (*Reward, error) {
		return reward, nil
	}
	system.SetOnSenderReward(senderCallback)
	assert.NotNil(t, system.onSenderReward)

	// Test setting recipient reward callback
	recipientCallback := func(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, sourceID string, source *IncentivesConfigIncentive, rewardConfig *EconomyConfigReward, reward *Reward) (*Reward, error) {
		return reward, nil
	}
	system.SetOnRecipientReward(recipientCallback)
	assert.NotNil(t, system.onRecipientReward)
}

// Code generation tests
func TestNakamaIncentivesSystem_GenerateIncentiveCode(t *testing.T) {
	config := &IncentivesConfig{}
	system := NewNakamaIncentivesSystem(config)

	code1 := system.generateIncentiveCode()
	code2 := system.generateIncentiveCode()

	// Codes should be different
	assert.NotEqual(t, code1, code2)

	// Codes should start with the prefix
	assert.Contains(t, code1, incentiveCodePrefix)
	assert.Contains(t, code2, incentiveCodePrefix)

	// Codes should be the expected length (prefix + 8 characters)
	expectedLength := len(incentiveCodePrefix) + 8
	assert.Equal(t, expectedLength, len(code1))
	assert.Equal(t, expectedLength, len(code2))

	// Test multiple generations to ensure uniqueness
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code := system.generateIncentiveCode()
		assert.False(t, codes[code], "Generated duplicate code: %s", code)
		codes[code] = true
	}
}

// Reward conversion tests
func TestNakamaIncentivesSystem_ConvertRewardConfig(t *testing.T) {
	config := &IncentivesConfig{}
	system := NewNakamaIncentivesSystem(config)

	// Test with nil config
	result := system.convertRewardConfigToAvailableRewards(nil)
	assert.Nil(t, result)

	// Test with valid config
	rewardConfig := &EconomyConfigReward{
		MaxRolls:       5,
		TotalWeight:    100,
		MaxRepeatRolls: 3,
	}

	result = system.convertRewardConfigToAvailableRewards(rewardConfig)
	assert.NotNil(t, result)
	assert.Equal(t, int64(5), result.MaxRolls)
	assert.Equal(t, int64(100), result.TotalWeight)
	assert.Equal(t, int64(3), result.MaxRepeatRolls)
}

// Reward merging tests
func TestNakamaIncentivesSystem_MergeRewards(t *testing.T) {
	config := &IncentivesConfig{}
	system := NewNakamaIncentivesSystem(config)

	// Test with nil rewards
	result := system.mergeRewards(nil, nil)
	assert.Nil(t, result)

	reward1 := &Reward{
		Currencies: map[string]int64{"gems": 100},
		Items:      map[string]int64{"sword": 1},
		Energies:   map[string]int32{"stamina": 10},
	}

	result = system.mergeRewards(reward1, nil)
	assert.Equal(t, reward1, result)

	result = system.mergeRewards(nil, reward1)
	assert.Equal(t, reward1, result)

	// Test merging two rewards
	reward2 := &Reward{
		Currencies: map[string]int64{"gems": 50, "gold": 200},
		Items:      map[string]int64{"shield": 1},
		Energies:   map[string]int32{"stamina": 5, "mana": 15},
	}

	result = system.mergeRewards(reward1, reward2)
	assert.NotNil(t, result)
	assert.Equal(t, int64(150), result.Currencies["gems"]) // 100 + 50
	assert.Equal(t, int64(200), result.Currencies["gold"])
	assert.Equal(t, int64(1), result.Items["sword"])
	assert.Equal(t, int64(1), result.Items["shield"])
	assert.Equal(t, int32(15), result.Energies["stamina"]) // 10 + 5
	assert.Equal(t, int32(15), result.Energies["mana"])
}

// Sender operations tests
func TestNakamaIncentivesSystem_SenderList_EmptyUser(t *testing.T) {
	config := createTestConfig()
	system := NewNakamaIncentivesSystem(config)
	logger := &mockIncentivesLogger{}
	nk := newMockIncentivesNakama()
	ctx := context.Background()
	userID := "user1"

	incentives, err := system.SenderList(ctx, logger, nk, userID)

	require.NoError(t, err)
	assert.NotNil(t, incentives)
	assert.Len(t, incentives, 0)
}

func TestNakamaIncentivesSystem_SenderCreate_Success(t *testing.T) {
	config := createTestConfig()
	system := NewNakamaIncentivesSystem(config)
	system.SetPamlogix(createMockPamlogix())
	logger := &mockIncentivesLogger{}
	nk := newMockIncentivesNakama()
	ctx := context.Background()
	userID := "user1"
	incentiveID := "invite"

	incentives, err := system.SenderCreate(ctx, logger, nk, userID, incentiveID)

	require.NoError(t, err)
	assert.NotNil(t, incentives)
	assert.Len(t, incentives, 1)

	incentive := incentives[0]
	assert.Equal(t, incentiveID, incentive.Id)
	assert.Equal(t, "Invite Friend", incentive.Name)
	assert.Equal(t, "Invite a friend to get rewards", incentive.Description)
	assert.Equal(t, IncentiveType_INCENTIVE_TYPE_INVITE, incentive.Type)
	assert.NotEmpty(t, incentive.Code)
	assert.Contains(t, incentive.Code, incentiveCodePrefix)
	assert.True(t, incentive.CreateTimeSec > 0)
	assert.True(t, incentive.UpdateTimeSec > 0)
	assert.True(t, incentive.ExpiryTimeSec > incentive.CreateTimeSec)
	assert.Equal(t, int64(10), incentive.MaxClaims)
	assert.NotNil(t, incentive.RecipientRewards)
	assert.NotNil(t, incentive.SenderRewards)
	assert.Len(t, incentive.UnclaimedRecipients, 0)
	assert.Len(t, incentive.Rewards, 0)
	assert.Len(t, incentive.Claims, 0)
	assert.NotNil(t, incentive.AdditionalProperties)
}

func TestNakamaIncentivesSystem_SenderCreate_InvalidIncentiveID(t *testing.T) {
	config := createTestConfig()
	system := NewNakamaIncentivesSystem(config)
	logger := &mockIncentivesLogger{}
	nk := newMockIncentivesNakama()
	ctx := context.Background()
	userID := "user1"
	incentiveID := "nonexistent"

	incentives, err := system.SenderCreate(ctx, logger, nk, userID, incentiveID)

	assert.Error(t, err)
	assert.Nil(t, incentives)
	assert.Contains(t, err.Error(), "incentive configuration not found")
}

func TestNakamaIncentivesSystem_SenderCreate_NoConfig(t *testing.T) {
	system := NewNakamaIncentivesSystem(nil)
	logger := &mockIncentivesLogger{}
	nk := newMockIncentivesNakama()
	ctx := context.Background()
	userID := "user1"
	incentiveID := "invite"

	incentives, err := system.SenderCreate(ctx, logger, nk, userID, incentiveID)

	assert.Error(t, err)
	assert.Nil(t, incentives)
	assert.Contains(t, err.Error(), "incentives system not configured")
}

func TestNakamaIncentivesSystem_SenderCreate_MaxConcurrentLimit(t *testing.T) {
	config := createTestConfig()
	system := NewNakamaIncentivesSystem(config)
	system.SetPamlogix(createMockPamlogix())
	logger := &mockIncentivesLogger{}
	nk := newMockIncentivesNakama()
	ctx := context.Background()
	userID := "user1"
	incentiveID := "invite"

	// Create max concurrent incentives
	for i := 0; i < 3; i++ {
		_, err := system.SenderCreate(ctx, logger, nk, userID, incentiveID)
		require.NoError(t, err)
	}

	// Try to create one more - should fail
	incentives, err := system.SenderCreate(ctx, logger, nk, userID, incentiveID)

	assert.Error(t, err)
	assert.Nil(t, incentives)
	assert.Contains(t, err.Error(), "maximum concurrent incentives reached")
}

func TestNakamaIncentivesSystem_SenderDelete_Success(t *testing.T) {
	config := createTestConfig()
	system := NewNakamaIncentivesSystem(config)
	system.SetPamlogix(createMockPamlogix())
	logger := &mockIncentivesLogger{}
	nk := newMockIncentivesNakama()
	ctx := context.Background()
	userID := "user1"
	incentiveID := "invite"

	// Create an incentive first
	incentives, err := system.SenderCreate(ctx, logger, nk, userID, incentiveID)
	require.NoError(t, err)
	require.Len(t, incentives, 1)

	code := incentives[0].Code

	// Delete the incentive
	incentives, err = system.SenderDelete(ctx, logger, nk, userID, code)

	require.NoError(t, err)
	assert.NotNil(t, incentives)
	assert.Len(t, incentives, 0)
}

func TestNakamaIncentivesSystem_SenderDelete_NotFound(t *testing.T) {
	config := createTestConfig()
	system := NewNakamaIncentivesSystem(config)
	logger := &mockIncentivesLogger{}
	nk := newMockIncentivesNakama()
	ctx := context.Background()
	userID := "user1"
	code := "INCNONEXIST"

	incentives, err := system.SenderDelete(ctx, logger, nk, userID, code)

	assert.Error(t, err)
	assert.Nil(t, incentives)
	assert.Contains(t, err.Error(), "incentive not found")
}

func TestNakamaIncentivesSystem_SenderDelete_AlreadyClaimed(t *testing.T) {
	config := createTestConfig()
	system := NewNakamaIncentivesSystem(config)
	system.SetPamlogix(createMockPamlogix())
	logger := &mockIncentivesLogger{}
	nk := newMockIncentivesNakama()
	ctx := context.Background()
	userID := "user1"
	incentiveID := "invite"

	// Create an incentive
	incentives, err := system.SenderCreate(ctx, logger, nk, userID, incentiveID)
	require.NoError(t, err)
	require.Len(t, incentives, 1)

	code := incentives[0].Code

	// Manually add a claim to simulate someone claimed it
	userIncentives, err := system.getUserIncentives(ctx, logger, nk, userID)
	require.NoError(t, err)

	if userIncentives[code].Claims == nil {
		userIncentives[code].Claims = make(map[string]*IncentiveClaim)
	}
	userIncentives[code].Claims["recipient1"] = &IncentiveClaim{
		ClaimTimeSec: time.Now().Unix(),
	}

	err = system.saveUserIncentives(ctx, logger, nk, userID, userIncentives)
	require.NoError(t, err)

	// Try to delete - should fail
	incentives, err = system.SenderDelete(ctx, logger, nk, userID, code)

	assert.Error(t, err)
	assert.Nil(t, incentives)
	assert.Contains(t, err.Error(), "cannot delete claimed incentive")
}

// Recipient operations tests
func TestNakamaIncentivesSystem_RecipientGet_Success(t *testing.T) {
	config := createTestConfig()
	system := NewNakamaIncentivesSystem(config)
	system.SetPamlogix(createMockPamlogix())
	logger := &mockIncentivesLogger{}
	nk := newMockIncentivesNakama()
	ctx := context.Background()
	senderID := "sender1"
	recipientID := "recipient1"
	incentiveID := "invite"

	// Create an incentive as sender
	incentives, err := system.SenderCreate(ctx, logger, nk, senderID, incentiveID)
	require.NoError(t, err)
	require.Len(t, incentives, 1)

	code := incentives[0].Code

	// Get incentive info as recipient
	incentiveInfo, err := system.RecipientGet(ctx, logger, nk, recipientID, code)

	require.NoError(t, err)
	assert.NotNil(t, incentiveInfo)
	assert.Equal(t, incentiveID, incentiveInfo.Id)
	assert.Equal(t, "Invite Friend", incentiveInfo.Name)
	assert.Equal(t, code, incentiveInfo.Code)
	assert.Equal(t, senderID, incentiveInfo.Sender)
	assert.True(t, incentiveInfo.CanClaim)
	assert.Nil(t, incentiveInfo.Reward)
	assert.Equal(t, int64(0), incentiveInfo.ClaimTimeSec)
	assert.NotNil(t, incentiveInfo.AvailableRewards)
}

func TestNakamaIncentivesSystem_RecipientGet_NotFound(t *testing.T) {
	config := createTestConfig()
	system := NewNakamaIncentivesSystem(config)
	logger := &mockIncentivesLogger{}
	nk := newMockIncentivesNakama()
	ctx := context.Background()
	recipientID := "recipient1"
	code := "INCNONEXIST"

	incentiveInfo, err := system.RecipientGet(ctx, logger, nk, recipientID, code)

	assert.Error(t, err)
	assert.Nil(t, incentiveInfo)
	assert.Contains(t, err.Error(), "incentive not found")
}

func TestNakamaIncentivesSystem_RecipientGet_Expired(t *testing.T) {
	config := createTestConfig()
	system := NewNakamaIncentivesSystem(config)
	system.SetPamlogix(createMockPamlogix())
	logger := &mockIncentivesLogger{}
	nk := newMockIncentivesNakama()
	ctx := context.Background()
	senderID := "sender1"
	recipientID := "recipient1"
	incentiveID := "invite"

	// Create an incentive
	incentives, err := system.SenderCreate(ctx, logger, nk, senderID, incentiveID)
	require.NoError(t, err)
	require.Len(t, incentives, 1)

	code := incentives[0].Code

	// Manually expire the incentive
	userIncentives, err := system.getUserIncentives(ctx, logger, nk, senderID)
	require.NoError(t, err)

	userIncentives[code].ExpiryTimeSec = time.Now().Unix() - 1
	err = system.saveUserIncentives(ctx, logger, nk, senderID, userIncentives)
	require.NoError(t, err)

	// Try to get expired incentive
	incentiveInfo, err := system.RecipientGet(ctx, logger, nk, recipientID, code)

	assert.Error(t, err)
	assert.Nil(t, incentiveInfo)
	assert.Contains(t, err.Error(), "incentive has expired")
}

func TestNakamaIncentivesSystem_RecipientClaim_Success(t *testing.T) {
	config := createTestConfig()
	system := NewNakamaIncentivesSystem(config)
	system.SetPamlogix(createMockPamlogix())
	logger := &mockIncentivesLogger{}
	nk := newMockIncentivesNakama()
	ctx := context.Background()
	senderID := "sender1"
	recipientID := "recipient1"
	incentiveID := "invite"

	// Create an incentive as sender
	incentives, err := system.SenderCreate(ctx, logger, nk, senderID, incentiveID)
	require.NoError(t, err)
	require.Len(t, incentives, 1)

	code := incentives[0].Code

	// Claim incentive as recipient
	incentiveInfo, err := system.RecipientClaim(ctx, logger, nk, recipientID, code)

	require.NoError(t, err)
	assert.NotNil(t, incentiveInfo)
	assert.Equal(t, incentiveID, incentiveInfo.Id)
	assert.Equal(t, code, incentiveInfo.Code)
	assert.Equal(t, senderID, incentiveInfo.Sender)
	assert.False(t, incentiveInfo.CanClaim) // Can't claim again
	assert.True(t, incentiveInfo.ClaimTimeSec > 0)

	// Verify the claim was recorded in sender's storage
	userIncentives, err := system.getUserIncentives(ctx, logger, nk, senderID)
	require.NoError(t, err)

	incentive := userIncentives[code]
	assert.Contains(t, incentive.Claims, recipientID)
	assert.Contains(t, incentive.UnclaimedRecipients, recipientID)
}

func TestNakamaIncentivesSystem_RecipientClaim_AlreadyClaimed(t *testing.T) {
	config := createTestConfig()
	system := NewNakamaIncentivesSystem(config)
	system.SetPamlogix(createMockPamlogix())
	logger := &mockIncentivesLogger{}
	nk := newMockIncentivesNakama()
	ctx := context.Background()
	senderID := "sender1"
	recipientID := "recipient1"
	incentiveID := "invite"

	// Create and claim an incentive
	incentives, err := system.SenderCreate(ctx, logger, nk, senderID, incentiveID)
	require.NoError(t, err)

	code := incentives[0].Code

	_, err = system.RecipientClaim(ctx, logger, nk, recipientID, code)
	require.NoError(t, err)

	// Try to claim again
	incentiveInfo, err := system.RecipientClaim(ctx, logger, nk, recipientID, code)

	assert.Error(t, err)
	assert.Nil(t, incentiveInfo)
	assert.Contains(t, err.Error(), "incentive already claimed")
}

func TestNakamaIncentivesSystem_RecipientClaim_MaxClaimsReached(t *testing.T) {
	config := createTestConfig()
	// Set max claims to 1 for testing
	config.Incentives["invite"].MaxClaims = 1

	system := NewNakamaIncentivesSystem(config)
	system.SetPamlogix(createMockPamlogix())
	logger := &mockIncentivesLogger{}
	nk := newMockIncentivesNakama()
	ctx := context.Background()
	senderID := "sender1"
	incentiveID := "invite"

	// Create an incentive
	incentives, err := system.SenderCreate(ctx, logger, nk, senderID, incentiveID)
	require.NoError(t, err)

	code := incentives[0].Code

	// First recipient claims successfully
	_, err = system.RecipientClaim(ctx, logger, nk, "recipient1", code)
	require.NoError(t, err)

	// Second recipient should fail
	incentiveInfo, err := system.RecipientClaim(ctx, logger, nk, "recipient2", code)

	assert.Error(t, err)
	assert.Nil(t, incentiveInfo)
	assert.Contains(t, err.Error(), "incentive claim limit reached")
}

// Sender claim tests
func TestNakamaIncentivesSystem_SenderClaim_Success(t *testing.T) {
	config := createTestConfig()
	system := NewNakamaIncentivesSystem(config)
	system.SetPamlogix(createMockPamlogix())
	logger := &mockIncentivesLogger{}
	nk := newMockIncentivesNakama()
	ctx := context.Background()
	senderID := "sender1"
	recipientID := "recipient1"
	incentiveID := "invite"

	// Create an incentive and have it claimed
	incentives, err := system.SenderCreate(ctx, logger, nk, senderID, incentiveID)
	require.NoError(t, err)

	code := incentives[0].Code

	_, err = system.RecipientClaim(ctx, logger, nk, recipientID, code)
	require.NoError(t, err)

	// Sender claims rewards
	incentives, err = system.SenderClaim(ctx, logger, nk, senderID, code, []string{recipientID})

	require.NoError(t, err)
	assert.NotNil(t, incentives)
	assert.Len(t, incentives, 1)

	// Verify recipient was removed from unclaimed list
	incentive := incentives[0]
	assert.NotContains(t, incentive.UnclaimedRecipients, recipientID)
}

func TestNakamaIncentivesSystem_SenderClaim_ClaimAll(t *testing.T) {
	config := createTestConfig()
	system := NewNakamaIncentivesSystem(config)
	system.SetPamlogix(createMockPamlogix())
	logger := &mockIncentivesLogger{}
	nk := newMockIncentivesNakama()
	ctx := context.Background()
	senderID := "sender1"
	incentiveID := "invite"

	// Create an incentive
	incentives, err := system.SenderCreate(ctx, logger, nk, senderID, incentiveID)
	require.NoError(t, err)

	code := incentives[0].Code

	// Multiple recipients claim
	recipients := []string{"recipient1", "recipient2", "recipient3"}
	for _, recipientID := range recipients {
		_, err = system.RecipientClaim(ctx, logger, nk, recipientID, code)
		require.NoError(t, err)
	}

	// Sender claims all rewards (empty claimantIDs means claim all)
	incentives, err = system.SenderClaim(ctx, logger, nk, senderID, code, []string{})

	require.NoError(t, err)
	assert.NotNil(t, incentives)
	assert.Len(t, incentives, 1)

	// Verify all recipients were removed from unclaimed list
	incentive := incentives[0]
	assert.Len(t, incentive.UnclaimedRecipients, 0)
}

// Error handling tests
func TestNakamaIncentivesSystem_StorageErrors(t *testing.T) {
	config := createTestConfig()
	system := NewNakamaIncentivesSystem(config)
	logger := &mockIncentivesLogger{}
	nk := newMockIncentivesNakama()
	ctx := context.Background()
	userID := "user1"

	// Test read error
	nk.failRead = true
	incentives, err := system.SenderList(ctx, logger, nk, userID)
	assert.Error(t, err)
	assert.Nil(t, incentives)

	nk.failRead = false
	nk.failWrite = true

	// Test write error
	incentives, err = system.SenderCreate(ctx, logger, nk, userID, "invite")
	assert.Error(t, err)
	assert.Nil(t, incentives)
}

func TestNakamaIncentivesSystem_NoEconomySystem(t *testing.T) {
	config := createTestConfig()
	system := NewNakamaIncentivesSystem(config)
	// Don't set pamlogix, so no economy system available
	logger := &mockIncentivesLogger{}
	nk := newMockIncentivesNakama()
	ctx := context.Background()
	senderID := "sender1"
	recipientID := "recipient1"
	incentiveID := "invite"

	// Create an incentive (this should work without economy system)
	incentives, err := system.SenderCreate(ctx, logger, nk, senderID, incentiveID)
	require.NoError(t, err)

	code := incentives[0].Code

	// Try to claim - should fail without economy system
	incentiveInfo, err := system.RecipientClaim(ctx, logger, nk, recipientID, code)

	assert.Error(t, err)
	assert.Nil(t, incentiveInfo)
	assert.Contains(t, err.Error(), "economy system not available")
}

// Age restriction tests
func TestNakamaIncentivesSystem_AgeRestriction(t *testing.T) {
	config := createTestConfig()
	system := NewNakamaIncentivesSystem(config)
	system.SetPamlogix(createMockPamlogix())
	logger := &mockIncentivesLogger{}
	nk := newMockIncentivesNakama()
	ctx := context.Background()
	senderID := "sender1"
	recipientID := "recipient1"
	incentiveID := "invite"

	// Create an old account (older than 7 days)
	oldTime := timestamppb.New(time.Now().Add(-8 * 24 * time.Hour))
	nk.accounts[recipientID] = &api.Account{
		User: &api.User{
			Id:         recipientID,
			Username:   "old_user",
			CreateTime: oldTime,
			UpdateTime: oldTime,
		},
	}

	// Create an incentive
	incentives, err := system.SenderCreate(ctx, logger, nk, senderID, incentiveID)
	require.NoError(t, err)

	code := incentives[0].Code

	// Old user should not be able to claim
	incentiveInfo, err := system.RecipientGet(ctx, logger, nk, recipientID, code)
	require.NoError(t, err)
	assert.False(t, incentiveInfo.CanClaim)

	// Try to claim anyway - should fail
	incentiveInfo, err = system.RecipientClaim(ctx, logger, nk, recipientID, code)
	assert.Error(t, err)
	assert.Nil(t, incentiveInfo)
	assert.Contains(t, err.Error(), "user cannot claim this incentive")
}

// Custom reward callback tests
func TestNakamaIncentivesSystem_CustomRewardCallbacks(t *testing.T) {
	config := createTestConfig()
	system := NewNakamaIncentivesSystem(config)
	system.SetPamlogix(createMockPamlogix())
	logger := &mockIncentivesLogger{}
	nk := newMockIncentivesNakama()
	ctx := context.Background()
	senderID := "sender1"
	recipientID := "recipient1"
	incentiveID := "invite"

	// Set custom reward callbacks
	senderCallbackCalled := false
	recipientCallbackCalled := false

	system.SetOnSenderReward(func(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, sourceID string, source *IncentivesConfigIncentive, rewardConfig *EconomyConfigReward, reward *Reward) (*Reward, error) {
		senderCallbackCalled = true
		// Double the reward
		if reward != nil && reward.Currencies != nil {
			for k, v := range reward.Currencies {
				reward.Currencies[k] = v * 2
			}
		}
		return reward, nil
	})

	system.SetOnRecipientReward(func(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, sourceID string, source *IncentivesConfigIncentive, rewardConfig *EconomyConfigReward, reward *Reward) (*Reward, error) {
		recipientCallbackCalled = true
		// Add bonus currency
		if reward != nil && reward.Currencies != nil {
			reward.Currencies["bonus"] = 25
		}
		return reward, nil
	})

	// Create and claim incentive
	incentives, err := system.SenderCreate(ctx, logger, nk, senderID, incentiveID)
	require.NoError(t, err)

	code := incentives[0].Code

	_, err = system.RecipientClaim(ctx, logger, nk, recipientID, code)
	require.NoError(t, err)

	_, err = system.SenderClaim(ctx, logger, nk, senderID, code, []string{recipientID})
	require.NoError(t, err)

	// Verify callbacks were called
	assert.True(t, recipientCallbackCalled)
	assert.True(t, senderCallbackCalled)
}

// Integration test
func TestNakamaIncentivesSystem_FullWorkflow(t *testing.T) {
	config := createTestConfig()
	system := NewNakamaIncentivesSystem(config)
	system.SetPamlogix(createMockPamlogix())
	logger := &mockIncentivesLogger{}
	nk := newMockIncentivesNakama()
	ctx := context.Background()

	senderID := "sender1"
	recipients := []string{"recipient1", "recipient2", "recipient3"}
	incentiveID := "invite"

	// 1. Sender creates incentive
	incentives, err := system.SenderList(ctx, logger, nk, senderID)
	require.NoError(t, err)
	assert.Len(t, incentives, 0)

	incentives, err = system.SenderCreate(ctx, logger, nk, senderID, incentiveID)
	require.NoError(t, err)
	assert.Len(t, incentives, 1)

	code := incentives[0].Code

	// 2. Recipients view and claim incentive
	for _, recipientID := range recipients {
		// View incentive
		incentiveInfo, err := system.RecipientGet(ctx, logger, nk, recipientID, code)
		require.NoError(t, err)
		assert.True(t, incentiveInfo.CanClaim)

		// Claim incentive
		incentiveInfo, err = system.RecipientClaim(ctx, logger, nk, recipientID, code)
		require.NoError(t, err)
		assert.False(t, incentiveInfo.CanClaim)
		assert.True(t, incentiveInfo.ClaimTimeSec > 0)
	}

	// 3. Sender checks updated incentive list
	incentives, err = system.SenderList(ctx, logger, nk, senderID)
	require.NoError(t, err)
	assert.Len(t, incentives, 1)
	assert.Len(t, incentives[0].UnclaimedRecipients, 3)
	assert.Len(t, incentives[0].Claims, 3)

	// 4. Sender claims rewards for all recipients
	incentives, err = system.SenderClaim(ctx, logger, nk, senderID, code, []string{})
	require.NoError(t, err)
	assert.Len(t, incentives, 1)
	assert.Len(t, incentives[0].UnclaimedRecipients, 0)
	assert.Len(t, incentives[0].Claims, 3)

	// 5. Verify recipients can't claim again
	for _, recipientID := range recipients {
		incentiveInfo, err := system.RecipientGet(ctx, logger, nk, recipientID, code)
		require.NoError(t, err)
		assert.False(t, incentiveInfo.CanClaim)

		_, err = system.RecipientClaim(ctx, logger, nk, recipientID, code)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "incentive already claimed")
	}

	// 6. Sender deletes incentive (should fail because it was claimed)
	_, err = system.SenderDelete(ctx, logger, nk, senderID, code)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete claimed incentive")
}

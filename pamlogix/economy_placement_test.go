package pamlogix

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/heroiclabs/nakama-common/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Tests for PlacementStatus

func TestPlacementStatus_Success(t *testing.T) {
	// Setup
	config := &EconomyConfig{
		Placements: map[string]*EconomyConfigPlacement{
			"ad_placement1": {
				Reward: &EconomyConfigReward{},
			},
		},
	}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"
	placementID := "ad_placement1"
	rewardID := "reward1"

	// Existing placement data
	timestamp := time.Now().Unix()
	placementData := map[string]interface{}{
		"status":    "completed",
		"metadata":  map[string]string{"platform": "ios"},
		"timestamp": timestamp,
	}
	jsonData, _ := json.Marshal(placementData)

	// Mock storage read
	nk.On("StorageRead", mock.Anything, mock.Anything).Return([]*api.StorageObject{
		{
			Collection: "economy_placements",
			Key:        placementID,
			UserId:     userID,
			Value:      string(jsonData),
			Version:    "v1",
		},
	}, nil)

	// Execute
	status, err := economy.PlacementStatus(ctx, logger, nk, userID, rewardID, placementID, 0)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, rewardID, status.RewardId)
	assert.Equal(t, placementID, status.PlacementId)
	assert.True(t, status.Success)
	assert.Equal(t, timestamp, status.CreateTimeSec)
	assert.Equal(t, timestamp, status.CompleteTimeSec)

	nk.AssertExpectations(t)
}

func TestPlacementStatus_PlacementNotFound(t *testing.T) {
	// Setup
	config := &EconomyConfig{
		Placements: map[string]*EconomyConfigPlacement{},
	}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"
	placementID := "non_existent_placement"
	rewardID := "reward1"

	// Execute
	status, err := economy.PlacementStatus(ctx, logger, nk, userID, rewardID, placementID, 0)

	// Verify
	assert.Equal(t, ErrEconomyNoPlacement, err)
	assert.Nil(t, status)
}

// Tests for PlacementStart

func TestPlacementStart_Success(t *testing.T) {
	// Setup
	config := &EconomyConfig{
		Placements: map[string]*EconomyConfigPlacement{
			"ad_placement1": {
				Reward: &EconomyConfigReward{},
			},
		},
	}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"
	placementID := "ad_placement1"
	metadata := map[string]string{"platform": "android"}

	// Mock storage write
	nk.On("StorageWrite", mock.Anything, mock.Anything).Return([]*api.StorageObjectAck{
		{
			Collection: "economy_placements",
			Key:        placementID,
			UserId:     userID,
			Version:    "v1",
		},
	}, nil)

	// Execute
	status, err := economy.PlacementStart(ctx, logger, nk, userID, placementID, metadata)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, placementID, status.PlacementId)
	assert.NotEmpty(t, status.RewardId)
	assert.NotZero(t, status.CreateTimeSec)
	assert.Equal(t, metadata, status.Metadata)

	nk.AssertExpectations(t)
}

func TestPlacementStart_PlacementNotFound(t *testing.T) {
	// Setup
	config := &EconomyConfig{
		Placements: map[string]*EconomyConfigPlacement{},
	}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"
	placementID := "non_existent_placement"
	metadata := map[string]string{}

	// Execute
	status, err := economy.PlacementStart(ctx, logger, nk, userID, placementID, metadata)

	// Verify
	assert.Equal(t, ErrEconomyNoPlacement, err)
	assert.Nil(t, status)
}

// Tests for PlacementSuccess

func TestPlacementSuccess_Success(t *testing.T) {
	// Setup
	config := &EconomyConfig{
		Placements: map[string]*EconomyConfigPlacement{
			"ad_placement1": {
				Reward: &EconomyConfigReward{
					Guaranteed: &EconomyConfigRewardContents{
						Currencies: map[string]*EconomyConfigRewardCurrency{
							"coins": {EconomyConfigRewardRangeInt64{Min: 50, Max: 50}},
						},
					},
				},
			},
		},
	}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"
	rewardID := "reward1"
	placementID := "ad_placement1"

	// Mock storage read
	placementData := map[string]interface{}{
		"status":    "started",
		"metadata":  map[string]string{"platform": "ios"},
		"timestamp": time.Now().Unix(),
	}
	jsonData, _ := json.Marshal(placementData)
	nk.On("StorageRead", mock.Anything, mock.Anything).Return([]*api.StorageObject{
		{
			Collection: "economy_placements",
			Key:        placementID,
			UserId:     userID,
			Value:      string(jsonData),
			Version:    "v1",
		},
	}, nil)

	// Mock wallet update for reward
	nk.On("WalletUpdate", mock.Anything, userID, mock.Anything, mock.Anything, false).Return(
		map[string]int64{"coins": 50}, map[string]int64{}, nil)

	// Mock storage write for status update
	nk.On("StorageWrite", mock.Anything, mock.Anything).Return([]*api.StorageObjectAck{}, nil)

	// Be more specific with the StorageList mock parameters
	nk.On("StorageList", mock.Anything, "", userID, "inventory", 100, "").Return([]*api.StorageObject{}, "", nil).Maybe()

	// Execute
	reward, metadata, err := economy.PlacementSuccess(ctx, logger, nk, userID, rewardID, placementID)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, reward)
	assert.Equal(t, placementData["metadata"].(map[string]string), metadata)
	assert.Equal(t, int64(50), reward.Currencies["coins"])

	nk.AssertExpectations(t)
}

func TestPlacementSuccess_PlacementNotFound(t *testing.T) {
	// Setup
	config := &EconomyConfig{
		Placements: map[string]*EconomyConfigPlacement{},
	}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"
	rewardID := "reward1"
	placementID := "non_existent_placement"

	// Execute
	reward, metadata, err := economy.PlacementSuccess(ctx, logger, nk, userID, rewardID, placementID)

	// Verify
	assert.Equal(t, ErrEconomyNoPlacement, err)
	assert.Nil(t, reward)
	assert.Nil(t, metadata)
}

// Tests for PlacementFail

func TestPlacementFail_Success(t *testing.T) {
	// Setup
	config := &EconomyConfig{
		Placements: map[string]*EconomyConfigPlacement{
			"ad_placement1": {
				Reward: &EconomyConfigReward{},
			},
		},
	}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"
	rewardID := "reward1"
	placementID := "ad_placement1"

	// Mock storage read
	placementData := map[string]interface{}{
		"status":    "started",
		"metadata":  map[string]string{"platform": "ios"},
		"timestamp": time.Now().Unix(),
	}
	jsonData, _ := json.Marshal(placementData)
	nk.On("StorageRead", mock.Anything, mock.Anything).Return([]*api.StorageObject{
		{
			Collection: "economy_placements",
			Key:        placementID,
			UserId:     userID,
			Value:      string(jsonData),
			Version:    "v1",
		},
	}, nil)

	// Mock storage write for status update
	nk.On("StorageWrite", mock.Anything, mock.Anything).Return([]*api.StorageObjectAck{}, nil)

	// Execute
	metadata, err := economy.PlacementFail(ctx, logger, nk, userID, rewardID, placementID)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, placementData["metadata"].(map[string]string), metadata)

	nk.AssertExpectations(t)
}

func TestPlacementFail_PlacementNotFound(t *testing.T) {
	// Setup
	config := &EconomyConfig{
		Placements: map[string]*EconomyConfigPlacement{},
	}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"
	rewardID := "reward1"
	placementID := "non_existent_placement"

	// Execute
	metadata, err := economy.PlacementFail(ctx, logger, nk, userID, rewardID, placementID)

	// Verify
	assert.Equal(t, ErrEconomyNoPlacement, err)
	assert.Nil(t, metadata)
}

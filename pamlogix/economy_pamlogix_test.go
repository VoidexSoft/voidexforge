package pamlogix

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/heroiclabs/nakama-common/api"
	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Helper functions remain the same
func formatStorageKey(collection, key, userID string) string {
	return collection + ":" + key + ":" + userID
}

func splitStorageKey(fullKey string) []string {
	return strings.SplitN(fullKey, ":", 3)
}

func hasPrefix(s, prefix string) bool {
	return len(prefix) == 0 || (len(s) >= len(prefix) && s[:len(prefix)] == prefix)
}

// mockLogger is a simple logger that implements runtime.Logger for testing.
type mockLogger struct{}

func (l *mockLogger) Debug(format string, v ...interface{})                   {}
func (l *mockLogger) Info(format string, v ...interface{})                    {}
func (l *mockLogger) Warn(format string, v ...interface{})                    {}
func (l *mockLogger) Error(format string, v ...interface{})                   {}
func (l *mockLogger) WithField(key string, v interface{}) runtime.Logger      { return l }
func (l *mockLogger) WithFields(fields map[string]interface{}) runtime.Logger { return l }
func (l *mockLogger) Fields() map[string]interface{}                          { return nil }

// Example usage in tests
func TestRewardRoll_GuaranteedCurrency(t *testing.T) {
	economy := NewNakamaEconomySystem(nil)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Set up expectations if needed
	// nk.On("StorageRead", mock.Anything, mock.Anything).Return([]*api.StorageObject{}, nil)

	rewardConfig := &EconomyConfigReward{
		Guaranteed: &EconomyConfigRewardContents{
			Currencies: map[string]*EconomyConfigRewardCurrency{
				"gold": {EconomyConfigRewardRangeInt64{Min: 100, Max: 100, Multiple: 1}},
			},
		},
	}

	reward, err := economy.RewardRoll(ctx, logger, nk, userID, rewardConfig)
	require.NoError(t, err)
	assert.NotNil(t, reward)
	assert.Equal(t, int64(100), reward.Currencies["gold"])

	// Verify expectations if set
	// nk.AssertExpectations(t)
}

func TestRewardGrant_CurrencyAndItem(t *testing.T) {
	economy := NewNakamaEconomySystem(nil)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user2"

	// Set expectations
	nk.On("WalletUpdate", mock.Anything, userID, mock.Anything, mock.Anything, false).Return(
		map[string]int64{"gold": 50}, map[string]int64{}, nil)
	nk.On("StorageWrite", mock.Anything, mock.Anything).Return([]*api.StorageObjectAck{}, nil)
	nk.On("StorageRead", mock.Anything, mock.Anything).Return([]*api.StorageObject{}, nil).Maybe()
	nk.On("StorageList", mock.Anything, "", userID, "inventory", 100, "").Return([]*api.StorageObject{}, "", nil)

	reward := &Reward{
		Currencies: map[string]int64{"gold": 50},
		Items:      map[string]int64{"potion": 2},
	}

	newItems, updatedItems, notGranted, err := economy.RewardGrant(ctx, logger, nk, userID, reward, nil, false)
	require.NoError(t, err)
	assert.Len(t, newItems, 1)
	assert.Contains(t, newItems, "potion")
	assert.Equal(t, int64(2), newItems["potion"].Count)
	assert.Len(t, updatedItems, 0)
	assert.Len(t, notGranted, 0)

	// Verify expectations
	nk.AssertExpectations(t)
}

func TestDonationRequest_NewDonation(t *testing.T) {
	donationID := "don1"
	userID := "user1"
	config := &EconomyConfig{
		Donations: map[string]*EconomyConfigDonation{
			donationID: {
				Description:              "desc",
				DurationSec:              1000,
				MaxCount:                 10,
				Name:                     "Donation Name",
				UserContributionMaxCount: 5,
				AdditionalProperties:     map[string]string{"foo": "bar"},
			},
		},
	}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()

	// No existing donations
	nk.On("StorageRead", ctx, mock.Anything).Return([]*api.StorageObject{}, nil)
	nk.On("StorageWrite", ctx, mock.Anything).Return([]*api.StorageObjectAck{}, nil)

	donation, success, err := economy.DonationRequest(ctx, logger, nk, userID, donationID)
	require.NoError(t, err)
	assert.True(t, success)
	assert.NotNil(t, donation)
	assert.Equal(t, donationID, donation.Id)
	assert.Equal(t, userID, donation.UserId)
	assert.Equal(t, "desc", donation.Description)
	assert.Equal(t, int64(10), donation.MaxCount)
	assert.Equal(t, "Donation Name", donation.Name)
	assert.Equal(t, int64(5), donation.UserContributionMaxCount)
	assert.Equal(t, map[string]string{"foo": "bar"}, donation.AdditionalProperties)

	nk.AssertExpectations(t)
}

func TestDonationRequest_ExistingDonation(t *testing.T) {
	donationID := "don1"
	userID := "user1"
	config := &EconomyConfig{
		Donations: map[string]*EconomyConfigDonation{
			donationID: {
				Description: "desc",
			},
		},
	}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()

	futureTime := time.Now().Unix() + 3600 // 1 hour in the future
	donationObj := &api.StorageObject{
		Key:   "donation:" + donationID,
		Value: fmt.Sprintf(`{"user_id":"user1","id":"don1","description":"desc","expire_time_sec":%d}`, futureTime),
	}
	nk.On("StorageRead", ctx, mock.Anything).Return([]*api.StorageObject{donationObj}, nil)

	donation, success, err := economy.DonationRequest(ctx, logger, nk, userID, donationID)
	require.NoError(t, err)
	assert.False(t, success)
	assert.NotNil(t, donation)
	assert.Equal(t, donationID, donation.Id)
	assert.Equal(t, userID, donation.UserId)
	assert.Equal(t, "desc", donation.Description)

	nk.AssertExpectations(t)
}

func TestList_ReturnsConfig(t *testing.T) {
	config := &EconomyConfig{
		StoreItems: map[string]*EconomyConfigStoreItem{
			"item1": {Name: "Sword"},
		},
		Placements: map[string]*EconomyConfigPlacement{
			"place1": {Reward: &EconomyConfigReward{}},
		},
	}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	nk.On("StorageRead", ctx, mock.Anything).Return([]*api.StorageObject{}, nil)

	storeItems, placements, rewardModifiers, timestamp, err := economy.List(ctx, logger, nk, userID)
	require.NoError(t, err)
	assert.Contains(t, storeItems, "item1")
	assert.Contains(t, placements, "place1")
	assert.NotNil(t, rewardModifiers)
	assert.True(t, timestamp > 0)

	nk.AssertExpectations(t)
}

func TestGrant_Success(t *testing.T) {
	config := &EconomyConfig{}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	nk.On("WalletUpdate", ctx, userID, mock.Anything, mock.Anything, false).Return(map[string]int64{"gold": 100}, map[string]int64{}, nil)
	nk.On("StorageWrite", ctx, mock.Anything).Return([]*api.StorageObjectAck{}, nil)
	nk.On("StorageRead", ctx, mock.Anything).Return([]*api.StorageObject{}, nil)
	nk.On("StorageList", ctx, "", userID, "inventory", 100, "").Return([]*api.StorageObject{}, "", nil)
	nk.On("AccountGetId", ctx, userID).Return(&api.Account{Wallet: `{"gold":100}`}, nil)

	currencies := map[string]int64{"gold": 100}
	items := map[string]int64{"potion": 2}
	modifiers := []*RewardModifier{{Id: "mod1", Type: "bonus", Operator: "+", Value: 1}}
	walletMetadata := map[string]interface{}{"meta": "data"}

	updatedWallet, rewardModifiers, timestamp, err := economy.Grant(ctx, logger, nk, userID, currencies, items, modifiers, walletMetadata)
	require.NoError(t, err)
	assert.Equal(t, int64(100), updatedWallet["gold"])
	assert.NotNil(t, rewardModifiers)
	assert.True(t, timestamp > 0)

	nk.AssertExpectations(t)
}

func TestGrant_Error(t *testing.T) {
	config := &EconomyConfig{}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Return empty maps instead of nil to avoid type conversion panic
	nk.On("WalletUpdate", ctx, userID, mock.Anything, mock.Anything, false).Return(
		map[string]int64{}, map[string]int64{}, assert.AnError)

	currencies := map[string]int64{"gold": 100}
	items := map[string]int64{"potion": 2}
	modifiers := []*RewardModifier{{Id: "mod1", Type: "bonus", Operator: "+", Value: 1}}
	walletMetadata := map[string]interface{}{"meta": "data"}

	updatedWallet, rewardModifiers, timestamp, err := economy.Grant(ctx, logger, nk, userID, currencies, items, modifiers, walletMetadata)
	assert.Error(t, err)
	assert.Nil(t, updatedWallet)
	assert.Nil(t, rewardModifiers)
	assert.Equal(t, int64(0), timestamp)

	nk.AssertExpectations(t)
}

func TestPurchaseItem_Success_AppleStore(t *testing.T) {
	// Setup
	config := &EconomyConfig{
		StoreItems: map[string]*EconomyConfigStoreItem{
			"premium_pack": {
				Name:        "Premium Pack",
				Description: "A premium item pack",
				Cost: &EconomyConfigStoreItemCost{
					Sku: "com.example.premiumpack",
				},
				Reward: &EconomyConfigReward{
					Guaranteed: &EconomyConfigRewardContents{
						Currencies: map[string]*EconomyConfigRewardCurrency{
							"gold": {EconomyConfigRewardRangeInt64: EconomyConfigRewardRangeInt64{Min: 100, Max: 100}},
						},
						Items: map[string]*EconomyConfigRewardItem{
							"premium_sword": {
								EconomyConfigRewardRangeInt64: EconomyConfigRewardRangeInt64{Min: 1, Max: 1},
								StringProperties:              nil,
								NumericProperties:             nil,
							},
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
	itemID := "premium_pack"
	receipt := "valid_receipt_data"

	// Mock the purchase validation response
	validatedPurchase := &api.ValidatedPurchase{
		ProductId:     "com.example.premiumpack",
		TransactionId: "transaction123",
		Store:         api.StoreProvider_APPLE_APP_STORE,
		PurchaseTime:  &timestamppb.Timestamp{Seconds: time.Now().Unix()},
		Environment:   api.StoreEnvironment_PRODUCTION,
	}

	validationResponse := &api.ValidatePurchaseResponse{
		ValidatedPurchases: []*api.ValidatedPurchase{validatedPurchase},
	}

	// Setup expectations
	nk.On("PurchaseValidateApple", ctx, userID, receipt, true, []string(nil)).Return(validationResponse, nil)
	nk.On("StorageRead", ctx, mock.Anything).Return([]*api.StorageObject{}, nil)
	nk.On("StorageWrite", ctx, mock.Anything).Return([]*api.StorageObjectAck{}, nil)
	nk.On("WalletUpdate", ctx, userID, mock.Anything, mock.Anything, false).Return(
		map[string]int64{"gold": 100}, map[string]int64{}, nil)
	nk.On("AccountGetId", ctx, userID).Return(&api.Account{Wallet: `{"gold":100}`}, nil)
	nk.On("StorageList", ctx, "", userID, "inventory", 100, "").Return([]*api.StorageObject{}, "", nil)

	// Execute the method
	wallet, inventory, reward, isSandbox, err := economy.PurchaseItem(ctx, logger, nil, nk, userID, itemID, EconomyStoreType_ECONOMY_STORE_TYPE_APPLE_APPSTORE, receipt)

	// Verify results
	require.NoError(t, err)
	assert.Equal(t, map[string]int64{"gold": 100}, wallet)
	assert.NotNil(t, inventory)
	assert.NotNil(t, reward)
	assert.False(t, isSandbox)
	assert.Equal(t, int64(100), reward.Currencies["gold"])
	assert.Equal(t, int64(1), reward.Items["premium_sword"])

	// Verify expectations
	nk.AssertExpectations(t)
}

func TestPurchaseItem_Success_GooglePlay(t *testing.T) {
	// Setup
	config := &EconomyConfig{
		StoreItems: map[string]*EconomyConfigStoreItem{
			"gems_pack": {
				Name:        "Gems Pack",
				Description: "A gem pack for premium currency",
				Cost: &EconomyConfigStoreItemCost{
					Sku: "com.example.gemspack",
				},
				Reward: &EconomyConfigReward{
					Guaranteed: &EconomyConfigRewardContents{
						Currencies: map[string]*EconomyConfigRewardCurrency{
							"gems": {EconomyConfigRewardRangeInt64: EconomyConfigRewardRangeInt64{Min: 50, Max: 50}},
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
	userID := "user2"
	itemID := "gems_pack"
	receipt := "valid_google_receipt"

	// Mock the purchase validation response for Google Play
	validatedPurchase := &api.ValidatedPurchase{
		ProductId:     "com.example.gemspack",
		TransactionId: "gp_transaction123",
		Store:         api.StoreProvider_GOOGLE_PLAY_STORE,
		PurchaseTime:  &timestamppb.Timestamp{Seconds: time.Now().Unix()},
		Environment:   api.StoreEnvironment_SANDBOX,
	}

	validationResponse := &api.ValidatePurchaseResponse{
		ValidatedPurchases: []*api.ValidatedPurchase{validatedPurchase},
	}

	// Setup expectations
	nk.On("PurchaseValidateGoogle", ctx, userID, receipt, true, []struct {
		ClientEmail string
		PrivateKey  string
	}(nil)).Return(validationResponse, nil)
	nk.On("StorageRead", ctx, mock.Anything).Return([]*api.StorageObject{}, nil)
	nk.On("StorageWrite", ctx, mock.Anything).Return([]*api.StorageObjectAck{}, nil)
	nk.On("WalletUpdate", ctx, userID, mock.Anything, mock.Anything, false).Return(
		map[string]int64{"gems": 50}, map[string]int64{}, nil)
	nk.On("AccountGetId", ctx, userID).Return(&api.Account{Wallet: `{"gems":50}`}, nil)
	nk.On("StorageList", ctx, "", userID, "inventory", 100, "").Return([]*api.StorageObject{}, "", nil).Maybe()
	nk.On("StorageList", ctx, "", userID, "purchase_transactions", 100, "").Return([]*api.StorageObject{}, "", nil).Maybe()

	// Execute the method
	wallet, inventory, reward, isSandbox, err := economy.PurchaseItem(ctx, logger, nil, nk, userID, itemID, EconomyStoreType_ECONOMY_STORE_TYPE_GOOGLE_PLAY, receipt)

	// Verify results
	require.NoError(t, err)
	assert.Equal(t, map[string]int64{"gems": 50}, wallet)
	assert.NotNil(t, inventory)
	assert.NotNil(t, reward)
	assert.True(t, isSandbox)
	assert.Equal(t, int64(50), reward.Currencies["gems"])

	// Verify expectations
	nk.AssertExpectations(t)
}

func TestPurchaseItem_WithPurchaseIntent(t *testing.T) {
	// Setup
	config := &EconomyConfig{
		StoreItems: map[string]*EconomyConfigStoreItem{
			"vip_pass": {
				Name:        "VIP Pass",
				Description: "Unlock VIP features",
				Cost: &EconomyConfigStoreItemCost{
					Sku: "com.example.vippass",
				},
				Reward: &EconomyConfigReward{
					Guaranteed: &EconomyConfigRewardContents{
						Items: map[string]*EconomyConfigRewardItem{
							"vip_badge": {
								EconomyConfigRewardRangeInt64: EconomyConfigRewardRangeInt64{Min: 1, Max: 1},
								StringProperties:              nil,
								NumericProperties:             nil,
							},
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
	userID := "user3"
	itemID := "vip_pass"
	receipt := "valid_receipt_with_intent"

	// Create a mock purchase intent stored in Nakama
	intentObj := &api.StorageObject{
		Collection: "purchase_intents",
		Key:        "purchase_intent:user3:vip_pass",
		UserId:     userID,
		Value:      `{"user_id":"user3","item_id":"vip_pass","store_type":"ECONOMY_STORE_TYPE_APPLE_APPSTORE","sku":"com.example.vippass","created_at":1625000000,"expires_at":1625003600,"status":"pending","is_consumed":false}`,
		Version:    "v1",
	}

	// Mock the purchase validation response
	validatedPurchase := &api.ValidatedPurchase{
		ProductId:     "com.example.vippass",
		TransactionId: "tx_intent_123",
		Store:         api.StoreProvider_APPLE_APP_STORE,
		PurchaseTime:  &timestamppb.Timestamp{Seconds: time.Now().Unix()},
		Environment:   api.StoreEnvironment_PRODUCTION,
	}

	validationResponse := &api.ValidatePurchaseResponse{
		ValidatedPurchases: []*api.ValidatedPurchase{validatedPurchase},
	}

	// Setup expectations
	nk.On("StorageRead", ctx, mock.MatchedBy(func(reqs []*runtime.StorageRead) bool {
		return len(reqs) > 0 && reqs[0].Key == "purchase_intent:user3:vip_pass"
	})).Return([]*api.StorageObject{intentObj}, nil)
	nk.On("StorageRead", ctx, mock.Anything).Return([]*api.StorageObject{}, nil).Maybe()
	nk.On("PurchaseValidateApple", ctx, userID, receipt, true, []string(nil)).Return(validationResponse, nil)
	nk.On("StorageWrite", ctx, mock.Anything).Return([]*api.StorageObjectAck{}, nil).Maybe()
	nk.On("WalletUpdate", ctx, userID, mock.Anything, mock.Anything, false).Return(
		map[string]int64{}, map[string]int64{}, nil)
	nk.On("AccountGetId", ctx, userID).Return(&api.Account{Wallet: `{}`}, nil)
	nk.On("StorageList", ctx, "", userID, "inventory", 100, "").Return([]*api.StorageObject{}, "", nil)

	// Execute the method
	wallet, inventory, reward, isSandbox, err := economy.PurchaseItem(ctx, logger, nil, nk, userID, itemID, EconomyStoreType_ECONOMY_STORE_TYPE_APPLE_APPSTORE, receipt)

	// Verify results
	require.NoError(t, err)
	assert.Equal(t, map[string]int64{}, wallet)
	assert.NotNil(t, inventory)
	assert.NotNil(t, reward)
	assert.False(t, isSandbox)
	assert.Equal(t, int64(1), reward.Items["vip_badge"])

	// Verify expectations
	nk.AssertExpectations(t)
}

func TestPurchaseItem_InvalidReceipt(t *testing.T) {
	config := &EconomyConfig{
		StoreItems: map[string]*EconomyConfigStoreItem{
			"item1": {
				Name: "Test Item",
				Cost: &EconomyConfigStoreItemCost{
					Sku: "com.example.item1",
				},
			},
		},
	}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"
	itemID := "item1"
	receipt := "invalid_receipt"

	// Setup expectations for invalid receipt
	nk.On("PurchaseValidateApple", ctx, userID, receipt, true, []string(nil)).Return(&api.ValidatePurchaseResponse{}, assert.AnError)
	nk.On("StorageRead", ctx, mock.Anything).Return([]*api.StorageObject{}, nil)

	// Execute the method
	wallet, inventory, reward, isSandbox, err := economy.PurchaseItem(ctx, logger, nil, nk, userID, itemID, EconomyStoreType_ECONOMY_STORE_TYPE_APPLE_APPSTORE, receipt)

	// Verify results
	assert.Error(t, err)
	assert.Nil(t, inventory)
	assert.Nil(t, reward)
	assert.False(t, isSandbox)
	assert.Empty(t, wallet)

	// Verify expectations
	nk.AssertExpectations(t)
}

func TestPurchaseItem_ItemNotFound(t *testing.T) {
	config := &EconomyConfig{
		StoreItems: map[string]*EconomyConfigStoreItem{
			"item1": {Name: "Test Item"},
		},
	}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"
	itemID := "non_existent_item"
	receipt := "valid_receipt"

	// Execute the method
	wallet, inventory, reward, isSandbox, err := economy.PurchaseItem(ctx, logger, nil, nk, userID, itemID, EconomyStoreType_ECONOMY_STORE_TYPE_APPLE_APPSTORE, receipt)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Nil(t, inventory)
	assert.Nil(t, reward)
	assert.False(t, isSandbox)
	assert.Empty(t, wallet)
}

func TestPurchaseItem_UnavailableItem(t *testing.T) {
	config := &EconomyConfig{
		StoreItems: map[string]*EconomyConfigStoreItem{
			"unavailable_item": {
				Name:        "Unavailable Item",
				Unavailable: true,
			},
		},
	}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"
	itemID := "unavailable_item"
	receipt := "valid_receipt"

	// Execute the method
	wallet, inventory, reward, isSandbox, err := economy.PurchaseItem(ctx, logger, nil, nk, userID, itemID, EconomyStoreType_ECONOMY_STORE_TYPE_APPLE_APPSTORE, receipt)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is unavailable")
	assert.Nil(t, inventory)
	assert.Nil(t, reward)
	assert.False(t, isSandbox)
	assert.Empty(t, wallet)
}

func TestPurchaseItem_DisabledItem(t *testing.T) {
	config := &EconomyConfig{
		StoreItems: map[string]*EconomyConfigStoreItem{
			"disabled_item": {
				Name:     "Disabled Item",
				Disabled: true,
			},
		},
	}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"
	itemID := "disabled_item"
	receipt := "valid_receipt"

	// Execute the method
	wallet, inventory, reward, isSandbox, err := economy.PurchaseItem(ctx, logger, nil, nk, userID, itemID, EconomyStoreType_ECONOMY_STORE_TYPE_APPLE_APPSTORE, receipt)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is disabled")
	assert.Nil(t, inventory)
	assert.Nil(t, reward)
	assert.False(t, isSandbox)
	assert.Empty(t, wallet)
}

func TestPurchaseItem_ConsumedIntent(t *testing.T) {
	config := &EconomyConfig{
		StoreItems: map[string]*EconomyConfigStoreItem{
			"item1": {
				Name: "Test Item",
				Cost: &EconomyConfigStoreItemCost{
					Sku: "com.example.item1",
				},
			},
		},
	}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"
	itemID := "item1"
	receipt := "valid_receipt"

	// Create a mock purchase intent already consumed
	intentObj := &api.StorageObject{
		Collection: "purchase_intents",
		Key:        "purchase_intent:user1:item1",
		UserId:     userID,
		Value:      `{"user_id":"user1","item_id":"item1","store_type":"ECONOMY_STORE_TYPE_APPLE_APPSTORE","sku":"com.example.item1","created_at":1625000000,"expires_at":1625003600,"status":"completed","is_consumed":true}`,
		Version:    "v1",
	}

	// Setup expectations
	nk.On("StorageRead", ctx, mock.MatchedBy(func(reqs []*runtime.StorageRead) bool {
		return len(reqs) > 0 && reqs[0].Key == "purchase_intent:user1:item1"
	})).Return([]*api.StorageObject{intentObj}, nil)

	// Execute the method
	wallet, inventory, reward, isSandbox, err := economy.PurchaseItem(ctx, logger, nil, nk, userID, itemID, EconomyStoreType_ECONOMY_STORE_TYPE_APPLE_APPSTORE, receipt)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already consumed")
	assert.Nil(t, inventory)
	assert.Nil(t, reward)
	assert.False(t, isSandbox)
	assert.Empty(t, wallet)

	// Verify expectations
	nk.AssertExpectations(t)
}

func TestPurchaseItem_UnsupportedStore(t *testing.T) {
	config := &EconomyConfig{
		StoreItems: map[string]*EconomyConfigStoreItem{
			"item1": {Name: "Test Item"},
		},
	}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"
	itemID := "item1"
	receipt := "valid_receipt"

	// Add mock for StorageRead
	nk.On("StorageRead", ctx, mock.Anything).Return([]*api.StorageObject{}, nil)

	// Execute the method with unsupported store
	wallet, inventory, reward, isSandbox, err := economy.PurchaseItem(ctx, logger, nil, nk, userID, itemID, EconomyStoreType(99), receipt)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported store type")
	assert.Nil(t, inventory)
	assert.Nil(t, reward)
	assert.False(t, isSandbox)
	assert.Empty(t, wallet)
}

func TestPurchaseRestore_Success(t *testing.T) {
	config := &EconomyConfig{}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"
	receipts := []string{"receipt1", "receipt2"}

	// Mock existing transactions (empty to simulate no previous transactions)
	nk.On("StorageList", ctx, "", userID, "purchase_transactions", mock.Anything, "").Return([]*api.StorageObject{}, "", nil)

	// Mock validation responses
	validatedPurchase1 := &api.ValidatedPurchase{
		ProductId:     "com.example.item1",
		TransactionId: "trans1",
		Store:         api.StoreProvider_APPLE_APP_STORE,
	}
	validatedPurchase2 := &api.ValidatedPurchase{
		ProductId:     "com.example.item2",
		TransactionId: "trans2",
		Store:         api.StoreProvider_APPLE_APP_STORE,
	}

	response1 := &api.ValidatePurchaseResponse{
		ValidatedPurchases: []*api.ValidatedPurchase{validatedPurchase1},
	}
	response2 := &api.ValidatePurchaseResponse{
		ValidatedPurchases: []*api.ValidatedPurchase{validatedPurchase2},
	}

	// Setup validation expectations
	nk.On("PurchaseValidateApple", ctx, userID, "receipt1", true, []string(nil)).Return(response1, nil)
	nk.On("PurchaseValidateApple", ctx, userID, "receipt2", true, []string(nil)).Return(response2, nil)

	// Mock storage write for recording transactions
	nk.On("StorageWrite", ctx, mock.Anything).Return([]*api.StorageObjectAck{}, nil).Maybe()

	// Execute the method
	err := economy.PurchaseRestore(ctx, logger, nk, userID, EconomyStoreType_ECONOMY_STORE_TYPE_APPLE_APPSTORE, receipts)

	// Verify results
	require.NoError(t, err)

	// Verify expectations
	nk.AssertExpectations(t)
}

func TestPurchaseRestore_ExistingTransactions(t *testing.T) {
	config := &EconomyConfig{}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"
	receipts := []string{"receipt1", "receipt2"}

	// Mock existing transactions (one already processed)
	existingTransaction := &api.StorageObject{
		Collection: "purchase_transactions",
		Key:        "transaction1",
		UserId:     userID,
		Value:      `{"id":"transaction1","user_id":"user1","receipt":"receipt1","transaction_id":"trans1"}`,
	}

	nk.On("StorageList", ctx, "", userID, "purchase_transactions", mock.Anything, "").Return([]*api.StorageObject{existingTransaction}, "", nil)

	// Mock validation responses
	validatedPurchase1 := &api.ValidatedPurchase{
		ProductId:     "com.example.item1",
		TransactionId: "trans1",
		Store:         api.StoreProvider_APPLE_APP_STORE,
	}
	validatedPurchase2 := &api.ValidatedPurchase{
		ProductId:     "com.example.item2",
		TransactionId: "trans2",
		Store:         api.StoreProvider_APPLE_APP_STORE,
	}

	response1 := &api.ValidatePurchaseResponse{
		ValidatedPurchases: []*api.ValidatedPurchase{validatedPurchase1},
	}
	response2 := &api.ValidatePurchaseResponse{
		ValidatedPurchases: []*api.ValidatedPurchase{validatedPurchase2},
	}

	// Setup validation expectations - for both receipts
	nk.On("PurchaseValidateApple", ctx, userID, "receipt1", true, []string(nil)).Return(response1, nil)
	nk.On("PurchaseValidateApple", ctx, userID, "receipt2", true, []string(nil)).Return(response2, nil)

	// Mock storage write for recording transactions
	nk.On("StorageWrite", ctx, mock.Anything).Return([]*api.StorageObjectAck{}, nil).Maybe()

	// Execute the method
	err := economy.PurchaseRestore(ctx, logger, nk, userID, EconomyStoreType_ECONOMY_STORE_TYPE_APPLE_APPSTORE, receipts)

	// Verify results
	require.NoError(t, err)

	// Verify expectations
	nk.AssertExpectations(t)
}

func TestPurchaseRestore_ValidationError(t *testing.T) {
	config := &EconomyConfig{}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"
	receipts := []string{"receipt1", "receipt2"}

	// Mock existing transactions (empty)
	nk.On("StorageList", ctx, "", userID, "purchase_transactions", mock.Anything, "").Return([]*api.StorageObject{}, "", nil)

	// Setup validation expectations with error for first receipt
	nk.On("PurchaseValidateApple", ctx, userID, "receipt1", true, []string(nil)).Return(&api.ValidatePurchaseResponse{}, assert.AnError)

	// Second receipt validation should still be attempted
	validatedPurchase2 := &api.ValidatedPurchase{
		ProductId:     "com.example.item2",
		TransactionId: "trans2",
		Store:         api.StoreProvider_APPLE_APP_STORE,
	}
	response2 := &api.ValidatePurchaseResponse{
		ValidatedPurchases: []*api.ValidatedPurchase{validatedPurchase2},
	}
	nk.On("PurchaseValidateApple", ctx, userID, "receipt2", true, []string(nil)).Return(response2, nil)

	// Mock storage write for recording transactions
	nk.On("StorageWrite", ctx, mock.Anything).Return([]*api.StorageObjectAck{}, nil).Maybe()

	// Execute the method
	err := economy.PurchaseRestore(ctx, logger, nk, userID, EconomyStoreType_ECONOMY_STORE_TYPE_APPLE_APPSTORE, receipts)

	// Verify results - should still succeed overall as we process each receipt independently
	require.NoError(t, err)

	// Verify expectations
	nk.AssertExpectations(t)
}

func TestPurchaseRestore_EmptyReceiptsList(t *testing.T) {
	config := &EconomyConfig{}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	userID := "user1"

	// Execute with empty receipts list
	err := economy.PurchaseRestore(ctx, logger, nk, userID, EconomyStoreType_ECONOMY_STORE_TYPE_APPLE_APPSTORE, []string{})

	// Should return an error for empty receipts
	assert.Error(t, err)
}

func TestPurchaseRestore_EmptyUserID(t *testing.T) {
	config := &EconomyConfig{}
	economy := NewNakamaEconomySystem(config)
	logger := &mockLogger{}
	nk := NewMockNakama(t)
	ctx := context.Background()
	receipts := []string{"receipt1"}

	// Execute with empty user ID
	err := economy.PurchaseRestore(ctx, logger, nk, "", EconomyStoreType_ECONOMY_STORE_TYPE_APPLE_APPSTORE, receipts)

	// Should return an error for empty user ID
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user ID is empty")
}

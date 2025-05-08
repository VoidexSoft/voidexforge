package pamlogix

import (
	"context"
	"strings"
	"testing"

	"github.com/heroiclabs/nakama-common/api"
	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testNakamaModule is a test double for runtime.NakamaModule
// Only implements the methods needed for the tests.
type testNakamaModule struct {
	runtime.NakamaModule
	storageData map[string]string // map of collection:key:userID -> value
}

func newTestNakama() *testNakamaModule {
	return &testNakamaModule{
		storageData: make(map[string]string),
	}
}

// StorageRead implementation for testing
func (n *testNakamaModule) StorageRead(ctx context.Context, reads []*runtime.StorageRead) ([]*api.StorageObject, error) {
	result := make([]*api.StorageObject, 0, len(reads))
	for _, read := range reads {
		key := formatStorageKey(read.Collection, read.Key, read.UserID)
		value, exists := n.storageData[key]
		if exists {
			result = append(result, &api.StorageObject{
				Collection:      read.Collection,
				Key:             read.Key,
				UserId:          read.UserID,
				Value:           value,
				Version:         "1",
				PermissionRead:  1,
				PermissionWrite: 0,
				CreateTime:      nil,
				UpdateTime:      nil,
			})
		}
	}
	return result, nil
}

// StorageWrite implementation for testing
func (n *testNakamaModule) StorageWrite(ctx context.Context, writes []*runtime.StorageWrite) ([]*api.StorageObjectAck, error) {
	result := make([]*api.StorageObjectAck, 0, len(writes))
	for _, write := range writes {
		key := formatStorageKey(write.Collection, write.Key, write.UserID)
		n.storageData[key] = write.Value
		result = append(result, &api.StorageObjectAck{
			Collection: write.Collection,
			Key:        write.Key,
			UserId:     write.UserID,
			Version:    "1",
			CreateTime: nil,
			UpdateTime: nil,
		})
	}
	return result, nil
}

// StorageList implementation for testing
func (n *testNakamaModule) StorageList(ctx context.Context, collection, userID, prefix string, limit int, cursor string) ([]*api.StorageObject, string, error) {
	result := make([]*api.StorageObject, 0)
	for key, value := range n.storageData {
		// key format: collection:key:userID
		parts := splitStorageKey(key)
		if len(parts) != 3 {
			continue
		}
		if parts[0] != collection && collection != "" {
			continue
		}
		if parts[2] != userID && userID != "" {
			continue
		}
		if prefix != "" && !hasPrefix(parts[1], prefix) {
			continue
		}
		result = append(result, &api.StorageObject{
			Collection: parts[0],
			Key:        parts[1],
			UserId:     parts[2],
			Value:      value,
			Version:    "1",
		})
	}
	return result, "", nil
}

// Helper function to format a storage key
func formatStorageKey(collection, key, userID string) string {
	return collection + ":" + key + ":" + userID
}

// Helper to split a storage key
func splitStorageKey(fullKey string) []string {
	return strings.SplitN(fullKey, ":", 3)
}

// Helper to check prefix
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

func TestRewardRoll_GuaranteedCurrency(t *testing.T) {
	economy := NewNakamaEconomySystem(nil)
	logger := &mockLogger{}
	nk := newTestNakama()
	ctx := context.Background()
	userID := "user1"

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
}

func TestRewardGrant_CurrencyAndItem(t *testing.T) {
	economy := NewNakamaEconomySystem(nil)
	logger := &mockLogger{}
	nk := newTestNakama()
	ctx := context.Background()
	userID := "user2"

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

	// Grant again to test update
	_, updatedItems, _, err = economy.RewardGrant(ctx, logger, nk, userID, reward, nil, false)
	require.NoError(t, err)
	assert.Len(t, updatedItems, 1)
	assert.Contains(t, updatedItems, "potion")
	assert.Equal(t, int64(4), updatedItems["potion"].Count)
}

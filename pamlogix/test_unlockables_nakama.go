package pamlogix

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/heroiclabs/nakama-common/api"
	"github.com/heroiclabs/nakama-common/runtime"
)

// Custom test Nakama module for unlockables tests
type TestUnlockablesNakama struct {
	*MockNakamaModule
	storage map[string]string
	wallets map[string]map[string]int64 // userID -> currencies
}

// Create a new test module with storage capabilities specifically for unlockables tests
func NewTestUnlockablesNakama(t *testing.T) *TestUnlockablesNakama {
	return &TestUnlockablesNakama{
		MockNakamaModule: NewMockNakama(t),
		storage:          make(map[string]string),
		wallets:          make(map[string]map[string]int64),
	}
}

// Override StorageRead with a custom implementation for unlockables tests
func (m *TestUnlockablesNakama) StorageRead(ctx context.Context, reads []*runtime.StorageRead) ([]*api.StorageObject, error) {
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
		} else {
			// Return an empty storage object for non-existent keys
			// This ensures unlockables code works properly for new users
			result = append(result, &api.StorageObject{
				Collection: r.Collection,
				Key:        r.Key,
				UserId:     r.UserID,
				Value:      "",
				Version:    "v1",
			})
		}
	}
	return result, nil
}

// Override StorageWrite with a custom implementation for unlockables tests
func (m *TestUnlockablesNakama) StorageWrite(ctx context.Context, writes []*runtime.StorageWrite) ([]*api.StorageObjectAck, error) {
	var acks []*api.StorageObjectAck
	for _, w := range writes {
		// Validate JSON data before storage to catch issues early
		var jsonObj interface{}
		if err := json.Unmarshal([]byte(w.Value), &jsonObj); err != nil {
			return nil, err
		}

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

// Override AccountGetId to support wallet operations
func (m *TestUnlockablesNakama) AccountGetId(ctx context.Context, userID string) (*api.Account, error) {
	// Create a wallet if it doesn't exist
	if _, ok := m.wallets[userID]; !ok {
		m.wallets[userID] = map[string]int64{
			"gems": 1000,
			"gold": 10000,
		}
	}

	// Convert the wallet to JSON
	walletJSON, err := json.Marshal(m.wallets[userID])
	if err != nil {
		return nil, err
	}

	// Create a mock account with the wallet
	account := &api.Account{
		User: &api.User{
			Id:       userID,
			Username: "test_user",
		},
		Wallet: string(walletJSON),
	}

	return account, nil
}

// Override WalletUpdate to support wallet operations
func (m *TestUnlockablesNakama) WalletUpdate(ctx context.Context, userID string, changeset map[string]int64, metadata map[string]interface{}, updateLedger bool) (updated map[string]int64, previous map[string]int64, err error) {
	// Create a wallet if it doesn't exist
	if _, ok := m.wallets[userID]; !ok {
		m.wallets[userID] = map[string]int64{
			"gems": 1000,
			"gold": 10000,
		}
	}

	// Store the previous wallet values
	previous = make(map[string]int64)
	for k, v := range m.wallets[userID] {
		previous[k] = v
	}

	// Apply the changeset
	for k, v := range changeset {
		if _, ok := m.wallets[userID][k]; !ok {
			m.wallets[userID][k] = 0
		}
		m.wallets[userID][k] += v
	}

	// Return the updated wallet
	updated = make(map[string]int64)
	for k, v := range m.wallets[userID] {
		updated[k] = v
	}

	return updated, previous, nil
}

package pamlogix

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/heroiclabs/nakama-common/api"
	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/stretchr/testify/assert"
	// Keep this as it's used by MockNakamaModule
)

// mockLogger is assumed to be defined in another test file in the same package (e.g., economy_pamlogix_test.go)

// testStatsNakama is a custom implementation for testing the stats system
type testStatsNakama struct {
	*MockNakamaModule
	storage map[string]string
}

// StorageRead implements a simple storage lookup based on the in-memory map
func (m *testStatsNakama) StorageRead(ctx context.Context, objectIDs []*runtime.StorageRead) ([]*api.StorageObject, error) {
	var result []*api.StorageObject
	for _, read := range objectIDs {
		key := read.Collection + ":" + read.Key + ":" + read.UserID
		if val, ok := m.storage[key]; ok {
			result = append(result, &api.StorageObject{
				Collection: read.Collection,
				Key:        read.Key,
				UserId:     read.UserID,
				Value:      val,
				Version:    "v1",
			})
		}
	}
	return result, nil
}

// StorageWrite implements writing to the in-memory map
func (m *testStatsNakama) StorageWrite(ctx context.Context, writes []*runtime.StorageWrite) ([]*api.StorageObjectAck, error) {
	var acks []*api.StorageObjectAck
	for _, write := range writes {
		key := write.Collection + ":" + write.Key + ":" + write.UserID
		m.storage[key] = write.Value
		acks = append(acks, &api.StorageObjectAck{
			Collection: write.Collection,
			Key:        write.Key,
			UserId:     write.UserID,
			Version:    "v1",
		})
	}
	return acks, nil
}

func TestNakamaStatsSystem_List_AndUpdate(t *testing.T) {
	ctx := context.Background()
	logger := &mockLogger{}

	// Create a custom mock that implements the interface directly
	originalMock := NewMockNakama(t)
	nk := &testStatsNakama{
		MockNakamaModule: originalMock,
		storage:          make(map[string]string),
	}

	config := &StatsConfig{
		Whitelist: []string{"strength", "dexterity"},
	}
	statsSystem := NewStatsSystem(config)

	userID := "test-user-id"

	// 1. Initial List - should be empty
	initialStatsMap, err := statsSystem.List(ctx, logger, nk, userID, []string{userID})
	assert.NoError(t, err)
	assert.NotNil(t, initialStatsMap)
	initialUserStats, ok := initialStatsMap[userID]
	assert.True(t, ok)
	assert.NotNil(t, initialUserStats)
	assert.Empty(t, initialUserStats.Public)
	assert.Empty(t, initialUserStats.Private)

	// 2. Update some stats
	publicUpdates := []*StatUpdate{
		{Name: "strength", Value: 10, Operator: StatUpdateOperator_STAT_UPDATE_OPERATOR_SET},
		{Name: "level", Value: 1, Operator: StatUpdateOperator_STAT_UPDATE_OPERATOR_SET}, // Not whitelisted initially but should be added
	}
	privateUpdates := []*StatUpdate{
		{Name: "secret_power", Value: 100, Operator: StatUpdateOperator_STAT_UPDATE_OPERATOR_SET},
	}

	updatedStats, err := statsSystem.Update(ctx, logger, nk, userID, publicUpdates, privateUpdates)
	assert.NoError(t, err)
	assert.NotNil(t, updatedStats)

	// Check public stats
	assert.Len(t, updatedStats.Public, 2)
	strengthStat, ok := updatedStats.Public["strength"]
	assert.True(t, ok)
	assert.EqualValues(t, 10, strengthStat.Value)
	assert.EqualValues(t, 1, strengthStat.Count)
	assert.EqualValues(t, 10, strengthStat.Total)
	assert.EqualValues(t, 10, strengthStat.Min)
	assert.EqualValues(t, 10, strengthStat.Max)
	assert.EqualValues(t, 10, strengthStat.First)
	assert.EqualValues(t, 10, strengthStat.Last)
	assert.True(t, strengthStat.Public)
	assert.Greater(t, strengthStat.UpdateTimeSec, int64(0))

	levelStat, ok := updatedStats.Public["level"]
	assert.True(t, ok)
	assert.EqualValues(t, 1, levelStat.Value)

	// Check private stats
	assert.Len(t, updatedStats.Private, 1)
	secretPowerStat, ok := updatedStats.Private["secret_power"]
	assert.True(t, ok)
	assert.EqualValues(t, 100, secretPowerStat.Value)
	assert.False(t, secretPowerStat.Public)

	// 3. List again - should reflect updates
	listedStatsMap, err := statsSystem.List(ctx, logger, nk, userID, []string{userID})
	assert.NoError(t, err)
	listedUserStats, ok := listedStatsMap[userID]
	assert.True(t, ok)
	assert.EqualValues(t, updatedStats, listedUserStats)

	// 4. Update existing stat with DELTA
	deltaStrengthUpdate := []*StatUpdate{
		{Name: "strength", Value: 5, Operator: StatUpdateOperator_STAT_UPDATE_OPERATOR_DELTA},
	}
	deltaUpdatedStats, err := statsSystem.Update(ctx, logger, nk, userID, deltaStrengthUpdate, nil)
	assert.NoError(t, err)
	assert.NotNil(t, deltaUpdatedStats)
	strengthStat = deltaUpdatedStats.Public["strength"]
	assert.EqualValues(t, 15, strengthStat.Value) // 10 + 5
	assert.EqualValues(t, 2, strengthStat.Count)
	assert.EqualValues(t, 15, strengthStat.Total) // 10 + 5
	assert.EqualValues(t, 10, strengthStat.Min)
	assert.EqualValues(t, 15, strengthStat.Max)
	assert.EqualValues(t, 10, strengthStat.First) // First remains the initial set value
	assert.EqualValues(t, 5, strengthStat.Last)   // Last is the delta value

	// 5. Update with MIN and MAX
	minMaxUpdates := []*StatUpdate{
		{Name: "strength", Value: 5, Operator: StatUpdateOperator_STAT_UPDATE_OPERATOR_MIN},   // Current 15, min 5 -> value becomes 5
		{Name: "level", Value: 10, Operator: StatUpdateOperator_STAT_UPDATE_OPERATOR_MAX},     // Current 1, max 10 -> value becomes 10
		{Name: "dexterity", Value: 20, Operator: StatUpdateOperator_STAT_UPDATE_OPERATOR_SET}, // New stat
	}

	minMaxUpdatedStats, err := statsSystem.Update(ctx, logger, nk, userID, minMaxUpdates, nil)
	assert.NoError(t, err)
	assert.NotNil(t, minMaxUpdatedStats)

	strengthStat = minMaxUpdatedStats.Public["strength"]
	assert.EqualValues(t, 5, strengthStat.Value)
	assert.EqualValues(t, 3, strengthStat.Count)
	assert.EqualValues(t, 20, strengthStat.Total) // 10 + 5 + 5
	assert.EqualValues(t, 5, strengthStat.Min)
	assert.EqualValues(t, 15, strengthStat.Max)
	assert.EqualValues(t, 10, strengthStat.First)
	assert.EqualValues(t, 5, strengthStat.Last)

	levelStat = minMaxUpdatedStats.Public["level"]
	assert.EqualValues(t, 10, levelStat.Value)
	assert.EqualValues(t, 2, levelStat.Count)

	dexterityStat, ok := minMaxUpdatedStats.Public["dexterity"]
	assert.True(t, ok)
	assert.EqualValues(t, 20, dexterityStat.Value)

	// Test with multiple users for List
	userID2 := "test-user-id-2"
	// Update stats for userID2
	user2Updates := []*StatUpdate{
		{Name: "mana", Value: 50, Operator: StatUpdateOperator_STAT_UPDATE_OPERATOR_SET},
	}
	_, err = statsSystem.Update(ctx, logger, nk, userID2, user2Updates, nil)
	assert.NoError(t, err)

	multiUserStatsMap, err := statsSystem.List(ctx, logger, nk, userID, []string{userID, userID2})
	assert.NoError(t, err)
	assert.Len(t, multiUserStatsMap, 2)
	assert.Contains(t, multiUserStatsMap, userID)
	assert.Contains(t, multiUserStatsMap, userID2)

	user1ListedStats := multiUserStatsMap[userID]
	assert.EqualValues(t, dexterityStat.Value, user1ListedStats.Public["dexterity"].Value)

	user2ListedStats := multiUserStatsMap[userID2]
	assert.NotNil(t, user2ListedStats)
	assert.Len(t, user2ListedStats.Public, 1)
	manaStat, ok := user2ListedStats.Public["mana"]
	assert.True(t, ok)
	assert.EqualValues(t, 50, manaStat.Value)
}

func TestRpcStatsGetAndUpdate(t *testing.T) {
	ctx := context.Background()
	logger := &mockLogger{}

	// Create a custom mock that implements the interface directly
	originalMock := NewMockNakama(t)
	nkModule := &testStatsNakama{
		MockNakamaModule: originalMock,
		storage:          make(map[string]string),
	}

	p := &pamlogixImpl{
		systems: make(map[SystemType]System),
	}

	statsConfig := &StatsConfig{}
	statsSystem := NewStatsSystem(statsConfig)
	p.systems[SystemTypeStats] = statsSystem

	rpcGet := rpcStatsGet(p)
	rpcUpd := rpcStatsUpdate(p)

	userID := "rpc-user-id"
	ctx = context.WithValue(ctx, runtime.RUNTIME_CTX_USER_ID, userID)

	// 1. RPC Get - Initial
	getPayload := ""
	getResult, err := rpcGet(ctx, logger, nil, nkModule, getPayload)
	assert.NoError(t, err)
	var initialStatsList StatList
	err = json.Unmarshal([]byte(getResult), &initialStatsList)
	assert.NoError(t, err)
	assert.Empty(t, initialStatsList.Public)
	assert.Empty(t, initialStatsList.Private)

	// 2. RPC Update
	updateReq := StatUpdateRequest{
		Public: []*StatUpdate{
			{Name: "gold", Value: 1000, Operator: StatUpdateOperator_STAT_UPDATE_OPERATOR_SET},
		},
		Private: []*StatUpdate{
			{Name: "gems", Value: 50, Operator: StatUpdateOperator_STAT_UPDATE_OPERATOR_SET},
		},
	}
	updatePayloadBytes, _ := json.Marshal(updateReq)
	updateResult, err := rpcUpd(ctx, logger, nil, nkModule, string(updatePayloadBytes))
	assert.NoError(t, err)
	var updatedStatsList StatList
	err = json.Unmarshal([]byte(updateResult), &updatedStatsList)
	assert.NoError(t, err)

	assert.Len(t, updatedStatsList.Public, 1)
	goldStat, ok := updatedStatsList.Public["gold"]
	assert.True(t, ok)
	assert.EqualValues(t, 1000, goldStat.Value)

	assert.Len(t, updatedStatsList.Private, 1)
	gemsStat, ok := updatedStatsList.Private["gems"]
	assert.True(t, ok)
	assert.EqualValues(t, 50, gemsStat.Value)

	// 3. RPC Get - After update
	getResultAfterUpdate, err := rpcGet(ctx, logger, nil, nkModule, getPayload)
	assert.NoError(t, err)
	var fetchedStatsList StatList
	err = json.Unmarshal([]byte(getResultAfterUpdate), &fetchedStatsList)
	assert.NoError(t, err)
	assert.EqualValues(t, updatedStatsList, fetchedStatsList)
}

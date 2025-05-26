package pamlogix_test

import (
	"context"
	"testing"

	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"voidexforge/pamlogix"
)

// Mock UnlockablesSystem for testing
type mockUnlockablesSystem struct {
	mock.Mock
}

func (m *mockUnlockablesSystem) GetType() pamlogix.SystemType {
	return pamlogix.SystemTypeUnlockables
}

func (m *mockUnlockablesSystem) GetConfig() any {
	args := m.Called()
	return args.Get(0)
}

func (m *mockUnlockablesSystem) SetPamlogix(pl pamlogix.Pamlogix) {
	m.Called(pl)
}

func (m *mockUnlockablesSystem) Create(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, unlockableID string, unlockableConfig *pamlogix.UnlockablesConfigUnlockable) (*pamlogix.UnlockablesList, error) {
	args := m.Called(ctx, logger, nk, userID, unlockableID, unlockableConfig)
	return args.Get(0).(*pamlogix.UnlockablesList), args.Error(1)
}

func (m *mockUnlockablesSystem) Get(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (*pamlogix.UnlockablesList, error) {
	args := m.Called(ctx, logger, nk, userID)
	return args.Get(0).(*pamlogix.UnlockablesList), args.Error(1)
}

func (m *mockUnlockablesSystem) UnlockAdvance(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, instanceID string, seconds int64) (*pamlogix.UnlockablesList, error) {
	args := m.Called(ctx, logger, nk, userID, instanceID, seconds)
	return args.Get(0).(*pamlogix.UnlockablesList), args.Error(1)
}

func (m *mockUnlockablesSystem) UnlockStart(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, instanceID string) (*pamlogix.UnlockablesList, error) {
	args := m.Called(ctx, logger, nk, userID, instanceID)
	return args.Get(0).(*pamlogix.UnlockablesList), args.Error(1)
}

func (m *mockUnlockablesSystem) PurchaseUnlock(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, instanceID string) (*pamlogix.UnlockablesList, error) {
	args := m.Called(ctx, logger, nk, userID, instanceID)
	return args.Get(0).(*pamlogix.UnlockablesList), args.Error(1)
}

func (m *mockUnlockablesSystem) PurchaseSlot(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (*pamlogix.UnlockablesList, error) {
	args := m.Called(ctx, logger, nk, userID)
	return args.Get(0).(*pamlogix.UnlockablesList), args.Error(1)
}

func (m *mockUnlockablesSystem) Claim(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, instanceID string) (*pamlogix.UnlockablesReward, error) {
	args := m.Called(ctx, logger, nk, userID, instanceID)
	return args.Get(0).(*pamlogix.UnlockablesReward), args.Error(1)
}

func (m *mockUnlockablesSystem) QueueAdd(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, instanceIDs []string) (*pamlogix.UnlockablesList, error) {
	args := m.Called(ctx, logger, nk, userID, instanceIDs)
	return args.Get(0).(*pamlogix.UnlockablesList), args.Error(1)
}

func (m *mockUnlockablesSystem) QueueRemove(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, instanceIDs []string) (*pamlogix.UnlockablesList, error) {
	args := m.Called(ctx, logger, nk, userID, instanceIDs)
	return args.Get(0).(*pamlogix.UnlockablesList), args.Error(1)
}

func (m *mockUnlockablesSystem) QueueSet(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, instanceIDs []string) (*pamlogix.UnlockablesList, error) {
	args := m.Called(ctx, logger, nk, userID, instanceIDs)
	return args.Get(0).(*pamlogix.UnlockablesList), args.Error(1)
}

func (m *mockUnlockablesSystem) SetOnClaimReward(fn pamlogix.OnReward[*pamlogix.UnlockablesConfigUnlockable]) {
	m.Called(fn)
}

// Test logger implementation
type testLoggerPublisher struct{}

func (l *testLoggerPublisher) Debug(format string, v ...interface{})                   {}
func (l *testLoggerPublisher) Info(format string, v ...interface{})                    {}
func (l *testLoggerPublisher) Warn(format string, v ...interface{})                    {}
func (l *testLoggerPublisher) Error(format string, v ...interface{})                   {}
func (l *testLoggerPublisher) WithField(key string, value interface{}) runtime.Logger  { return l }
func (l *testLoggerPublisher) WithFields(fields map[string]interface{}) runtime.Logger { return l }
func (l *testLoggerPublisher) Fields() map[string]interface{}                          { return map[string]interface{}{} }

// Test basic publisher creation
func TestUnlockableRewardedVideoPublisher_Creation(t *testing.T) {
	mockUnlockables := &mockUnlockablesSystem{}

	publisher := &pamlogix.UnlockableRewardedVideoPublisher{
		Unlockables: mockUnlockables,
	}

	assert.NotNil(t, publisher)
	assert.Equal(t, mockUnlockables, publisher.Unlockables)
}

// Test Authenticate method (should be no-op)
func TestUnlockableRewardedVideoPublisher_Authenticate(t *testing.T) {
	mockUnlockables := &mockUnlockablesSystem{}
	publisher := &pamlogix.UnlockableRewardedVideoPublisher{
		Unlockables: mockUnlockables,
	}

	ctx := context.Background()
	logger := &testLoggerPublisher{}
	userID := "test_user"

	// Should not panic and should be a no-op
	publisher.Authenticate(ctx, logger, nil, userID, true)
	publisher.Authenticate(ctx, logger, nil, userID, false)

	// No expectations on mock since it should be a no-op
	mockUnlockables.AssertExpectations(t)
}

// Test Send method with non-placement events (should be ignored)
func TestUnlockableRewardedVideoPublisher_Send_NonPlacementEvents(t *testing.T) {
	mockUnlockables := &mockUnlockablesSystem{}
	publisher := &pamlogix.UnlockableRewardedVideoPublisher{
		Unlockables: mockUnlockables,
	}

	ctx := context.Background()
	logger := &testLoggerPublisher{}
	userID := "test_user"

	// Test with various non-placement events
	events := []*pamlogix.PublisherEvent{
		{
			Name: "user_login",
			Metadata: map[string]string{
				"platform": "ios",
			},
		},
		{
			Name: "level_complete",
			Metadata: map[string]string{
				"level": "1",
			},
		},
		{
			Name: "achievement_unlocked",
			Metadata: map[string]string{
				"achievement_id": "first_win",
			},
		},
	}

	// Should not call any methods on the unlockables system
	publisher.Send(ctx, logger, nil, userID, events)

	// No expectations on mock since non-placement events should be ignored
	mockUnlockables.AssertExpectations(t)
}

// Test Send method with placement_success event but wrong placement_id
func TestUnlockableRewardedVideoPublisher_Send_WrongPlacementId(t *testing.T) {
	mockUnlockables := &mockUnlockablesSystem{}
	publisher := &pamlogix.UnlockableRewardedVideoPublisher{
		Unlockables: mockUnlockables,
	}

	ctx := context.Background()
	logger := &testLoggerPublisher{}
	userID := "test_user"

	events := []*pamlogix.PublisherEvent{
		{
			Name: "placement_success",
			Metadata: map[string]string{
				"placement_id": "banner_ad",
				"instance_id":  "some_instance",
			},
		},
		{
			Name: "placement_success",
			Metadata: map[string]string{
				"placement_id": "interstitial_ad",
				"instance_id":  "another_instance",
			},
		},
	}

	// Should not call PurchaseUnlock since placement_id is not "unlockable_rewarded_video"
	publisher.Send(ctx, logger, nil, userID, events)

	// No expectations on mock since wrong placement_id should be ignored
	mockUnlockables.AssertExpectations(t)
}

// Test Send method with placement_success event but missing instance_id
func TestUnlockableRewardedVideoPublisher_Send_MissingInstanceId(t *testing.T) {
	mockUnlockables := &mockUnlockablesSystem{}
	publisher := &pamlogix.UnlockableRewardedVideoPublisher{
		Unlockables: mockUnlockables,
	}

	ctx := context.Background()
	logger := &testLoggerPublisher{}
	userID := "test_user"

	events := []*pamlogix.PublisherEvent{
		{
			Name: "placement_success",
			Metadata: map[string]string{
				"placement_id": "unlockable_rewarded_video",
				// Missing instance_id
			},
		},
		{
			Name: "placement_success",
			Metadata: map[string]string{
				"placement_id": "unlockable_rewarded_video",
				"instance_id":  "", // Empty instance_id
			},
		},
	}

	// Should not call PurchaseUnlock since instance_id is missing or empty
	publisher.Send(ctx, logger, nil, userID, events)

	// No expectations on mock since missing instance_id should be ignored
	mockUnlockables.AssertExpectations(t)
}

// Test Send method with placement_success event but nil Unlockables system
func TestUnlockableRewardedVideoPublisher_Send_NilUnlockables(t *testing.T) {
	publisher := &pamlogix.UnlockableRewardedVideoPublisher{
		Unlockables: nil, // No unlockables system
	}

	ctx := context.Background()
	logger := &testLoggerPublisher{}
	userID := "test_user"

	events := []*pamlogix.PublisherEvent{
		{
			Name: "placement_success",
			Metadata: map[string]string{
				"placement_id": "unlockable_rewarded_video",
				"instance_id":  "test_instance",
			},
		},
	}

	// Should not panic even with nil Unlockables system
	publisher.Send(ctx, logger, nil, userID, events)
}

// Test Send method with valid placement_success event - successful unlock
func TestUnlockableRewardedVideoPublisher_Send_SuccessfulUnlock(t *testing.T) {
	mockUnlockables := &mockUnlockablesSystem{}
	publisher := &pamlogix.UnlockableRewardedVideoPublisher{
		Unlockables: mockUnlockables,
	}

	ctx := context.Background()
	logger := &testLoggerPublisher{}
	userID := "test_user"
	instanceID := "test_instance_123"

	// Mock successful unlock
	expectedUnlockables := &pamlogix.UnlockablesList{
		Slots:       5,
		ActiveSlots: 2,
		Unlockables: []*pamlogix.Unlockable{
			{
				InstanceId: instanceID,
				CanClaim:   true, // Should be claimable after instant unlock
			},
		},
	}

	mockUnlockables.On("PurchaseUnlock", ctx, logger, mock.Anything, userID, instanceID).Return(expectedUnlockables, nil)

	events := []*pamlogix.PublisherEvent{
		{
			Name: "placement_success",
			Metadata: map[string]string{
				"placement_id": "unlockable_rewarded_video",
				"instance_id":  instanceID,
			},
		},
	}

	// Should call PurchaseUnlock with correct parameters
	publisher.Send(ctx, logger, nil, userID, events)

	// Verify the mock was called correctly
	mockUnlockables.AssertExpectations(t)
}

// Test Send method with valid placement_success event - unlock fails
func TestUnlockableRewardedVideoPublisher_Send_UnlockFails(t *testing.T) {
	mockUnlockables := &mockUnlockablesSystem{}
	publisher := &pamlogix.UnlockableRewardedVideoPublisher{
		Unlockables: mockUnlockables,
	}

	ctx := context.Background()
	logger := &testLoggerPublisher{}
	userID := "test_user"
	instanceID := "test_instance_456"

	// Mock failed unlock (e.g., unlockable not found)
	mockUnlockables.On("PurchaseUnlock", ctx, logger, mock.Anything, userID, instanceID).Return((*pamlogix.UnlockablesList)(nil), assert.AnError)

	events := []*pamlogix.PublisherEvent{
		{
			Name: "placement_success",
			Metadata: map[string]string{
				"placement_id": "unlockable_rewarded_video",
				"instance_id":  instanceID,
			},
		},
	}

	// Should call PurchaseUnlock and handle the error gracefully (no panic)
	publisher.Send(ctx, logger, nil, userID, events)

	// Verify the mock was called correctly
	mockUnlockables.AssertExpectations(t)
}

// Test Send method with multiple events, some valid and some invalid
func TestUnlockableRewardedVideoPublisher_Send_MixedEvents(t *testing.T) {
	mockUnlockables := &mockUnlockablesSystem{}
	publisher := &pamlogix.UnlockableRewardedVideoPublisher{
		Unlockables: mockUnlockables,
	}

	ctx := context.Background()
	logger := &testLoggerPublisher{}
	userID := "test_user"
	instanceID1 := "valid_instance_1"
	instanceID2 := "valid_instance_2"

	// Mock successful unlocks for valid events
	expectedUnlockables1 := &pamlogix.UnlockablesList{
		Unlockables: []*pamlogix.Unlockable{
			{InstanceId: instanceID1, CanClaim: true},
		},
	}
	expectedUnlockables2 := &pamlogix.UnlockablesList{
		Unlockables: []*pamlogix.Unlockable{
			{InstanceId: instanceID2, CanClaim: true},
		},
	}

	mockUnlockables.On("PurchaseUnlock", ctx, logger, mock.Anything, userID, instanceID1).Return(expectedUnlockables1, nil)
	mockUnlockables.On("PurchaseUnlock", ctx, logger, mock.Anything, userID, instanceID2).Return(expectedUnlockables2, nil)

	events := []*pamlogix.PublisherEvent{
		// Invalid event - wrong name
		{
			Name: "user_login",
			Metadata: map[string]string{
				"placement_id": "unlockable_rewarded_video",
				"instance_id":  "should_be_ignored",
			},
		},
		// Valid event 1
		{
			Name: "placement_success",
			Metadata: map[string]string{
				"placement_id": "unlockable_rewarded_video",
				"instance_id":  instanceID1,
			},
		},
		// Invalid event - wrong placement_id
		{
			Name: "placement_success",
			Metadata: map[string]string{
				"placement_id": "banner_ad",
				"instance_id":  "should_be_ignored",
			},
		},
		// Valid event 2
		{
			Name: "placement_success",
			Metadata: map[string]string{
				"placement_id": "unlockable_rewarded_video",
				"instance_id":  instanceID2,
			},
		},
		// Invalid event - missing instance_id
		{
			Name: "placement_success",
			Metadata: map[string]string{
				"placement_id": "unlockable_rewarded_video",
			},
		},
	}

	// Should only call PurchaseUnlock for the two valid events
	publisher.Send(ctx, logger, nil, userID, events)

	// Verify the mock was called correctly
	mockUnlockables.AssertExpectations(t)
}

// Test Send method with nil metadata
func TestUnlockableRewardedVideoPublisher_Send_NilMetadata(t *testing.T) {
	mockUnlockables := &mockUnlockablesSystem{}
	publisher := &pamlogix.UnlockableRewardedVideoPublisher{
		Unlockables: mockUnlockables,
	}

	ctx := context.Background()
	logger := &testLoggerPublisher{}
	userID := "test_user"

	events := []*pamlogix.PublisherEvent{
		{
			Name:     "placement_success",
			Metadata: nil, // Nil metadata
		},
	}

	// Should not call PurchaseUnlock since metadata is nil
	publisher.Send(ctx, logger, nil, userID, events)

	// No expectations on mock since nil metadata should be handled gracefully
	mockUnlockables.AssertExpectations(t)
}

// Test Send method with empty events slice
func TestUnlockableRewardedVideoPublisher_Send_EmptyEvents(t *testing.T) {
	mockUnlockables := &mockUnlockablesSystem{}
	publisher := &pamlogix.UnlockableRewardedVideoPublisher{
		Unlockables: mockUnlockables,
	}

	ctx := context.Background()
	logger := &testLoggerPublisher{}
	userID := "test_user"

	events := []*pamlogix.PublisherEvent{} // Empty slice

	// Should not call any methods on the unlockables system
	publisher.Send(ctx, logger, nil, userID, events)

	// No expectations on mock since empty events should be handled gracefully
	mockUnlockables.AssertExpectations(t)
}

// Test Send method with nil events slice
func TestUnlockableRewardedVideoPublisher_Send_NilEvents(t *testing.T) {
	mockUnlockables := &mockUnlockablesSystem{}
	publisher := &pamlogix.UnlockableRewardedVideoPublisher{
		Unlockables: mockUnlockables,
	}

	ctx := context.Background()
	logger := &testLoggerPublisher{}
	userID := "test_user"

	var events []*pamlogix.PublisherEvent // Nil slice

	// Should not panic with nil events
	publisher.Send(ctx, logger, nil, userID, events)

	// No expectations on mock since nil events should be handled gracefully
	mockUnlockables.AssertExpectations(t)
}

// Integration test: Test the complete flow with real unlockables system
func TestUnlockableRewardedVideoPublisher_IntegrationTest(t *testing.T) {
	// Create a real unlockables system with test configuration
	config := &pamlogix.UnlockablesConfig{
		ActiveSlots:    2,
		MaxActiveSlots: 5,
		Slots:          5,
		Unlockables: map[string]*pamlogix.UnlockablesConfigUnlockable{
			"test_chest": {
				Probability: 10,
				Category:    "chest",
				Name:        "Test Chest",
				Description: "A test chest for rewarded video unlock",
				WaitTimeSec: 3600, // 1 hour
				// No cost for instant unlock in this test - rewarded video should unlock for free
			},
		},
		MaxQueuedUnlocks: 3,
	}

	unlockablesSystem := pamlogix.NewUnlockablesSystem(config)
	publisher := &pamlogix.UnlockableRewardedVideoPublisher{
		Unlockables: unlockablesSystem,
	}

	ctx := context.Background()
	logger := &testLoggerPublisher{}
	nk := pamlogix.NewTestUnlockablesNakama(t)
	userID := "integration_test_user"

	// Step 1: Create an unlockable
	unlockables, err := unlockablesSystem.Create(ctx, logger, nk, userID, "test_chest", nil)
	require.NoError(t, err)
	require.NotNil(t, unlockables)
	require.Len(t, unlockables.Unlockables, 1)

	instanceID := unlockables.Unlockables[0].InstanceId
	assert.NotEmpty(t, instanceID)
	assert.False(t, unlockables.Unlockables[0].CanClaim)

	// Step 2: Start the unlock process
	unlockables, err = unlockablesSystem.UnlockStart(ctx, logger, nk, userID, instanceID)
	require.NoError(t, err)
	assert.Greater(t, unlockables.Unlockables[0].UnlockStartTimeSec, int64(0))
	assert.False(t, unlockables.Unlockables[0].CanClaim)

	// Step 3: Simulate rewarded video completion via publisher event
	events := []*pamlogix.PublisherEvent{
		{
			Name: "placement_success",
			Metadata: map[string]string{
				"placement_id": "unlockable_rewarded_video",
				"instance_id":  instanceID,
			},
		},
	}

	// This should instantly unlock the unlockable
	publisher.Send(ctx, logger, nk, userID, events)

	// Step 4: Verify the unlockable was instantly unlocked
	unlockables, err = unlockablesSystem.Get(ctx, logger, nk, userID)
	require.NoError(t, err)
	require.Len(t, unlockables.Unlockables, 1)

	// The unlockable should now be claimable
	assert.True(t, unlockables.Unlockables[0].CanClaim, "Unlockable should be claimable after rewarded video")
	assert.Equal(t, instanceID, unlockables.Unlockables[0].InstanceId)
}

// Test edge case: Multiple placement_success events for the same instance
func TestUnlockableRewardedVideoPublisher_Send_DuplicateEvents(t *testing.T) {
	mockUnlockables := &mockUnlockablesSystem{}
	publisher := &pamlogix.UnlockableRewardedVideoPublisher{
		Unlockables: mockUnlockables,
	}

	ctx := context.Background()
	logger := &testLoggerPublisher{}
	userID := "test_user"
	instanceID := "duplicate_instance"

	// Mock the first call to succeed
	expectedUnlockables := &pamlogix.UnlockablesList{
		Unlockables: []*pamlogix.Unlockable{
			{InstanceId: instanceID, CanClaim: true},
		},
	}

	// The second call might fail (e.g., already unlocked)
	mockUnlockables.On("PurchaseUnlock", ctx, logger, mock.Anything, userID, instanceID).Return(expectedUnlockables, nil).Once()
	mockUnlockables.On("PurchaseUnlock", ctx, logger, mock.Anything, userID, instanceID).Return((*pamlogix.UnlockablesList)(nil), assert.AnError).Once()

	events := []*pamlogix.PublisherEvent{
		{
			Name: "placement_success",
			Metadata: map[string]string{
				"placement_id": "unlockable_rewarded_video",
				"instance_id":  instanceID,
			},
		},
		{
			Name: "placement_success",
			Metadata: map[string]string{
				"placement_id": "unlockable_rewarded_video",
				"instance_id":  instanceID, // Same instance ID
			},
		},
	}

	// Should call PurchaseUnlock twice, handling both success and failure
	publisher.Send(ctx, logger, nil, userID, events)

	// Verify the mock was called correctly
	mockUnlockables.AssertExpectations(t)
}

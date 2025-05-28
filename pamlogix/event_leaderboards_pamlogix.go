package pamlogix

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/heroiclabs/nakama-common/runtime"
)

const (
	eventLeaderboardsStorageCollection = "event_leaderboards"
	eventLeaderboardUserStateKey       = "user_state"
	eventLeaderboardCohortPrefix       = "cohort_"
	eventLeaderboardBackingPrefix      = "backing_"
)

// NakamaEventLeaderboardsSystem implements the EventLeaderboardsSystem interface using Nakama as the backend.
type NakamaEventLeaderboardsSystem struct {
	config                            *EventLeaderboardsConfig
	onEventLeaderboardsReward         OnReward[*EventLeaderboardsConfigLeaderboard]
	onEventLeaderboardCohortSelection OnEventLeaderboardCohortSelection
	pamlogix                          Pamlogix
}

// NewNakamaEventLeaderboardsSystem creates a new instance of the event leaderboards system with the given configuration.
func NewNakamaEventLeaderboardsSystem(config *EventLeaderboardsConfig) *NakamaEventLeaderboardsSystem {
	return &NakamaEventLeaderboardsSystem{
		config: config,
	}
}

// SetPamlogix sets the Pamlogix instance for this event leaderboards system
func (e *NakamaEventLeaderboardsSystem) SetPamlogix(pl Pamlogix) {
	e.pamlogix = pl
}

// GetType returns the system type for the event leaderboards system.
func (e *NakamaEventLeaderboardsSystem) GetType() SystemType {
	return SystemTypeEventLeaderboards
}

// GetConfig returns the configuration for the event leaderboards system.
func (e *NakamaEventLeaderboardsSystem) GetConfig() any {
	return e.config
}

// EventLeaderboardUserState represents the user's state for event leaderboards
type EventLeaderboardUserState struct {
	EventLeaderboards map[string]*EventLeaderboardUserEventState `json:"event_leaderboards,omitempty"`
}

// EventLeaderboardUserEventState represents the user's state for a specific event leaderboard
type EventLeaderboardUserEventState struct {
	CohortID         string                 `json:"cohort_id,omitempty"`
	Tier             int32                  `json:"tier,omitempty"`
	ClaimTimeSec     int64                  `json:"claim_time_sec,omitempty"`
	LastResetTimeSec int64                  `json:"last_reset_time_sec,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`

	// Reroll tracking
	RerollCount    int32 `json:"reroll_count,omitempty"`
	LastRerollTime int64 `json:"last_reroll_time,omitempty"`

	// Target score tracking
	HasReachedTarget bool  `json:"has_reached_target,omitempty"`
	WinTimeSec       int64 `json:"win_time_sec,omitempty"`

	// Participation tracking
	TotalParticipation int32 `json:"total_participation,omitempty"`
}

// EventLeaderboardCohortState represents the state of a cohort
type EventLeaderboardCohortState struct {
	ID                   string                 `json:"id,omitempty"`
	EventLeaderboardID   string                 `json:"event_leaderboard_id,omitempty"`
	Tier                 int32                  `json:"tier,omitempty"`
	CreateTimeSec        int64                  `json:"create_time_sec,omitempty"`
	StartTimeSec         int64                  `json:"start_time_sec,omitempty"`
	EndTimeSec           int64                  `json:"end_time_sec,omitempty"`
	UserIDs              []string               `json:"user_ids,omitempty"`
	MatchmakerProperties map[string]interface{} `json:"matchmaker_properties,omitempty"`
	MaxSize              int                    `json:"max_size,omitempty"`
}

// ListEventLeaderboard returns available event leaderboards for the user.
func (e *NakamaEventLeaderboardsSystem) ListEventLeaderboard(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, withScores bool, categories []string) ([]*EventLeaderboard, error) {
	if e.config == nil || len(e.config.EventLeaderboards) == 0 {
		return []*EventLeaderboard{}, nil
	}

	now := time.Now().Unix()
	var eventLeaderboards []*EventLeaderboard

	// Get user state
	userState, err := e.getUserState(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user state: %v", err)
		return nil, ErrInternal
	}

	for eventID, config := range e.config.EventLeaderboards {
		// Filter by categories if specified
		if len(categories) > 0 {
			found := false
			for _, category := range categories {
				if config.Category == category {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Check if event is within time bounds
		if !e.isEventActive(config, now) && !e.isEventClaimable(config, now) {
			continue
		}

		eventLeaderboard, err := e.buildEventLeaderboard(ctx, logger, nk, userID, eventID, config, userState, withScores, now)
		if err != nil {
			logger.Error("Failed to build event leaderboard %s: %v", eventID, err)
			continue
		}

		if eventLeaderboard != nil {
			eventLeaderboards = append(eventLeaderboards, eventLeaderboard)
		}
	}

	return eventLeaderboards, nil
}

// GetEventLeaderboard returns a specified event leaderboard's cohort for the user.
func (e *NakamaEventLeaderboardsSystem) GetEventLeaderboard(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, eventLeaderboardID string) (*EventLeaderboard, error) {
	config, exists := e.config.EventLeaderboards[eventLeaderboardID]
	if !exists {
		return nil, ErrBadInput
	}

	now := time.Now().Unix()

	// Get user state
	userState, err := e.getUserState(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user state: %v", err)
		return nil, ErrInternal
	}

	eventLeaderboard, err := e.buildEventLeaderboard(ctx, logger, nk, userID, eventLeaderboardID, config, userState, true, now)
	if err != nil {
		logger.Error("Failed to build event leaderboard %s: %v", eventLeaderboardID, err)
		return nil, ErrInternal
	}

	return eventLeaderboard, nil
}

// RollEventLeaderboard places the user into a new cohort for the specified event leaderboard if possible.
func (e *NakamaEventLeaderboardsSystem) RollEventLeaderboard(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, eventLeaderboardID string, tier *int, matchmakerProperties map[string]interface{}) (*EventLeaderboard, error) {
	config, exists := e.config.EventLeaderboards[eventLeaderboardID]
	if !exists {
		return nil, ErrBadInput
	}

	now := time.Now().Unix()

	// Check if event is active
	if !e.isEventActive(config, now) {
		return nil, ErrBadInput
	}

	// Get user state
	userState, err := e.getUserState(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user state: %v", err)
		return nil, ErrInternal
	}

	// Get or create user event state
	if userState.EventLeaderboards == nil {
		userState.EventLeaderboards = make(map[string]*EventLeaderboardUserEventState)
	}
	if userState.EventLeaderboards[eventLeaderboardID] == nil {
		userState.EventLeaderboards[eventLeaderboardID] = &EventLeaderboardUserEventState{}
	}
	userEventState := userState.EventLeaderboards[eventLeaderboardID]

	// Check reroll limits
	if config.MaxRerolls > 0 && userEventState.RerollCount >= int32(config.MaxRerolls) {
		return nil, ErrBadInput
	}

	// Check if user already has an active cohort and this is a reroll
	isReroll := userEventState.CohortID != ""

	// Handle reroll cost
	if isReroll && config.RerollCost != nil {
		// Check affordability
		canAfford, err := e.checkUserCanAffordReward(ctx, logger, nk, userID, config.RerollCost)
		if err != nil {
			logger.Error("Failed to check reroll cost affordability: %v", err)
			return nil, ErrInternal
		}
		if !canAfford {
			return nil, ErrBadInput
		}

		// Deduct cost
		err = e.deductRewardCost(ctx, logger, nk, userID, config.RerollCost)
		if err != nil {
			logger.Error("Failed to deduct reroll cost: %v", err)
			return nil, ErrInternal
		}
	}

	// Handle participation cost for first time joining
	if !isReroll && config.ParticipationCost != nil {
		// Check affordability
		canAfford, err := e.checkUserCanAffordReward(ctx, logger, nk, userID, config.ParticipationCost)
		if err != nil {
			logger.Error("Failed to check participation cost affordability: %v", err)
			return nil, ErrInternal
		}
		if !canAfford {
			return nil, ErrBadInput
		}

		// Deduct cost
		err = e.deductRewardCost(ctx, logger, nk, userID, config.ParticipationCost)
		if err != nil {
			logger.Error("Failed to deduct participation cost: %v", err)
			return nil, ErrInternal
		}
	}

	// Determine user tier
	userTier := int32(0)
	if tier != nil {
		userTier = int32(*tier)
	} else if userEventState.Tier > 0 {
		userTier = userEventState.Tier
	}

	// Ensure tier is within bounds
	if userTier < 0 {
		userTier = 0
	}
	if config.Tiers > 0 && userTier >= int32(config.Tiers) {
		userTier = int32(config.Tiers - 1)
	}

	// Find or create a cohort
	cohortID, err := e.findOrCreateCohort(ctx, logger, nk, eventLeaderboardID, config, userID, userTier, matchmakerProperties)
	if err != nil {
		logger.Error("Failed to find or create cohort: %v", err)
		return nil, ErrInternal
	}

	// Update user state
	userEventState.CohortID = cohortID
	userEventState.Tier = userTier
	userEventState.ClaimTimeSec = 0         // Reset claim time
	userEventState.HasReachedTarget = false // Reset target achievement
	userEventState.WinTimeSec = 0           // Reset win time

	// Update reroll tracking
	if isReroll {
		userEventState.RerollCount++
		userEventState.LastRerollTime = now
	} else {
		// First time joining this event, increment participation
		userEventState.TotalParticipation++
	}

	// Save user state
	if err := e.saveUserState(ctx, logger, nk, userID, userState); err != nil {
		logger.Error("Failed to save user state: %v", err)
		return nil, ErrInternal
	}

	// Return the updated event leaderboard
	return e.buildEventLeaderboard(ctx, logger, nk, userID, eventLeaderboardID, config, userState, true, now)
}

// UpdateEventLeaderboard updates the user's score in the specified event leaderboard, and returns the user's updated cohort information.
func (e *NakamaEventLeaderboardsSystem) UpdateEventLeaderboard(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, username, eventLeaderboardID string, score, subscore int64, metadata map[string]interface{}) (*EventLeaderboard, error) {
	config, exists := e.config.EventLeaderboards[eventLeaderboardID]
	if !exists {
		return nil, ErrBadInput
	}

	now := time.Now().Unix()

	// Check if event is active
	if !e.isEventActive(config, now) {
		return nil, ErrBadInput
	}

	// Get user state
	userState, err := e.getUserState(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user state: %v", err)
		return nil, ErrInternal
	}

	// Check if user has a cohort
	userEventState, exists := userState.EventLeaderboards[eventLeaderboardID]
	if !exists || userEventState.CohortID == "" {
		return nil, ErrBadInput
	}

	// Get backing leaderboard ID
	backingID := e.getBackingLeaderboardID(eventLeaderboardID, userEventState.CohortID)

	// Submit score to backing leaderboard
	_, err = nk.LeaderboardRecordWrite(ctx, backingID, userID, username, score, subscore, metadata, nil)
	if err != nil {
		logger.Error("Failed to write leaderboard record: %v", err)
		return nil, ErrInternal
	}

	// Check for target score achievement
	if config.TargetScore > 0 && !userEventState.HasReachedTarget {
		if score >= config.TargetScore {
			userEventState.HasReachedTarget = true
			userEventState.WinTimeSec = now

			// Check if this user is among the winners
			if config.WinnerCount > 0 {
				// Get current winners count for this cohort
				winnersCount, err := e.getWinnersCount(ctx, logger, nk, eventLeaderboardID, userEventState.CohortID, config.TargetScore)
				if err != nil {
					logger.Error("Failed to get winners count: %v", err)
				} else if winnersCount <= config.WinnerCount {
					// User is a winner, check if we should trigger reroll for the cohort
					if winnersCount == config.WinnerCount {
						// Maximum winners reached, trigger reroll for all users in this cohort
						err = e.triggerCohortReroll(ctx, logger, nk, eventLeaderboardID, config, userEventState.CohortID)
						if err != nil {
							logger.Error("Failed to trigger cohort reroll: %v", err)
						}
					}
				}
			}

			// Save user state with target achievement
			if err := e.saveUserState(ctx, logger, nk, userID, userState); err != nil {
				logger.Error("Failed to save user state after target achievement: %v", err)
				return nil, ErrInternal
			}
		}
	}

	// Return the updated event leaderboard
	return e.buildEventLeaderboard(ctx, logger, nk, userID, eventLeaderboardID, config, userState, true, now)
}

// ClaimEventLeaderboard claims the user's reward for the given event leaderboard.
func (e *NakamaEventLeaderboardsSystem) ClaimEventLeaderboard(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, eventLeaderboardID string) (*EventLeaderboard, error) {
	config, exists := e.config.EventLeaderboards[eventLeaderboardID]
	if !exists {
		return nil, ErrBadInput
	}

	now := time.Now().Unix()

	// Get user state
	userState, err := e.getUserState(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user state: %v", err)
		return nil, ErrInternal
	}

	// Check if user has a cohort and can claim
	userEventState, exists := userState.EventLeaderboards[eventLeaderboardID]
	if !exists || userEventState.CohortID == "" {
		return nil, ErrBadInput
	}

	// Check if already claimed
	if userEventState.ClaimTimeSec > 0 {
		return nil, ErrBadInput
	}

	// Check if event is claimable
	if !e.isEventClaimable(config, now) {
		return nil, ErrBadInput
	}

	// Get user's rank and calculate reward
	backingID := e.getBackingLeaderboardID(eventLeaderboardID, userEventState.CohortID)
	records, _, _, _, err := nk.LeaderboardRecordsList(ctx, backingID, []string{userID}, 1, "", 0)
	if err != nil {
		logger.Error("Failed to get leaderboard records: %v", err)
		return nil, ErrInternal
	}

	if len(records) == 0 {
		return nil, ErrBadInput
	}

	userRecord := records[0]
	userRank := userRecord.Rank

	// Find applicable reward tier
	rewardTier := e.findRewardTier(config, userEventState.Tier, int32(userRank))
	if rewardTier == nil {
		// No reward for this rank
		userEventState.ClaimTimeSec = now
		if err := e.saveUserState(ctx, logger, nk, userID, userState); err != nil {
			logger.Error("Failed to save user state: %v", err)
			return nil, ErrInternal
		}
		return e.buildEventLeaderboard(ctx, logger, nk, userID, eventLeaderboardID, config, userState, true, now)
	}

	// Process reward
	var reward *Reward
	if rewardTier.Reward != nil {
		economySystem := e.pamlogix.GetEconomySystem()
		if economySystem != nil {
			reward, err = economySystem.RewardRoll(ctx, logger, nk, userID, rewardTier.Reward)
			if err != nil {
				logger.Error("Failed to roll reward: %v", err)
				return nil, ErrInternal
			}

			// Apply custom reward function if set
			if e.onEventLeaderboardsReward != nil {
				reward, err = e.onEventLeaderboardsReward(ctx, logger, nk, userID, eventLeaderboardID, config, rewardTier.Reward, reward)
				if err != nil {
					logger.Error("Failed to apply custom reward: %v", err)
					return nil, ErrInternal
				}
			}

			// Grant the reward
			if reward != nil {
				_, _, _, err = economySystem.RewardGrant(ctx, logger, nk, userID, reward, nil, true)
				if err != nil {
					logger.Error("Failed to grant reward: %v", err)
					return nil, ErrInternal
				}
			}
		}
	}

	// Update tier if specified
	if rewardTier.TierChange != 0 {
		newTier := userEventState.Tier + int32(rewardTier.TierChange)
		if newTier < 0 {
			newTier = 0
		}
		if config.Tiers > 0 && newTier >= int32(config.Tiers) {
			newTier = int32(config.Tiers - 1)
		}
		userEventState.Tier = newTier
	}

	// Mark as claimed
	userEventState.ClaimTimeSec = now

	// Save user state
	if err := e.saveUserState(ctx, logger, nk, userID, userState); err != nil {
		logger.Error("Failed to save user state: %v", err)
		return nil, ErrInternal
	}

	// Return the updated event leaderboard
	return e.buildEventLeaderboard(ctx, logger, nk, userID, eventLeaderboardID, config, userState, true, now)
}

// SetOnEventLeaderboardsReward sets a custom reward function which will run after an event leaderboard's reward is rolled.
func (e *NakamaEventLeaderboardsSystem) SetOnEventLeaderboardsReward(fn OnReward[*EventLeaderboardsConfigLeaderboard]) {
	e.onEventLeaderboardsReward = fn
}

// SetOnEventLeaderboardCohortSelection sets a custom function that can replace the cohort or opponent selection feature of event leaderboards.
func (e *NakamaEventLeaderboardsSystem) SetOnEventLeaderboardCohortSelection(fn OnEventLeaderboardCohortSelection) {
	e.onEventLeaderboardCohortSelection = fn
}

// DebugFill fills the user's current cohort with dummy users for all remaining available slots.
func (e *NakamaEventLeaderboardsSystem) DebugFill(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, eventLeaderboardID string, targetCount int) (*EventLeaderboard, error) {
	config, exists := e.config.EventLeaderboards[eventLeaderboardID]
	if !exists {
		return nil, ErrBadInput
	}

	// Get user state
	userState, err := e.getUserState(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user state: %v", err)
		return nil, ErrInternal
	}

	// Check if user has a cohort
	userEventState, exists := userState.EventLeaderboards[eventLeaderboardID]
	if !exists || userEventState.CohortID == "" {
		return nil, ErrBadInput
	}

	// Get backing leaderboard ID
	backingID := e.getBackingLeaderboardID(eventLeaderboardID, userEventState.CohortID)

	// Get current records
	records, _, _, _, err := nk.LeaderboardRecordsList(ctx, backingID, nil, 100, "", 0)
	if err != nil {
		logger.Error("Failed to get leaderboard records: %v", err)
		return nil, ErrInternal
	}

	currentCount := len(records)
	if currentCount >= targetCount {
		// Already at or above target count
		return e.buildEventLeaderboard(ctx, logger, nk, userID, eventLeaderboardID, config, userState, true, time.Now().Unix())
	}

	// Add dummy users
	for i := currentCount; i < targetCount; i++ {
		dummyUserID := fmt.Sprintf("dummy_%s_%d", userEventState.CohortID, i)
		dummyUsername := fmt.Sprintf("Bot%d", i+1)

		_, err = nk.LeaderboardRecordWrite(ctx, backingID, dummyUserID, dummyUsername, 0, 0, map[string]interface{}{}, nil)
		if err != nil {
			logger.Error("Failed to write dummy leaderboard record: %v", err)
			continue
		}
	}

	// Return the updated event leaderboard
	return e.buildEventLeaderboard(ctx, logger, nk, userID, eventLeaderboardID, config, userState, true, time.Now().Unix())
}

// DebugRandomScores assigns random scores to the participants of the user's current cohort, except to the user themselves.
func (e *NakamaEventLeaderboardsSystem) DebugRandomScores(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, eventLeaderboardID string, scoreMin, scoreMax, subscoreMin, subscoreMax int64, operator *int) (*EventLeaderboard, error) {
	config, exists := e.config.EventLeaderboards[eventLeaderboardID]
	if !exists {
		return nil, ErrBadInput
	}

	// Get user state
	userState, err := e.getUserState(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user state: %v", err)
		return nil, ErrInternal
	}

	// Check if user has a cohort
	userEventState, exists := userState.EventLeaderboards[eventLeaderboardID]
	if !exists || userEventState.CohortID == "" {
		return nil, ErrBadInput
	}

	// Get backing leaderboard ID
	backingID := e.getBackingLeaderboardID(eventLeaderboardID, userEventState.CohortID)

	// Get current records
	records, _, _, _, err := nk.LeaderboardRecordsList(ctx, backingID, nil, 100, "", 0)
	if err != nil {
		logger.Error("Failed to get leaderboard records: %v", err)
		return nil, ErrInternal
	}

	// Assign random scores to all users except the requesting user
	for _, record := range records {
		if record.OwnerId == userID {
			continue // Skip the requesting user
		}

		// Generate random score and subscore
		score := scoreMin + rand.Int63n(scoreMax-scoreMin+1)
		subscore := subscoreMin + rand.Int63n(subscoreMax-subscoreMin+1)

		// Get username from record
		username := record.OwnerId // Default to user ID
		if record.Username != nil {
			username = record.Username.Value
		}

		_, err = nk.LeaderboardRecordWrite(ctx, backingID, record.OwnerId, username, score, subscore, map[string]interface{}{}, nil)
		if err != nil {
			logger.Error("Failed to write random leaderboard record: %v", err)
			continue
		}
	}

	// Return the updated event leaderboard
	return e.buildEventLeaderboard(ctx, logger, nk, userID, eventLeaderboardID, config, userState, true, time.Now().Unix())
}

// ProcessEventEnd handles end-of-event logic including tier changes based on change zones
func (e *NakamaEventLeaderboardsSystem) ProcessEventEnd(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, eventLeaderboardID string) error {
	config, exists := e.config.EventLeaderboards[eventLeaderboardID]
	if !exists {
		return ErrBadInput
	}

	now := time.Now().Unix()

	// Check if event has actually ended
	if config.EndTimeSec == 0 || now < config.EndTimeSec {
		return ErrBadInput
	}

	// Get all cohorts for this event
	cohorts, err := e.getAllCohortsForEvent(ctx, logger, nk, eventLeaderboardID)
	if err != nil {
		logger.Error("Failed to get cohorts for event %s: %v", eventLeaderboardID, err)
		return err
	}

	// Process each cohort
	for _, cohort := range cohorts {
		err := e.processCohortTierChanges(ctx, logger, nk, eventLeaderboardID, config, cohort)
		if err != nil {
			logger.Error("Failed to process tier changes for cohort %s: %v", cohort.ID, err)
			// Continue processing other cohorts
		}
	}

	return nil
}

// Helper function to get all cohorts for an event
func (e *NakamaEventLeaderboardsSystem) getAllCohortsForEvent(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, eventLeaderboardID string) ([]*EventLeaderboardCohortState, error) {
	// List all cohort storage objects for this event
	objects, _, err := nk.StorageList(ctx, "", "", eventLeaderboardsStorageCollection, 100, "")
	if err != nil {
		return nil, err
	}

	var cohorts []*EventLeaderboardCohortState
	for _, obj := range objects {
		if strings.HasPrefix(obj.Key, eventLeaderboardCohortPrefix) {
			var cohort EventLeaderboardCohortState
			if err := json.Unmarshal([]byte(obj.Value), &cohort); err != nil {
				logger.Error("Failed to unmarshal cohort state: %v", err)
				continue
			}

			// Only include cohorts for this event
			if cohort.EventLeaderboardID == eventLeaderboardID {
				cohorts = append(cohorts, &cohort)
			}
		}
	}

	return cohorts, nil
}

// Helper function to process tier changes for a cohort based on change zones
func (e *NakamaEventLeaderboardsSystem) processCohortTierChanges(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, eventLeaderboardID string, config *EventLeaderboardsConfigLeaderboard, cohort *EventLeaderboardCohortState) error {
	// Get change zone for this tier
	tierStr := strconv.Itoa(int(cohort.Tier))
	changeZone, exists := config.ChangeZones[tierStr]
	if !exists {
		// No change zone configured for this tier
		return nil
	}

	// Get leaderboard records for this cohort
	backingID := e.getBackingLeaderboardID(eventLeaderboardID, cohort.ID)
	records, _, _, _, err := nk.LeaderboardRecordsList(ctx, backingID, nil, 100, "", 0)
	if err != nil {
		logger.Error("Failed to get leaderboard records for cohort %s: %v", cohort.ID, err)
		return err
	}

	if len(records) == 0 {
		return nil
	}

	// Calculate promotion and demotion thresholds
	totalParticipants := len(records)
	promotionCount := int(float64(totalParticipants) * changeZone.Promotion)
	demotionCount := int(float64(totalParticipants) * changeZone.Demotion)

	// Process promotions (top performers)
	for i := 0; i < promotionCount && i < len(records); i++ {
		record := records[i]
		err := e.applyTierChange(ctx, logger, nk, record.OwnerId, eventLeaderboardID, 1) // +1 tier
		if err != nil {
			logger.Error("Failed to promote user %s: %v", record.OwnerId, err)
		}
	}

	// Process demotions (bottom performers)
	demotionStartIndex := len(records) - demotionCount
	for i := demotionStartIndex; i < len(records); i++ {
		record := records[i]

		// Check if user should be demoted for being idle
		shouldDemoteIdle := changeZone.DemoteIdle && record.Score == 0

		if shouldDemoteIdle || i >= demotionStartIndex {
			err := e.applyTierChange(ctx, logger, nk, record.OwnerId, eventLeaderboardID, -1) // -1 tier
			if err != nil {
				logger.Error("Failed to demote user %s: %v", record.OwnerId, err)
			}
		}
	}

	// Handle idle demotion for users who didn't submit any scores
	if changeZone.DemoteIdle {
		for _, userID := range cohort.UserIDs {
			// Check if user has a record
			hasRecord := false
			for _, record := range records {
				if record.OwnerId == userID {
					hasRecord = true
					break
				}
			}

			if !hasRecord {
				// User didn't participate, demote them
				err := e.applyTierChange(ctx, logger, nk, userID, eventLeaderboardID, -1)
				if err != nil {
					logger.Error("Failed to demote idle user %s: %v", userID, err)
				}
			}
		}
	}

	return nil
}

// Helper function to apply tier change to a user
func (e *NakamaEventLeaderboardsSystem) applyTierChange(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, eventLeaderboardID string, tierChange int) error {
	config, exists := e.config.EventLeaderboards[eventLeaderboardID]
	if !exists {
		return ErrBadInput
	}

	// Get user state
	userState, err := e.getUserState(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user state for tier change: %v", err)
		return err
	}

	// Get or create user event state
	if userState.EventLeaderboards == nil {
		userState.EventLeaderboards = make(map[string]*EventLeaderboardUserEventState)
	}
	if userState.EventLeaderboards[eventLeaderboardID] == nil {
		userState.EventLeaderboards[eventLeaderboardID] = &EventLeaderboardUserEventState{}
	}

	userEventState := userState.EventLeaderboards[eventLeaderboardID]

	// Calculate new tier
	newTier := int32(int(userEventState.Tier) + tierChange)

	// Apply tier bounds
	if newTier < 0 {
		newTier = 0
	}
	if config.Tiers > 0 && newTier >= int32(config.Tiers) {
		newTier = int32(config.Tiers - 1)
	}

	// Apply max idle tier drop limit
	if tierChange < 0 && config.MaxIdleTierDrop > 0 {
		maxDrop := int32(config.MaxIdleTierDrop)
		if userEventState.Tier-newTier > maxDrop {
			newTier = userEventState.Tier - maxDrop
		}
	}

	// Update tier
	oldTier := userEventState.Tier
	userEventState.Tier = newTier

	// Reset cohort to force new assignment in next event
	userEventState.CohortID = ""
	userEventState.HasReachedTarget = false
	userEventState.WinTimeSec = 0

	// Save user state
	if err := e.saveUserState(ctx, logger, nk, userID, userState); err != nil {
		logger.Error("Failed to save user state after tier change: %v", err)
		return err
	}

	logger.Info("Applied tier change for user %s in event %s: %d -> %d", userID, eventLeaderboardID, oldTier, newTier)
	return nil
}

// Helper methods

func (e *NakamaEventLeaderboardsSystem) getUserState(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (*EventLeaderboardUserState, error) {
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: eventLeaderboardsStorageCollection,
			Key:        eventLeaderboardUserStateKey,
			UserID:     userID,
		},
	})
	if err != nil {
		return nil, err
	}

	userState := &EventLeaderboardUserState{
		EventLeaderboards: make(map[string]*EventLeaderboardUserEventState),
	}

	if len(objects) > 0 && objects[0].Value != "" {
		if err := json.Unmarshal([]byte(objects[0].Value), userState); err != nil {
			return nil, err
		}
	}

	return userState, nil
}

func (e *NakamaEventLeaderboardsSystem) saveUserState(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, userState *EventLeaderboardUserState) error {
	data, err := json.Marshal(userState)
	if err != nil {
		return err
	}

	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{
		{
			Collection: eventLeaderboardsStorageCollection,
			Key:        eventLeaderboardUserStateKey,
			UserID:     userID,
			Value:      string(data),
		},
	})

	return err
}

func (e *NakamaEventLeaderboardsSystem) isEventActive(config *EventLeaderboardsConfigLeaderboard, now int64) bool {
	if config.StartTimeSec > 0 && now < config.StartTimeSec {
		return false
	}
	if config.EndTimeSec > 0 && now >= config.EndTimeSec {
		return false
	}
	return true
}

func (e *NakamaEventLeaderboardsSystem) isEventClaimable(config *EventLeaderboardsConfigLeaderboard, now int64) bool {
	if config.EndTimeSec > 0 && now >= config.EndTimeSec {
		// Event has ended, check if it's still within claim window
		claimWindow := int64(86400) // 24 hours default
		if config.Duration > 0 {
			claimWindow = config.Duration
		}
		return now < config.EndTimeSec+claimWindow
	}
	return false
}

func (e *NakamaEventLeaderboardsSystem) getBackingLeaderboardID(eventLeaderboardID, cohortID string) string {
	return fmt.Sprintf("%s%s_%s", eventLeaderboardBackingPrefix, eventLeaderboardID, cohortID)
}

func (e *NakamaEventLeaderboardsSystem) findOrCreateCohort(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, eventLeaderboardID string, config *EventLeaderboardsConfigLeaderboard, userID string, tier int32, matchmakerProperties map[string]interface{}) (string, error) {
	// Use custom cohort selection if available
	if e.onEventLeaderboardCohortSelection != nil {
		cohortID, cohortUserIDs, newCohortConfig, err := e.onEventLeaderboardCohortSelection(ctx, logger, nk, eventLeaderboardsStorageCollection, eventLeaderboardID, config, userID, int(tier), matchmakerProperties)
		if err != nil {
			return "", err
		}

		if cohortID != "" {
			// Custom function provided a cohort
			if newCohortConfig != nil && newCohortConfig.ForceNewCohort {
				// Create new cohort with specified users
				return e.createCohort(ctx, logger, nk, eventLeaderboardID, config, tier, cohortUserIDs, matchmakerProperties)
			}
			return cohortID, nil
		}
	}

	// Default cohort selection logic
	// Try to find an existing cohort with available space
	cohortID := e.findAvailableCohort(ctx, logger, nk, eventLeaderboardID, tier, config.CohortSize, userID)
	if cohortID != "" {
		// Add user to the existing cohort
		err := e.addUserToCohort(ctx, logger, nk, cohortID, userID)
		if err != nil {
			logger.Error("Failed to add user to existing cohort %s: %v", cohortID, err)
			// Fall back to creating a new cohort
		} else {
			return cohortID, nil
		}
	}

	// Create a new cohort
	return e.createCohort(ctx, logger, nk, eventLeaderboardID, config, tier, []string{userID}, matchmakerProperties)
}

// addUserToCohort adds a user to an existing cohort
func (e *NakamaEventLeaderboardsSystem) addUserToCohort(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, cohortID, userID string) error {
	// Get current cohort state
	cohortState, err := e.getCohortState(ctx, logger, nk, cohortID)
	if err != nil {
		return err
	}

	// Check if user is already in the cohort
	for _, existingUserID := range cohortState.UserIDs {
		if existingUserID == userID {
			// User is already in this cohort
			return nil
		}
	}

	// Check if cohort has space
	if len(cohortState.UserIDs) >= cohortState.MaxSize {
		return fmt.Errorf("cohort %s is full", cohortID)
	}

	// Add user to cohort
	cohortState.UserIDs = append(cohortState.UserIDs, userID)

	// Save updated cohort state
	data, err := json.Marshal(cohortState)
	if err != nil {
		return err
	}

	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{
		{
			Collection: eventLeaderboardsStorageCollection,
			Key:        eventLeaderboardCohortPrefix + cohortID,
			Value:      string(data),
		},
	})
	if err != nil {
		return err
	}

	logger.Debug("Added user %s to cohort %s (new size: %d)", userID, cohortID, len(cohortState.UserIDs))
	return nil
}

func (e *NakamaEventLeaderboardsSystem) findAvailableCohort(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, eventLeaderboardID string, tier int32, maxSize int, userID string) string {
	// List all cohort storage objects
	objects, _, err := nk.StorageList(ctx, "", "", eventLeaderboardsStorageCollection, 100, "")
	if err != nil {
		logger.Error("Failed to list storage objects for cohort search: %v", err)
		return ""
	}

	// Search through existing cohorts
	for _, obj := range objects {
		// Only check cohort objects
		if !strings.HasPrefix(obj.Key, eventLeaderboardCohortPrefix) {
			continue
		}

		// Parse cohort state
		var cohortState EventLeaderboardCohortState
		if err := json.Unmarshal([]byte(obj.Value), &cohortState); err != nil {
			logger.Error("Failed to unmarshal cohort state: %v", err)
			continue
		}

		// Check if this cohort matches our criteria
		if cohortState.EventLeaderboardID != eventLeaderboardID {
			continue // Different event
		}

		if cohortState.Tier != tier {
			continue // Different tier
		}

		// Check if user is already in this cohort
		userAlreadyInCohort := false
		for _, existingUserID := range cohortState.UserIDs {
			if existingUserID == userID {
				userAlreadyInCohort = true
				break
			}
		}
		if userAlreadyInCohort {
			continue // User is already in this cohort
		}

		// Check if cohort has available space
		currentSize := len(cohortState.UserIDs)
		if currentSize >= maxSize || currentSize >= cohortState.MaxSize {
			continue // Cohort is full
		}

		// Check if cohort is still active (not ended)
		now := time.Now().Unix()
		if cohortState.EndTimeSec > 0 && now >= cohortState.EndTimeSec {
			continue // Cohort has ended
		}

		// Found an available cohort
		logger.Debug("Found available cohort %s for event %s, tier %d (current size: %d, max size: %d)",
			cohortState.ID, eventLeaderboardID, tier, currentSize, maxSize)
		return cohortState.ID
	}

	// No available cohort found
	logger.Debug("No available cohort found for event %s, tier %d", eventLeaderboardID, tier)
	return ""
}

func (e *NakamaEventLeaderboardsSystem) createCohort(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, eventLeaderboardID string, config *EventLeaderboardsConfigLeaderboard, tier int32, userIDs []string, matchmakerProperties map[string]interface{}) (string, error) {
	cohortID := uuid.New().String()
	now := time.Now().Unix()

	cohortState := &EventLeaderboardCohortState{
		ID:                   cohortID,
		EventLeaderboardID:   eventLeaderboardID,
		Tier:                 tier,
		CreateTimeSec:        now,
		StartTimeSec:         config.StartTimeSec,
		EndTimeSec:           config.EndTimeSec,
		UserIDs:              userIDs,
		MatchmakerProperties: matchmakerProperties,
		MaxSize:              config.CohortSize,
	}

	// Save cohort state
	data, err := json.Marshal(cohortState)
	if err != nil {
		return "", err
	}

	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{
		{
			Collection: eventLeaderboardsStorageCollection,
			Key:        eventLeaderboardCohortPrefix + cohortID,
			Value:      string(data),
		},
	})
	if err != nil {
		return "", err
	}

	// Create backing leaderboard
	backingID := e.getBackingLeaderboardID(eventLeaderboardID, cohortID)

	// Calculate backing leaderboard ID with config
	if config.BackingId != "" {
		config.CalculatedBackingId = config.BackingId + "_" + cohortID
	} else {
		config.CalculatedBackingId = backingID
	}

	// Convert ascending bool to sort order string
	sortOrder := "desc"
	if config.Ascending {
		sortOrder = "asc"
	}

	err = nk.LeaderboardCreate(ctx, config.CalculatedBackingId, false, sortOrder, config.Operator, config.ResetSchedule, nil, false)
	if err != nil {
		// Leaderboard might already exist, which is fine
		logger.Debug("Leaderboard creation failed (might already exist): %v", err)
	}

	return cohortID, nil
}

func (e *NakamaEventLeaderboardsSystem) buildEventLeaderboard(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, eventLeaderboardID string, config *EventLeaderboardsConfigLeaderboard, userState *EventLeaderboardUserState, withScores bool, now int64) (*EventLeaderboard, error) {
	eventLeaderboard := &EventLeaderboard{
		Id:                   eventLeaderboardID,
		Name:                 config.Name,
		Description:          config.Description,
		Category:             config.Category,
		Ascending:            config.Ascending,
		Operator:             config.Operator,
		StartTimeSec:         config.StartTimeSec,
		EndTimeSec:           config.EndTimeSec,
		AdditionalProperties: config.AdditionalProperties,
		MaxNumScore:          int64(config.MaxNumScore),
		CurrentTimeSec:       now,
		RewardTiers:          make(map[int32]*EventLeaderboardRewardTiers),
		ChangeZones:          make(map[int32]*EventLeaderboardChangeZone),
	}

	// Set state flags
	eventLeaderboard.IsActive = e.isEventActive(config, now)
	eventLeaderboard.CanClaim = false
	eventLeaderboard.CanRoll = eventLeaderboard.IsActive

	// Get user event state
	userEventState, hasUserState := userState.EventLeaderboards[eventLeaderboardID]
	if hasUserState {
		eventLeaderboard.Tier = userEventState.Tier
		eventLeaderboard.ClaimTimeSec = userEventState.ClaimTimeSec
		eventLeaderboard.CohortId = userEventState.CohortID

		// Check if can claim
		if userEventState.CohortID != "" && userEventState.ClaimTimeSec == 0 && e.isEventClaimable(config, now) {
			eventLeaderboard.CanClaim = true
			eventLeaderboard.CanRoll = false
		}

		// Enhanced reroll logic
		if eventLeaderboard.IsActive {
			// Can roll if no cohort or if rerolls are available
			if userEventState.CohortID == "" {
				eventLeaderboard.CanRoll = true
			} else {
				// Check reroll limits
				if config.MaxRerolls == 0 || userEventState.RerollCount < int32(config.MaxRerolls) {
					eventLeaderboard.CanRoll = true
				} else {
					eventLeaderboard.CanRoll = false
				}
			}
		} else {
			eventLeaderboard.CanRoll = false
		}

		// If user has claimed, can't roll
		if userEventState.ClaimTimeSec > 0 {
			eventLeaderboard.CanRoll = false
		}
	}

	// Build reward tiers
	if config.RewardTiers != nil {
		for tierStr, rewardTiers := range config.RewardTiers {
			tierInt, _ := strconv.Atoi(tierStr)
			eventLeaderboardRewardTiers := &EventLeaderboardRewardTiers{
				RewardTiers: make([]*EventLeaderboardRewardTier, 0, len(rewardTiers)),
			}

			for _, rewardTier := range rewardTiers {
				tier := &EventLeaderboardRewardTier{
					Name:       rewardTier.Name,
					RankMax:    int32(rewardTier.RankMax),
					RankMin:    int32(rewardTier.RankMin),
					TierChange: int32(rewardTier.TierChange),
				}

				// Convert reward to available rewards
				if rewardTier.Reward != nil {
					economySystem := e.pamlogix.GetEconomySystem()
					if economySystem != nil {
						availableRewards := economySystem.RewardConvert(nil)
						if availableRewards != nil {
							// This would need proper implementation to convert EconomyConfigReward to AvailableRewards
							// For now, we'll leave it nil
							tier.AvailableRewards = nil
						}
					}
				}

				eventLeaderboardRewardTiers.RewardTiers = append(eventLeaderboardRewardTiers.RewardTiers, tier)
			}

			eventLeaderboard.RewardTiers[int32(tierInt)] = eventLeaderboardRewardTiers
		}
	}

	// Build change zones
	if config.ChangeZones != nil {
		for tierStr, changeZone := range config.ChangeZones {
			tierInt, _ := strconv.Atoi(tierStr)
			eventLeaderboard.ChangeZones[int32(tierInt)] = &EventLeaderboardChangeZone{
				Promotion:  changeZone.Promotion,
				Demotion:   changeZone.Demotion,
				DemoteIdle: changeZone.DemoteIdle,
			}
		}
	}

	// Load scores if requested and user has a cohort
	if withScores && hasUserState && userEventState.CohortID != "" {
		backingID := e.getBackingLeaderboardID(eventLeaderboardID, userEventState.CohortID)

		records, _, _, _, err := nk.LeaderboardRecordsList(ctx, backingID, nil, 100, "", 0)
		if err != nil {
			logger.Error("Failed to get leaderboard records: %v", err)
		} else {
			eventLeaderboard.Scores = make([]*EventLeaderboardScore, 0, len(records))
			eventLeaderboard.Count = int64(len(records))
			eventLeaderboard.MaxCount = int64(config.CohortSize)

			for _, record := range records {
				username := record.OwnerId // Default to user ID
				if record.Username != nil {
					username = record.Username.Value
				}

				// Convert timestamps to Unix seconds
				createTime := int64(0)
				if record.CreateTime != nil {
					createTime = record.CreateTime.Seconds
				}
				updateTime := int64(0)
				if record.UpdateTime != nil {
					updateTime = record.UpdateTime.Seconds
				}

				score := &EventLeaderboardScore{
					Id:            record.OwnerId,
					Username:      username,
					DisplayName:   username, // Could be enhanced with actual display names
					CreateTimeSec: createTime,
					UpdateTimeSec: updateTime,
					Rank:          record.Rank,
					Score:         record.Score,
					Subscore:      record.Subscore,
					NumScores:     int64(record.NumScore),
					Metadata:      record.Metadata,
				}
				eventLeaderboard.Scores = append(eventLeaderboard.Scores, score)
			}
		}
	}

	// Set backing ID
	if hasUserState && userEventState.CohortID != "" {
		eventLeaderboard.BackingId = e.getBackingLeaderboardID(eventLeaderboardID, userEventState.CohortID)
	}

	return eventLeaderboard, nil
}

func (e *NakamaEventLeaderboardsSystem) findRewardTier(config *EventLeaderboardsConfigLeaderboard, tier int32, rank int32) *EventLeaderboardsConfigLeaderboardRewardTier {
	tierStr := strconv.Itoa(int(tier))
	rewardTiers, exists := config.RewardTiers[tierStr]
	if !exists {
		return nil
	}

	for _, rewardTier := range rewardTiers {
		if rank >= int32(rewardTier.RankMin) && rank <= int32(rewardTier.RankMax) {
			return rewardTier
		}
	}

	return nil
}

// Helper function to check if user can afford a reward cost
func (e *NakamaEventLeaderboardsSystem) checkUserCanAffordReward(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, rewardConfig *EconomyConfigReward) (bool, error) {
	if rewardConfig == nil || rewardConfig.Guaranteed == nil {
		return true, nil
	}

	// Check currencies
	if rewardConfig.Guaranteed.Currencies != nil && len(rewardConfig.Guaranteed.Currencies) > 0 {
		economySystem := e.pamlogix.GetEconomySystem()
		if economySystem == nil {
			return false, ErrSystemNotAvailable
		}

		// Get user's account to check wallet
		account, err := nk.AccountGetId(ctx, userID)
		if err != nil {
			logger.Error("Failed to get user account: %v", err)
			return false, err
		}

		// Get wallet from account
		wallet, err := economySystem.UnmarshalWallet(account)
		if err != nil {
			logger.Error("Failed to unmarshal wallet: %v", err)
			return false, err
		}

		// Check if user has enough of each required currency
		for currencyID, currencyReward := range rewardConfig.Guaranteed.Currencies {
			// Use the maximum amount as the cost
			amount := currencyReward.Max
			if amount < 0 {
				amount = -amount // Convert negative to positive for affordability check
			}
			if wallet[currencyID] < amount {
				logger.Debug("User %s does not have enough of currency %s (has %d, needs %d)", userID, currencyID, wallet[currencyID], amount)
				return false, nil
			}
		}
	}

	// Check items
	if rewardConfig.Guaranteed.Items != nil && len(rewardConfig.Guaranteed.Items) > 0 {
		inventorySystem := e.pamlogix.GetInventorySystem()
		if inventorySystem == nil {
			return false, ErrSystemNotAvailable
		}

		// Get user's inventory
		inventory, err := inventorySystem.ListInventoryItems(ctx, logger, nk, userID, "")
		if err != nil {
			logger.Error("Failed to get user inventory: %v", err)
			return false, err
		}

		// Check if user has enough of each required item
		for itemID, itemReward := range rewardConfig.Guaranteed.Items {
			// Use the maximum amount as the cost
			amount := itemReward.Max
			if amount < 0 {
				amount = -amount // Convert negative to positive for affordability check
			}

			found := false
			for _, item := range inventory.Items {
				if item.Id == itemID && item.Count >= amount {
					found = true
					break
				}
			}

			if !found {
				logger.Debug("User %s does not have enough of item %s", userID, itemID)
				return false, nil
			}
		}
	}

	return true, nil
}

// Helper function to deduct reward cost from user
func (e *NakamaEventLeaderboardsSystem) deductRewardCost(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, rewardConfig *EconomyConfigReward) error {
	if rewardConfig == nil || rewardConfig.Guaranteed == nil {
		return nil
	}

	// Deduct currencies
	if rewardConfig.Guaranteed.Currencies != nil && len(rewardConfig.Guaranteed.Currencies) > 0 {
		economySystem := e.pamlogix.GetEconomySystem()
		if economySystem == nil {
			return ErrSystemNotAvailable
		}

		// Convert to negative values for deduction
		deductCurrencies := make(map[string]int64)
		for currencyID, currencyReward := range rewardConfig.Guaranteed.Currencies {
			// Use the maximum amount as the cost
			amount := currencyReward.Max
			if amount < 0 {
				amount = -amount // Convert negative to positive, then make negative for deduction
			}
			deductCurrencies[currencyID] = -amount
		}

		// Deduct currencies
		_, _, _, err := economySystem.Grant(ctx, logger, nk, userID, deductCurrencies, nil, nil, map[string]interface{}{
			"source": "event_leaderboard_cost",
		})
		if err != nil {
			logger.Error("Failed to deduct currencies: %v", err)
			return err
		}
	}

	// Deduct items
	if rewardConfig.Guaranteed.Items != nil && len(rewardConfig.Guaranteed.Items) > 0 {
		inventorySystem := e.pamlogix.GetInventorySystem()
		if inventorySystem == nil {
			return ErrSystemNotAvailable
		}

		// Convert to negative values for deduction
		deductItems := make(map[string]int64)
		for itemID, itemReward := range rewardConfig.Guaranteed.Items {
			// Use the maximum amount as the cost
			amount := itemReward.Max
			if amount < 0 {
				amount = -amount // Convert negative to positive, then make negative for deduction
			}
			deductItems[itemID] = -amount
		}

		// Deduct items
		_, _, _, _, err := inventorySystem.GrantItems(ctx, logger, nk, userID, deductItems, false)
		if err != nil {
			logger.Error("Failed to deduct items: %v", err)
			return err
		}
	}

	return nil
}

// Helper function to get the count of users who have reached the target score in a cohort
func (e *NakamaEventLeaderboardsSystem) getWinnersCount(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, eventLeaderboardID, cohortID string, targetScore int64) (int, error) {
	backingID := e.getBackingLeaderboardID(eventLeaderboardID, cohortID)

	// Get all records from the leaderboard
	records, _, _, _, err := nk.LeaderboardRecordsList(ctx, backingID, nil, 100, "", 0)
	if err != nil {
		return 0, err
	}

	winnersCount := 0
	for _, record := range records {
		if record.Score >= targetScore {
			winnersCount++
		}
	}

	return winnersCount, nil
}

// Helper function to trigger reroll for all users in a cohort when target winners are reached
func (e *NakamaEventLeaderboardsSystem) triggerCohortReroll(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, eventLeaderboardID string, config *EventLeaderboardsConfigLeaderboard, cohortID string) error {
	// Get cohort state to find all users
	cohortState, err := e.getCohortState(ctx, logger, nk, cohortID)
	if err != nil {
		logger.Error("Failed to get cohort state: %v", err)
		return err
	}

	// For each user in the cohort, reset their state to allow reroll
	for _, userID := range cohortState.UserIDs {
		userState, err := e.getUserState(ctx, logger, nk, userID)
		if err != nil {
			logger.Error("Failed to get user state for cohort reroll: %v", err)
			continue
		}

		if userEventState, exists := userState.EventLeaderboards[eventLeaderboardID]; exists {
			// Reset cohort but keep tier and other progress
			userEventState.CohortID = ""
			userEventState.HasReachedTarget = false
			userEventState.WinTimeSec = 0
			// Don't reset reroll count - this is an automatic reroll

			// Save user state
			if err := e.saveUserState(ctx, logger, nk, userID, userState); err != nil {
				logger.Error("Failed to save user state during cohort reroll: %v", err)
			}
		}
	}

	logger.Info("Triggered cohort reroll for event %s, cohort %s", eventLeaderboardID, cohortID)
	return nil
}

// Helper function to get cohort state
func (e *NakamaEventLeaderboardsSystem) getCohortState(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, cohortID string) (*EventLeaderboardCohortState, error) {
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: eventLeaderboardsStorageCollection,
			Key:        eventLeaderboardCohortPrefix + cohortID,
		},
	})
	if err != nil {
		return nil, err
	}

	if len(objects) == 0 {
		return nil, ErrBadInput
	}

	var cohortState EventLeaderboardCohortState
	if err := json.Unmarshal([]byte(objects[0].Value), &cohortState); err != nil {
		return nil, err
	}

	return &cohortState, nil
}

// Enhanced economy integration features

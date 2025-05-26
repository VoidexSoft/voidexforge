package pamlogix

import (
	"context"
	"encoding/json"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/robfig/cron/v3"
)

const (
	streaksStorageCollection = "streaks"
	userStreaksStorageKey    = "user_streaks"
)

// NakamaStreaksSystem implements the StreaksSystem interface using Nakama as the backend.
type NakamaStreaksSystem struct {
	config        *StreaksConfig
	onClaimReward OnReward[*StreaksConfigStreak]
	pamlogix      Pamlogix
	cronParser    cron.Parser
}

// NewNakamaStreaksSystem creates a new instance of the streaks system with the given configuration.
func NewNakamaStreaksSystem(config *StreaksConfig) *NakamaStreaksSystem {
	return &NakamaStreaksSystem{
		config:     config,
		cronParser: cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
	}
}

// SetPamlogix sets the Pamlogix instance for this streaks system
func (s *NakamaStreaksSystem) SetPamlogix(pl Pamlogix) {
	s.pamlogix = pl
}

// GetType returns the system type for the streaks system.
func (s *NakamaStreaksSystem) GetType() SystemType {
	return SystemTypeStreaks
}

// GetConfig returns the configuration for the streaks system.
func (s *NakamaStreaksSystem) GetConfig() any {
	return s.config
}

// List all streaks and their current state and progress for a given user.
func (s *NakamaStreaksSystem) List(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (streaks map[string]*Streak, err error) {
	if s.config == nil {
		return nil, runtime.NewError("streaks config not loaded", 13)
	}

	// Get user streaks from storage
	userStreaks, err := s.getUserStreaks(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user streaks: %v", err)
		return nil, err
	}

	now := time.Now().Unix()
	streaks = make(map[string]*Streak)

	// Process each configured streak
	for streakID, streakConfig := range s.config.Streaks {
		if streakConfig.Disabled {
			continue
		}

		// Check if streak is within its active time window
		if streakConfig.StartTimeSec > 0 && now < streakConfig.StartTimeSec {
			continue
		}
		if streakConfig.EndTimeSec > 0 && now > streakConfig.EndTimeSec {
			continue
		}

		// Get or create user streak data
		userStreak, exists := userStreaks[streakID]
		if !exists {
			userStreak = &SyncStreakUpdate{
				Count:             streakConfig.Count,
				CountCurrentReset: 0,
				ClaimCount:        0,
				CreateTimeSec:     now,
				UpdateTimeSec:     now,
				ClaimTimeSec:      0,
				ClaimedRewards:    make([]*StreakReward, 0),
			}
		}

		// Apply any scheduled resets
		userStreak = s.applyScheduledResets(logger, streakConfig, userStreak, now)

		// Build the streak response
		streak := s.buildStreakResponse(streakID, streakConfig, userStreak, now)
		streaks[streakID] = streak
	}

	return streaks, nil
}

// Update one or more streaks with the indicated counts for the given user.
func (s *NakamaStreaksSystem) Update(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, streakIDs map[string]int64) (streaks map[string]*Streak, err error) {
	if s.config == nil {
		return nil, runtime.NewError("streaks config not loaded", 13)
	}

	// Get user streaks from storage
	userStreaks, err := s.getUserStreaks(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user streaks: %v", err)
		return nil, err
	}

	now := time.Now().Unix()
	streaks = make(map[string]*Streak)
	needsSave := false

	// Process each streak update
	for streakID, updateAmount := range streakIDs {
		if updateAmount == 0 {
			continue
		}

		streakConfig, exists := s.config.Streaks[streakID]
		if !exists {
			logger.Warn("Streak config not found: %s", streakID)
			continue
		}

		if streakConfig.Disabled {
			continue
		}

		// Check if streak is within its active time window
		if streakConfig.StartTimeSec > 0 && now < streakConfig.StartTimeSec {
			continue
		}
		if streakConfig.EndTimeSec > 0 && now > streakConfig.EndTimeSec {
			continue
		}

		// Get or create user streak data
		userStreak, streakExists := userStreaks[streakID]
		if !streakExists {
			userStreak = &SyncStreakUpdate{
				Count:             streakConfig.Count,
				CountCurrentReset: 0,
				ClaimCount:        0,
				CreateTimeSec:     now,
				UpdateTimeSec:     now,
				ClaimTimeSec:      0,
				ClaimedRewards:    make([]*StreakReward, 0),
			}
			userStreaks[streakID] = userStreak
		}

		// Apply any scheduled resets
		userStreak = s.applyScheduledResets(logger, streakConfig, userStreak, now)

		// Check if we can update this streak
		if !s.canUpdateStreak(streakConfig, userStreak, now) {
			logger.Info("Cannot update streak %s at this time", streakID)
			continue
		}

		// Apply the update
		needsSave = true
		userStreak.Count += updateAmount
		userStreak.CountCurrentReset += updateAmount
		userStreak.UpdateTimeSec = now

		// Apply limits
		if userStreak.Count < 0 {
			userStreak.Count = 0
		}
		if streakConfig.MaxCount > 0 && userStreak.Count > streakConfig.MaxCount {
			userStreak.Count = streakConfig.MaxCount
		}
		if streakConfig.MaxCountCurrentReset > 0 && userStreak.CountCurrentReset > streakConfig.MaxCountCurrentReset {
			userStreak.CountCurrentReset = streakConfig.MaxCountCurrentReset
		}

		// Build the streak response
		streak := s.buildStreakResponse(streakID, streakConfig, userStreak, now)
		streaks[streakID] = streak
	}

	// Save changes if needed
	if needsSave {
		if err := s.saveUserStreaks(ctx, logger, nk, userID, userStreaks); err != nil {
			logger.Error("Failed to save user streaks: %v", err)
			return nil, err
		}
	}

	return streaks, nil
}

// Claim rewards for one or more streaks for the given user.
func (s *NakamaStreaksSystem) Claim(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, streakIDs []string) (streaks map[string]*Streak, err error) {
	if s.config == nil {
		return nil, runtime.NewError("streaks config not loaded", 13)
	}

	if s.pamlogix == nil {
		return nil, runtime.NewError("pamlogix instance not set", 13)
	}

	economySystem := s.pamlogix.GetEconomySystem()
	if economySystem == nil {
		return nil, runtime.NewError("economy system not available", 13)
	}

	// Get user streaks from storage
	userStreaks, err := s.getUserStreaks(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user streaks: %v", err)
		return nil, err
	}

	now := time.Now().Unix()
	streaks = make(map[string]*Streak)
	needsSave := false

	// Process each streak claim
	for _, streakID := range streakIDs {
		streakConfig, exists := s.config.Streaks[streakID]
		if !exists {
			logger.Warn("Streak config not found: %s", streakID)
			continue
		}

		if streakConfig.Disabled {
			continue
		}

		userStreak, streakExists := userStreaks[streakID]
		if !streakExists {
			logger.Warn("User streak not found: %s", streakID)
			continue
		}

		// Apply any scheduled resets
		userStreak = s.applyScheduledResets(logger, streakConfig, userStreak, now)

		// Check if we can claim this streak
		if !s.canClaimStreak(streakConfig, userStreak, now) {
			logger.Info("Cannot claim streak %s at this time", streakID)
			continue
		}

		// Find available rewards to claim
		availableRewards := s.getAvailableRewards(streakConfig, userStreak)
		if len(availableRewards) == 0 {
			logger.Info("No rewards available to claim for streak %s", streakID)
			continue
		}

		// Process each available reward
		for _, rewardConfig := range availableRewards {
			// Roll the reward
			rolledReward, err := economySystem.RewardRoll(ctx, logger, nk, userID, rewardConfig.Reward)
			if err != nil {
				logger.Error("Failed to roll reward for streak %s: %v", streakID, err)
				continue
			}

			if rolledReward == nil {
				continue
			}

			// Apply custom reward hook if available
			if s.onClaimReward != nil {
				rolledReward, err = s.onClaimReward(ctx, logger, nk, userID, streakID, streakConfig, rewardConfig.Reward, rolledReward)
				if err != nil {
					logger.Error("Error in onClaimReward hook for streak %s: %v", streakID, err)
					continue
				}
			}

			// Grant the reward
			_, _, _, err = economySystem.RewardGrant(ctx, logger, nk, userID, rolledReward, map[string]interface{}{
				"streak_id": streakID,
				"type":      "streak_reward",
			}, false)

			if err != nil {
				logger.Error("Failed to grant reward for streak %s: %v", streakID, err)
				continue
			}

			// Record the claimed reward
			claimedReward := &StreakReward{
				CountMin:     rewardConfig.CountMin,
				CountMax:     rewardConfig.CountMax,
				Reward:       rolledReward,
				ClaimTimeSec: now,
			}

			userStreak.ClaimedRewards = append(userStreak.ClaimedRewards, claimedReward)
		}

		// Update claim tracking
		needsSave = true
		userStreak.ClaimCount = userStreak.Count
		userStreak.ClaimTimeSec = now

		// Build the streak response
		streak := s.buildStreakResponse(streakID, streakConfig, userStreak, now)
		streaks[streakID] = streak
	}

	// Save changes if needed
	if needsSave {
		if err := s.saveUserStreaks(ctx, logger, nk, userID, userStreaks); err != nil {
			logger.Error("Failed to save user streaks: %v", err)
			return nil, err
		}
	}

	return streaks, nil
}

// Reset progress on selected streaks for the given user.
func (s *NakamaStreaksSystem) Reset(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, streakIDs []string) (streaks map[string]*Streak, err error) {
	if s.config == nil {
		return nil, runtime.NewError("streaks config not loaded", 13)
	}

	// Get user streaks from storage
	userStreaks, err := s.getUserStreaks(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get user streaks: %v", err)
		return nil, err
	}

	now := time.Now().Unix()
	streaks = make(map[string]*Streak)
	needsSave := false

	// Process each streak reset
	for _, streakID := range streakIDs {
		streakConfig, exists := s.config.Streaks[streakID]
		if !exists {
			logger.Warn("Streak config not found: %s", streakID)
			continue
		}

		userStreak, streakExists := userStreaks[streakID]
		if !streakExists {
			logger.Warn("User streak not found: %s", streakID)
			continue
		}

		// Check if we can reset this streak
		if !s.canResetStreak(streakConfig, userStreak, now) {
			logger.Info("Cannot reset streak %s at this time", streakID)
			continue
		}

		// Reset the streak
		needsSave = true
		userStreak.Count = streakConfig.Count
		userStreak.CountCurrentReset = 0
		userStreak.ClaimCount = 0
		userStreak.UpdateTimeSec = now
		userStreak.ClaimTimeSec = 0
		userStreak.ClaimedRewards = make([]*StreakReward, 0)

		// Build the streak response
		streak := s.buildStreakResponse(streakID, streakConfig, userStreak, now)
		streaks[streakID] = streak
	}

	// Save changes if needed
	if needsSave {
		if err := s.saveUserStreaks(ctx, logger, nk, userID, userStreaks); err != nil {
			logger.Error("Failed to save user streaks: %v", err)
			return nil, err
		}
	}

	return streaks, nil
}

// SetOnClaimReward sets a custom reward function which will run after a streak's reward is rolled.
func (s *NakamaStreaksSystem) SetOnClaimReward(fn OnReward[*StreaksConfigStreak]) {
	s.onClaimReward = fn
}

// Helper functions

// getUserStreaks fetches the stored streak data for a user from Nakama storage.
func (s *NakamaStreaksSystem) getUserStreaks(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (map[string]*SyncStreakUpdate, error) {
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: streaksStorageCollection,
			Key:        userStreaksStorageKey,
			UserID:     userID,
		},
	})

	if err != nil {
		logger.Error("Failed to read user streaks: %v", err)
		return nil, err
	}

	streaks := make(map[string]*SyncStreakUpdate)

	// If no data found, return empty map
	if len(objects) == 0 || objects[0].Value == "" {
		return streaks, nil
	}

	// Unmarshal the stored streak data
	syncStreaks := &SyncStreaks{}
	if err := json.Unmarshal([]byte(objects[0].Value), syncStreaks); err != nil {
		logger.Error("Failed to unmarshal user streaks: %v", err)
		return nil, err
	}

	if syncStreaks.Updates != nil {
		return syncStreaks.Updates, nil
	}

	return streaks, nil
}

// saveUserStreaks stores the updated streak data for a user in Nakama storage.
func (s *NakamaStreaksSystem) saveUserStreaks(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, streaks map[string]*SyncStreakUpdate) error {
	// Marshal the streak data
	syncStreaks := &SyncStreaks{
		Updates: streaks,
		Resets:  make([]string, 0),
	}

	data, err := json.Marshal(syncStreaks)
	if err != nil {
		logger.Error("Failed to marshal user streaks: %v", err)
		return err
	}

	// Write to storage
	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{
		{
			Collection:      streaksStorageCollection,
			Key:             userStreaksStorageKey,
			UserID:          userID,
			Value:           string(data),
			PermissionRead:  runtime.STORAGE_PERMISSION_OWNER_READ,
			PermissionWrite: runtime.STORAGE_PERMISSION_OWNER_WRITE,
		},
	})

	if err != nil {
		logger.Error("Failed to write user streaks: %v", err)
		return err
	}

	return nil
}

// calculateNextResetTime calculates the next reset time based on CRON expression
func (s *NakamaStreaksSystem) calculateNextResetTime(cronExpr string, now time.Time) (int64, error) {
	if cronExpr == "" {
		return 0, nil // No reset scheduled
	}

	sched, err := s.cronParser.Parse(cronExpr)
	if err != nil {
		return 0, err
	}

	nextReset := sched.Next(now)
	return nextReset.Unix(), nil
}

// applyScheduledResets applies any scheduled resets that should have occurred
func (s *NakamaStreaksSystem) applyScheduledResets(logger runtime.Logger, config *StreaksConfigStreak, userStreak *SyncStreakUpdate, now int64) *SyncStreakUpdate {
	if config.ResetCronexpr == "" {
		return userStreak
	}

	// Calculate when the next reset should occur
	nextResetTime, err := s.calculateNextResetTime(config.ResetCronexpr, time.Unix(userStreak.UpdateTimeSec, 0))
	if err != nil {
		logger.Error("Failed to parse CRON expression for streak: %v", err)
		return userStreak
	}

	// If we've passed the reset time, apply reset logic
	if nextResetTime > 0 && now >= nextResetTime {
		// Calculate how many reset periods have passed
		resetsPassed := int64(0)
		currentTime := time.Unix(userStreak.UpdateTimeSec, 0)
		for {
			nextTime, err := s.calculateNextResetTime(config.ResetCronexpr, currentTime)
			if err != nil || nextTime == 0 || nextTime > now {
				break
			}
			resetsPassed++
			currentTime = time.Unix(nextTime, 0)
		}

		if resetsPassed > 0 {
			// Apply idle decay if the user didn't update during reset periods
			if config.IdleCountDecayReset > 0 {
				decayAmount := resetsPassed * config.IdleCountDecayReset
				if config.MaxIdleCountDecay > 0 && decayAmount > config.MaxIdleCountDecay {
					decayAmount = config.MaxIdleCountDecay
				}
				userStreak.Count -= decayAmount
				if userStreak.Count < 0 {
					userStreak.Count = 0
				}
			}

			// Reset current reset count
			userStreak.CountCurrentReset = 0
		}
	}

	return userStreak
}

// buildStreakResponse builds a Streak response from config and user data
func (s *NakamaStreaksSystem) buildStreakResponse(streakID string, config *StreaksConfigStreak, userStreak *SyncStreakUpdate, now int64) *Streak {
	// Calculate next reset time
	var nextResetTime int64
	if config.ResetCronexpr != "" {
		nextTime, err := s.calculateNextResetTime(config.ResetCronexpr, time.Unix(now, 0))
		if err == nil {
			nextResetTime = nextTime
		}
	}

	// Calculate previous reset time
	var prevResetTime int64
	if config.ResetCronexpr != "" && nextResetTime > 0 {
		// Approximate previous reset time
		prevTime, err := s.calculateNextResetTime(config.ResetCronexpr, time.Unix(nextResetTime-86400, 0))
		if err == nil && prevTime < nextResetTime {
			prevResetTime = prevTime
		}
	}

	// Build available rewards
	availableRewards := s.getAvailableRewardsForResponse(config, userStreak)

	// Build all rewards
	allRewards := make([]*StreakAvailableReward, 0, len(config.Rewards))
	for _, rewardConfig := range config.Rewards {
		reward := &StreakAvailableReward{
			CountMin: rewardConfig.CountMin,
			CountMax: rewardConfig.CountMax,
		}
		if rewardConfig.Reward != nil {
			// Convert to AvailableRewards format using economy system
			if s.pamlogix != nil {
				if economySystem := s.pamlogix.GetEconomySystem(); economySystem != nil {
					// Use the economy system to convert the reward config to available rewards format
					// This is a reverse conversion, so we'll build it manually
					reward.Reward = s.convertRewardConfigToAvailableRewards(rewardConfig.Reward)
				}
			}
		}
		allRewards = append(allRewards, reward)
	}

	return &Streak{
		Id:                   streakID,
		Name:                 config.Name,
		Description:          config.Description,
		Count:                userStreak.Count,
		MaxCount:             config.MaxCount,
		CountCurrentReset:    userStreak.CountCurrentReset,
		MaxCountCurrentReset: config.MaxCountCurrentReset,
		IdleCountDecayReset:  config.IdleCountDecayReset,
		MaxIdleCountDecay:    config.MaxIdleCountDecay,
		PrevResetTimeSec:     prevResetTime,
		ResetTimeSec:         nextResetTime,
		CreateTimeSec:        userStreak.CreateTimeSec,
		UpdateTimeSec:        userStreak.UpdateTimeSec,
		ClaimTimeSec:         userStreak.ClaimTimeSec,
		StartTimeSec:         config.StartTimeSec,
		EndTimeSec:           config.EndTimeSec,
		Rewards:              allRewards,
		AvailableRewards:     availableRewards,
		ClaimedRewards:       userStreak.ClaimedRewards,
		CanClaim:             s.canClaimStreak(config, userStreak, now),
		CanUpdate:            s.canUpdateStreak(config, userStreak, now),
		CanReset:             s.canResetStreak(config, userStreak, now),
		ClaimCount:           userStreak.ClaimCount,
	}
}

// canUpdateStreak checks if a streak can be updated
func (s *NakamaStreaksSystem) canUpdateStreak(config *StreaksConfigStreak, userStreak *SyncStreakUpdate, now int64) bool {
	// Check time window
	if config.StartTimeSec > 0 && now < config.StartTimeSec {
		return false
	}
	if config.EndTimeSec > 0 && now > config.EndTimeSec {
		return false
	}

	// Check if we've reached max count
	if config.MaxCount > 0 && userStreak.Count >= config.MaxCount {
		return false
	}

	// Check if we've reached max count for current reset
	if config.MaxCountCurrentReset > 0 && userStreak.CountCurrentReset >= config.MaxCountCurrentReset {
		return false
	}

	return true
}

// canClaimStreak checks if a streak can be claimed
func (s *NakamaStreaksSystem) canClaimStreak(config *StreaksConfigStreak, userStreak *SyncStreakUpdate, now int64) bool {
	// Check time window
	if config.StartTimeSec > 0 && now < config.StartTimeSec {
		return false
	}
	if config.EndTimeSec > 0 && now > config.EndTimeSec {
		return false
	}

	// Check if there are any rewards available to claim
	availableRewards := s.getAvailableRewards(config, userStreak)
	return len(availableRewards) > 0
}

// canResetStreak checks if a streak can be reset
func (s *NakamaStreaksSystem) canResetStreak(config *StreaksConfigStreak, userStreak *SyncStreakUpdate, now int64) bool {
	// Check time window
	if config.StartTimeSec > 0 && now < config.StartTimeSec {
		return false
	}
	if config.EndTimeSec > 0 && now > config.EndTimeSec {
		return false
	}

	// Can always reset if there's progress
	return userStreak.Count > 0 || userStreak.CountCurrentReset > 0
}

// getAvailableRewards returns rewards that can be claimed
func (s *NakamaStreaksSystem) getAvailableRewards(config *StreaksConfigStreak, userStreak *SyncStreakUpdate) []*StreaksConfigStreakReward {
	availableRewards := make([]*StreaksConfigStreakReward, 0)

	for _, rewardConfig := range config.Rewards {
		// Check if the user's count is within the reward range
		if userStreak.Count >= rewardConfig.CountMin && userStreak.Count <= rewardConfig.CountMax {
			// Check if this reward hasn't been claimed yet
			alreadyClaimed := false
			for _, claimedReward := range userStreak.ClaimedRewards {
				if claimedReward.CountMin == rewardConfig.CountMin && claimedReward.CountMax == rewardConfig.CountMax {
					alreadyClaimed = true
					break
				}
			}

			if !alreadyClaimed {
				availableRewards = append(availableRewards, rewardConfig)
			}
		}
	}

	return availableRewards
}

// getAvailableRewardsForResponse returns available rewards in response format
func (s *NakamaStreaksSystem) getAvailableRewardsForResponse(config *StreaksConfigStreak, userStreak *SyncStreakUpdate) []*StreakAvailableReward {
	availableRewards := s.getAvailableRewards(config, userStreak)
	responseRewards := make([]*StreakAvailableReward, 0, len(availableRewards))

	for _, rewardConfig := range availableRewards {
		reward := &StreakAvailableReward{
			CountMin: rewardConfig.CountMin,
			CountMax: rewardConfig.CountMax,
		}
		if rewardConfig.Reward != nil {
			// Convert to AvailableRewards format using economy system
			if s.pamlogix != nil {
				if economySystem := s.pamlogix.GetEconomySystem(); economySystem != nil {
					// Use the economy system to convert the reward config to available rewards format
					// This is a reverse conversion, so we'll build it manually
					reward.Reward = s.convertRewardConfigToAvailableRewards(rewardConfig.Reward)
				}
			}
		}
		responseRewards = append(responseRewards, reward)
	}

	return responseRewards
}

// convertRewardConfigToAvailableRewards converts EconomyConfigReward to AvailableRewards
func (s *NakamaStreaksSystem) convertRewardConfigToAvailableRewards(rewardConfig *EconomyConfigReward) *AvailableRewards {
	if rewardConfig == nil {
		return nil
	}

	availableRewards := &AvailableRewards{
		MaxRolls:       rewardConfig.MaxRolls,
		TotalWeight:    rewardConfig.TotalWeight,
		MaxRepeatRolls: rewardConfig.MaxRepeatRolls,
	}

	// Convert guaranteed rewards
	if rewardConfig.Guaranteed != nil {
		availableRewards.Guaranteed = s.convertRewardContentsToAvailableRewardsContents(rewardConfig.Guaranteed)
	}

	// Convert weighted rewards
	if len(rewardConfig.Weighted) > 0 {
		availableRewards.Weighted = make([]*AvailableRewardsContents, len(rewardConfig.Weighted))
		for i, weighted := range rewardConfig.Weighted {
			availableRewards.Weighted[i] = s.convertRewardContentsToAvailableRewardsContents(weighted)
		}
	}

	return availableRewards
}

// convertRewardContentsToAvailableRewardsContents converts EconomyConfigRewardContents to AvailableRewardsContents
func (s *NakamaStreaksSystem) convertRewardContentsToAvailableRewardsContents(contents *EconomyConfigRewardContents) *AvailableRewardsContents {
	if contents == nil {
		return nil
	}

	availableContents := &AvailableRewardsContents{
		Weight: contents.Weight,
	}

	// Convert currencies
	if len(contents.Currencies) > 0 {
		availableContents.Currencies = make(map[string]*AvailableRewardsCurrency)
		for k, v := range contents.Currencies {
			availableContents.Currencies[k] = &AvailableRewardsCurrency{
				Count: &RewardRangeInt64{
					Min:      v.Min,
					Max:      v.Max,
					Multiple: v.Multiple,
				},
			}
		}
	}

	// Convert items
	if len(contents.Items) > 0 {
		availableContents.Items = make(map[string]*AvailableRewardsItem)
		for k, v := range contents.Items {
			item := &AvailableRewardsItem{
				Count: &RewardRangeInt64{
					Min:      v.Min,
					Max:      v.Max,
					Multiple: v.Multiple,
				},
			}

			// Convert string properties
			if len(v.StringProperties) > 0 {
				item.StringProperties = make(map[string]*AvailableRewardsStringProperty)
				for propKey, propVal := range v.StringProperties {
					stringProp := &AvailableRewardsStringProperty{
						TotalWeight: propVal.TotalWeight,
					}
					if len(propVal.Options) > 0 {
						stringProp.Options = make(map[string]*AvailableRewardsStringPropertyOption)
						for optKey, optVal := range propVal.Options {
							stringProp.Options[optKey] = &AvailableRewardsStringPropertyOption{
								Weight: optVal.Weight,
							}
						}
					}
					item.StringProperties[propKey] = stringProp
				}
			}

			// Convert numeric properties
			if len(v.NumericProperties) > 0 {
				item.NumericProperties = make(map[string]*RewardRangeDouble)
				for propKey, propVal := range v.NumericProperties {
					item.NumericProperties[propKey] = &RewardRangeDouble{
						Min:      propVal.Min,
						Max:      propVal.Max,
						Multiple: propVal.Multiple,
					}
				}
			}

			availableContents.Items[k] = item
		}
	}

	// Convert energies
	if len(contents.Energies) > 0 {
		availableContents.Energies = make(map[string]*AvailableRewardsEnergy)
		for k, v := range contents.Energies {
			availableContents.Energies[k] = &AvailableRewardsEnergy{
				Count: &RewardRangeInt32{
					Min:      v.Min,
					Max:      v.Max,
					Multiple: v.Multiple,
				},
			}
		}
	}

	// Convert item sets
	if len(contents.ItemSets) > 0 {
		availableContents.ItemSets = make([]*AvailableRewardsItemSet, len(contents.ItemSets))
		for i, itemSet := range contents.ItemSets {
			availableContents.ItemSets[i] = &AvailableRewardsItemSet{
				Count: &RewardRangeInt64{
					Min:      itemSet.Min,
					Max:      itemSet.Max,
					Multiple: itemSet.Multiple,
				},
				MaxRepeats: itemSet.MaxRepeats,
				Set:        itemSet.Set,
			}
		}
	}

	// Convert energy modifiers
	if len(contents.EnergyModifiers) > 0 {
		availableContents.EnergyModifiers = make([]*AvailableRewardsEnergyModifier, len(contents.EnergyModifiers))
		for i, modifier := range contents.EnergyModifiers {
			availableModifier := &AvailableRewardsEnergyModifier{
				Id:       modifier.Id,
				Operator: modifier.Operator,
			}
			if modifier.Value != nil {
				availableModifier.Value = &RewardRangeInt64{
					Min:      modifier.Value.Min,
					Max:      modifier.Value.Max,
					Multiple: modifier.Value.Multiple,
				}
			}
			if modifier.DurationSec != nil {
				availableModifier.DurationSec = &RewardRangeUInt64{
					Min:      modifier.DurationSec.Min,
					Max:      modifier.DurationSec.Max,
					Multiple: modifier.DurationSec.Multiple,
				}
			}
			availableContents.EnergyModifiers[i] = availableModifier
		}
	}

	// Convert reward modifiers
	if len(contents.RewardModifiers) > 0 {
		availableContents.RewardModifiers = make([]*AvailableRewardsRewardModifier, len(contents.RewardModifiers))
		for i, modifier := range contents.RewardModifiers {
			availableModifier := &AvailableRewardsRewardModifier{
				Id:       modifier.Id,
				Type:     modifier.Type,
				Operator: modifier.Operator,
			}
			if modifier.Value != nil {
				availableModifier.Value = &RewardRangeInt64{
					Min:      modifier.Value.Min,
					Max:      modifier.Value.Max,
					Multiple: modifier.Value.Multiple,
				}
			}
			if modifier.DurationSec != nil {
				availableModifier.DurationSec = &RewardRangeUInt64{
					Min:      modifier.DurationSec.Min,
					Max:      modifier.DurationSec.Max,
					Multiple: modifier.DurationSec.Multiple,
				}
			}
			availableContents.RewardModifiers[i] = availableModifier
		}
	}

	return availableContents
}

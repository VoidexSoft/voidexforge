package pamlogix

import (
	"context"
	"encoding/json"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/robfig/cron/v3"
)

const (
	achievementStorageCollection = "achievements"
	userAchievementsStorageKey   = "user_achievements"
)

// NakamaAchievementsSystem implements the AchievementsSystem interface.
type NakamaAchievementsSystem struct {
	config     *AchievementsConfig
	pamlogix   Pamlogix
	cronParser cron.Parser

	onAchievementReward      OnReward[*AchievementsConfigAchievement]
	onSubAchievementReward   OnReward[*AchievementsConfigSubAchievement]
	onAchievementTotalReward OnReward[*AchievementsConfigAchievement]
}

func NewNakamaAchievementsSystem(config *AchievementsConfig) *NakamaAchievementsSystem {
	return &NakamaAchievementsSystem{
		config:     config,
		cronParser: cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
	}
}

// Helper function to check if an achievement has expired based on duration_sec
func (s *NakamaAchievementsSystem) isAchievementExpired(ach *Achievement, achConfig *AchievementsConfigAchievement, now int64) bool {
	if achConfig.DurationSec <= 0 {
		return false // No expiration set
	}

	// Check if the achievement has an explicit expiration time set
	if ach.ExpireTimeSec > 0 {
		return now > ach.ExpireTimeSec
	}

	return false
}

// Helper function to check if a sub-achievement has expired based on duration_sec
func (s *NakamaAchievementsSystem) isSubAchievementExpired(subAch *SubAchievement, subAchConfig *AchievementsConfigSubAchievement, now int64) bool {
	if subAchConfig.DurationSec <= 0 {
		return false // No expiration set
	}

	// Check if the sub-achievement has an explicit expiration time set
	if subAch.ExpireTimeSec > 0 {
		return now > subAch.ExpireTimeSec
	}

	return false
}

// Helper function to process sub-achievement reward
func (s *NakamaAchievementsSystem) processSubAchievementReward(
	ctx context.Context,
	logger runtime.Logger,
	nk runtime.NakamaModule,
	userID string,
	parentID string,
	subID string,
	subAchConfig *AchievementsConfigSubAchievement,
	economySystem EconomySystem,
) (*Reward, error) {
	if subAchConfig.Reward == nil {
		return nil, nil
	}

	rolledSubReward, err := economySystem.RewardRoll(ctx, logger, nk, userID, subAchConfig.Reward)
	if err != nil {
		logger.Error("Failed to roll sub-achievement reward for %s: %v", subID, err)
		return nil, err
	}

	if rolledSubReward == nil {
		return nil, nil
	}

	// Process reward through hook if available
	if s.onSubAchievementReward != nil {
		var errHook error
		rolledSubReward, errHook = s.onSubAchievementReward(ctx, logger, nk, userID, subID, subAchConfig, subAchConfig.Reward, rolledSubReward)
		if errHook != nil {
			logger.Error("Error in onSubAchievementReward hook for %s (sub of %s): %v", subID, parentID, errHook)
			return rolledSubReward, errHook
		}
	}

	// Grant the reward
	_, _, _, errGrant := economySystem.RewardGrant(ctx, logger, nk, userID, rolledSubReward, map[string]interface{}{
		"achievement_id":     parentID,
		"sub_achievement_id": subID,
		"type":               "sub_achievement",
	}, false)

	if errGrant != nil {
		logger.Error("Failed to grant sub-achievement reward for %s: %v", subID, errGrant)
		return rolledSubReward, errGrant
	}

	return rolledSubReward, nil
}

// Helper function to calculate next reset time based on CRON expression
func (s *NakamaAchievementsSystem) calculateNextResetTime(cronExpr string, now time.Time) (int64, error) {
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

// Helper function to check achievement preconditions
func (s *NakamaAchievementsSystem) checkPreconditions(achievementList *AchievementList, preconditionIDs []string) bool {
	if len(preconditionIDs) == 0 {
		return true // No preconditions
	}

	for _, preconditionID := range preconditionIDs {
		ach, ok := achievementList.Achievements[preconditionID]
		if !ok || ach.ClaimTimeSec == 0 {
			// Precondition not found or not completed
			return false
		}
	}

	return true
}

// System interface methods
func (s *NakamaAchievementsSystem) GetType() SystemType {
	return SystemTypeAchievements
}

func (s *NakamaAchievementsSystem) GetConfig() any {
	return s.config
}

// AchievementsSystem interface methods
func (s *NakamaAchievementsSystem) ClaimAchievements(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, achievementIDs []string, claimTotal bool) (map[string]*Achievement, map[string]*Achievement, error) {
	if s.pamlogix == nil {
		logger.Error("Pamlogix instance not set in AchievementsSystem")
		return nil, nil, runtime.NewError("Achievements system not initialized: Pamlogix instance missing", 13) // INTERNAL
	}
	economySystem := s.pamlogix.GetEconomySystem()
	if economySystem == nil {
		logger.Error("EconomySystem not available via Pamlogix")
		return nil, nil, runtime.NewError("Achievements system not initialized: EconomySystem missing", 13) // INTERNAL
	}

	// Read user achievement state
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{{
		Collection: achievementStorageCollection,
		Key:        userAchievementsStorageKey,
		UserID:     userID,
	}})
	if err != nil {
		logger.Error("Failed to read user achievements: %v", err)
		return nil, nil, runtime.NewError("failed to read user achievements data", 13) // INTERNAL
	}

	achievementList := &AchievementList{
		Achievements:       make(map[string]*Achievement),
		RepeatAchievements: make(map[string]*Achievement),
	}
	var version string
	if len(objects) > 0 && objects[0].Value != "" {
		if err := json.Unmarshal([]byte(objects[0].Value), achievementList); err != nil {
			logger.Error("Failed to unmarshal user achievements: %v", err)
			return nil, nil, runtime.NewError("failed to parse user achievements data", 13) // INTERNAL
		}
		version = objects[0].Version
	}

	now := time.Now().Unix()
	updatedAchievements := make(map[string]*Achievement)
	updatedRepeatAchievements := make(map[string]*Achievement)

	for _, id := range achievementIDs {
		ach, ok := achievementList.Achievements[id]
		if !ok {
			ach, ok = achievementList.RepeatAchievements[id]
			if !ok {
				logger.Warn("Achievement ID not found: %s", id)
				continue
			}
			// Repeat achievement
			achConfig, configExists := s.config.Achievements[id]
			if !configExists {
				logger.Warn("Config for achievement ID not found: %s", id)
				continue
			}

			// Check if the achievement has expired
			if s.isAchievementExpired(ach, achConfig, now) {
				logger.Info("Achievement %s has expired and cannot be claimed", id)
				continue
			}

			if ach.ClaimTimeSec == 0 && ach.Count >= ach.MaxCount {
				// Check preconditions
				if !s.checkPreconditions(achievementList, achConfig.PreconditionIDs) {
					logger.Info("Achievement %s preconditions not met", id)
					continue
				}

				ach.ClaimTimeSec = now
				if achConfig.Reward != nil {
					rolledReward, err := economySystem.RewardRoll(ctx, logger, nk, userID, achConfig.Reward)
					if err != nil {
						logger.Error("Failed to roll reward for achievement %s: %v", id, err)
					} else if rolledReward != nil {
						if s.onAchievementReward != nil {
							rolledReward, err = s.onAchievementReward(ctx, logger, nk, userID, id, achConfig, achConfig.Reward, rolledReward)
							if err != nil {
								logger.Error("Error in onAchievementReward hook for %s: %v", id, err)
							}
						}
						_, _, _, err = economySystem.RewardGrant(ctx, logger, nk, userID, rolledReward, map[string]interface{}{"achievement_id": id, "type": "repeat"}, false)
						if err != nil {
							logger.Error("Failed to grant reward for achievement %s: %v", id, err)
						} else {
							ach.Reward = rolledReward
						}
					}
				}
				updatedRepeatAchievements[id] = ach

				// Handle auto-reset for repeatable achievements
				if achConfig.AutoReset {
					ach.Count = 0
					ach.ClaimTimeSec = 0
					ach.Reward = nil

					// Calculate next reset time if CRON is specified
					if achConfig.ResetCronexpr != "" {
						nextResetTime, cronErr := s.calculateNextResetTime(achConfig.ResetCronexpr, time.Unix(now, 0))
						if cronErr == nil && nextResetTime > 0 {
							ach.ResetTimeSec = nextResetTime
						} else if cronErr != nil {
							logger.Error("Failed to parse CRON expression for achievement %s: %v", id, cronErr)
						}
					}
				}
			}
			continue
		}
		// One-off achievement
		achConfig, configExists := s.config.Achievements[id]
		if !configExists {
			logger.Warn("Config for achievement ID not found: %s", id)
			continue
		}

		// Check if the achievement has expired
		if s.isAchievementExpired(ach, achConfig, now) {
			logger.Info("Achievement %s has expired and cannot be claimed", id)
			continue
		}

		// Check preconditions
		if !s.checkPreconditions(achievementList, achConfig.PreconditionIDs) {
			logger.Info("Achievement %s preconditions not met", id)
			continue
		}

		if ach.ClaimTimeSec == 0 && ach.Count >= achConfig.MaxCount {
			ach.ClaimTimeSec = now
			if achConfig.Reward != nil {
				rolledReward, err := economySystem.RewardRoll(ctx, logger, nk, userID, achConfig.Reward)
				if err != nil {
					logger.Error("Failed to roll reward for achievement %s: %v", id, err)
				} else if rolledReward != nil {
					if s.onAchievementReward != nil {
						rolledReward, err = s.onAchievementReward(ctx, logger, nk, userID, id, achConfig, achConfig.Reward, rolledReward)
						if err != nil {
							logger.Error("Error in onAchievementReward hook for %s: %v", id, err)
						}
					}
					_, _, _, err = economySystem.RewardGrant(ctx, logger, nk, userID, rolledReward, map[string]interface{}{"achievement_id": id, "type": "standard"}, false)
					if err != nil {
						logger.Error("Failed to grant reward for achievement %s: %v", id, err)
					} else {
						ach.Reward = rolledReward
					}
				}
			}
			updatedAchievements[id] = ach

			// Also try to claim total reward if auto_claim_total is enabled
			if achConfig.AutoClaimTotal {
				claimTotal = true
			}
		}

		if claimTotal && ach.TotalClaimTimeSec == 0 && ach.Count >= achConfig.MaxCount {
			// Check if all sub-achievements are completed
			allSubAchievementsComplete := true
			if len(achConfig.SubAchievements) > 0 {
				for subID, subAchConfig := range achConfig.SubAchievements {
					userSubAch, subAchExists := ach.SubAchievements[subID]
					if !subAchExists || userSubAch.Count < subAchConfig.MaxCount || userSubAch.ClaimTimeSec == 0 {
						allSubAchievementsComplete = false
						break
					}
				}
			}

			if allSubAchievementsComplete {
				ach.TotalClaimTimeSec = now
				if achConfig.TotalReward != nil {
					rolledTotalReward, err := economySystem.RewardRoll(ctx, logger, nk, userID, achConfig.TotalReward)
					if err != nil {
						logger.Error("Failed to roll total reward for achievement %s: %v", id, err)
					} else if rolledTotalReward != nil {
						if s.onAchievementTotalReward != nil {
							rolledTotalReward, err = s.onAchievementTotalReward(ctx, logger, nk, userID, id, achConfig, achConfig.TotalReward, rolledTotalReward)
							if err != nil {
								logger.Error("Error in onAchievementTotalReward hook for %s: %v", id, err)
							}
						}
						_, _, _, err = economySystem.RewardGrant(ctx, logger, nk, userID, rolledTotalReward, map[string]interface{}{"achievement_id": id, "type": "total"}, false)
						if err != nil {
							logger.Error("Failed to grant total reward for achievement %s: %v", id, err)
						} else {
							ach.TotalReward = rolledTotalReward
						}
					}
				}
				updatedAchievements[id] = ach
			}
		}
	}

	// Save updated state
	data, err := json.Marshal(achievementList)
	if err != nil {
		logger.Error("Failed to marshal updated achievements: %v", err)
		return nil, nil, runtime.NewError("failed to serialize achievements update", 13) // INTERNAL
	}
	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{{
		Collection:      achievementStorageCollection,
		Key:             userAchievementsStorageKey,
		UserID:          userID,
		Value:           string(data),
		Version:         version,
		PermissionRead:  runtime.STORAGE_PERMISSION_OWNER_READ,
		PermissionWrite: runtime.STORAGE_PERMISSION_OWNER_WRITE,
	}})
	if err != nil {
		logger.Error("Failed to write updated achievements: %v", err)
		return nil, nil, runtime.NewError("failed to save achievements update", 13) // INTERNAL
	}

	return updatedAchievements, updatedRepeatAchievements, nil
}

func (s *NakamaAchievementsSystem) GetAchievements(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (map[string]*Achievement, map[string]*Achievement, error) {
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{{
		Collection: achievementStorageCollection,
		Key:        userAchievementsStorageKey,
		UserID:     userID,
	}})
	if err != nil {
		logger.Error("Failed to read user achievements in GetAchievements: %v", err)
		return nil, nil, runtime.NewError("failed to read user achievements data", 13) // INTERNAL
	}

	achievementList := &AchievementList{
		Achievements:       make(map[string]*Achievement),
		RepeatAchievements: make(map[string]*Achievement),
	}

	if len(objects) > 0 && objects[0].Value != "" {
		if err := json.Unmarshal([]byte(objects[0].Value), achievementList); err != nil {
			logger.Error("Failed to unmarshal user achievements in GetAchievements: %v", err)
			return nil, nil, runtime.NewError("failed to parse user achievements data", 13) // INTERNAL
		}
	}

	return achievementList.Achievements, achievementList.RepeatAchievements, nil
}

func (s *NakamaAchievementsSystem) UpdateAchievements(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, achievementUpdates map[string]int64) (map[string]*Achievement, map[string]*Achievement, error) {
	// Read user achievement state
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{{
		Collection: achievementStorageCollection,
		Key:        userAchievementsStorageKey,
		UserID:     userID,
	}})
	if err != nil {
		logger.Error("Failed to read user achievements in UpdateAchievements: %v", err)
		return nil, nil, runtime.NewError("failed to read user achievements data", 13) // INTERNAL
	}

	achievementList := &AchievementList{
		Achievements:       make(map[string]*Achievement),
		RepeatAchievements: make(map[string]*Achievement),
	}
	var version string
	if len(objects) > 0 && objects[0].Value != "" {
		if errUnmarshal := json.Unmarshal([]byte(objects[0].Value), achievementList); errUnmarshal != nil {
			logger.Error("Failed to unmarshal user achievements in UpdateAchievements: %v", errUnmarshal)
			return nil, nil, runtime.NewError("failed to parse user achievements data", 13) // INTERNAL
		}
		version = objects[0].Version
	}

	now := time.Now().Unix()
	updatedStandardAchievements := make(map[string]*Achievement)
	updatedRepeatAchievements := make(map[string]*Achievement)
	achsToAutoClaim := make([]string, 0)
	needsSave := false

	if s.pamlogix == nil {
		logger.Error("Pamlogix instance not set in AchievementsSystem for UpdateAchievements")
		return nil, nil, runtime.NewError("Achievements system not initialized: Pamlogix instance missing", 13)
	}
	economySystem := s.pamlogix.GetEconomySystem()
	if economySystem == nil {
		logger.Error("EconomySystem not available via Pamlogix for UpdateAchievements")
		return nil, nil, runtime.NewError("Achievements system not initialized: EconomySystem missing", 13)
	}

	for id, updateAmount := range achievementUpdates {
		if updateAmount == 0 {
			continue
		}

		achConfig, configExists := s.config.Achievements[id]
		if !configExists {
			logger.Warn("UpdateAchievements: Config for achievement ID not found: %s", id)
			continue
		}

		// Check standard achievements first
		if ach, ok := achievementList.Achievements[id]; ok {
			// Check if achievement has expired
			if s.isAchievementExpired(ach, achConfig, now) {
				logger.Info("Achievement %s has expired and cannot be progressed", id)
				continue
			}

			// Check if the reset time has passed and we need to reset
			if ach.ResetTimeSec > 0 && now > ach.ResetTimeSec {
				logger.Info("Achievement %s is being reset due to scheduled reset time", id)
				ach.Count = 0
				ach.ClaimTimeSec = 0
				ach.Reward = nil

				// Calculate next reset time
				if achConfig.ResetCronexpr != "" {
					nextResetTime, cronErr := s.calculateNextResetTime(achConfig.ResetCronexpr, time.Unix(now, 0))
					if cronErr == nil && nextResetTime > 0 {
						ach.ResetTimeSec = nextResetTime
					} else if cronErr != nil {
						logger.Error("Failed to parse CRON expression for achievement %s: %v", id, cronErr)
					}
				}
			}

			// Check preconditions
			if !s.checkPreconditions(achievementList, achConfig.PreconditionIDs) {
				logger.Info("Achievement %s preconditions not met, cannot progress", id)
				continue
			}

			needsSave = true
			ach.Count += updateAmount
			if ach.Count < 0 {
				ach.Count = 0
			}
			if ach.MaxCount > 0 && ach.Count > ach.MaxCount {
				ach.Count = ach.MaxCount
			}
			ach.CurrentTimeSec = now

			// If achievement is completed and auto-claim is enabled, add to the auto-claim list
			if ach.Count >= ach.MaxCount && ach.ClaimTimeSec == 0 && achConfig.AutoClaim {
				achsToAutoClaim = append(achsToAutoClaim, id)
			}

			// Check sub-achievements for completion
			if len(achConfig.SubAchievements) > 0 {
				for subID, subAchConfig := range achConfig.SubAchievements {
					userSubAch, subAchExists := ach.SubAchievements[subID]
					if !subAchExists {
						userSubAch = &SubAchievement{Id: subID, Count: 0, CurrentTimeSec: now}
						if subAchConfig.DurationSec > 0 {
							userSubAch.ExpireTimeSec = now + subAchConfig.DurationSec
						}
						ach.SubAchievements[subID] = userSubAch
					}

					// Check if sub-achievement has expired
					if s.isSubAchievementExpired(userSubAch, subAchConfig, now) {
						logger.Info("Sub-achievement %s has expired and cannot be progressed", subID)
						continue
					}

					// Check preconditions for sub-achievement
					if !s.checkPreconditions(achievementList, subAchConfig.PreconditionIDs) {
						logger.Info("Sub-achievement %s preconditions not met, cannot progress", subID)
						continue
					}

					if userSubAch.Count < subAchConfig.MaxCount {
						userSubAch.Count += updateAmount
						if userSubAch.Count < 0 {
							userSubAch.Count = 0
						}
						if userSubAch.Count > subAchConfig.MaxCount {
							userSubAch.Count = subAchConfig.MaxCount
						}
						userSubAch.CurrentTimeSec = now
					}

					if userSubAch.Count >= subAchConfig.MaxCount && userSubAch.ClaimTimeSec == 0 {
						// Handle auto-claim for sub-achievements
						if subAchConfig.AutoClaim {
							userSubAch.ClaimTimeSec = now

							if subAchConfig.Reward != nil {
								rolledSubReward, errRoll := economySystem.RewardRoll(ctx, logger, nk, userID, subAchConfig.Reward)
								if errRoll != nil {
									logger.Error("Failed to roll sub-achievement reward for %s: %v", subID, errRoll)
								} else if rolledSubReward != nil {
									if s.onSubAchievementReward != nil {
										var errHook error
										rolledSubReward, errHook = s.onSubAchievementReward(ctx, logger, nk, userID, subID, subAchConfig, subAchConfig.Reward, rolledSubReward)
										if errHook != nil {
											logger.Error("Error in onSubAchievementReward hook for %s (sub of %s): %v", subID, id, errHook)
										}
									}

									_, _, _, errGrant := economySystem.RewardGrant(ctx, logger, nk, userID, rolledSubReward, map[string]interface{}{
										"achievement_id":     id,
										"sub_achievement_id": subID,
										"type":               "sub_achievement",
									}, false)

									if errGrant != nil {
										logger.Error("Failed to grant sub-achievement reward for %s: %v", subID, errGrant)
									} else {
										userSubAch.Reward = rolledSubReward
									}
								}
							}

							// Check if sub-achievement should auto-reset
							if subAchConfig.AutoReset {
								userSubAch.Count = 0
								userSubAch.ClaimTimeSec = 0
								userSubAch.Reward = nil

								// Calculate next reset time for sub-achievement
								if subAchConfig.ResetCronexpr != "" {
									nextResetTime, cronErr := s.calculateNextResetTime(subAchConfig.ResetCronexpr, time.Unix(now, 0))
									if cronErr == nil && nextResetTime > 0 {
										userSubAch.ResetTimeSec = nextResetTime
									} else if cronErr != nil {
										logger.Error("Failed to parse CRON expression for sub-achievement %s: %v", subID, cronErr)
									}
								}
							}
						} else if s.onSubAchievementReward != nil && subAchConfig.Reward != nil {
							// Still trigger the reward hook for processing, but don't auto-claim
							rolledSubReward, errRoll := economySystem.RewardRoll(ctx, logger, nk, userID, subAchConfig.Reward)
							if errRoll != nil {
								logger.Error("Failed to roll sub-achievement reward for %s (new parent %s): %v", subID, id, errRoll)
							} else if rolledSubReward != nil {
								_, errHook := s.onSubAchievementReward(ctx, logger, nk, userID, subID, subAchConfig, subAchConfig.Reward, rolledSubReward)
								if errHook != nil {
									logger.Error("Error in onSubAchievementReward hook for %s (new parent %s): %v", subID, id, errHook)
								}
							}
						}
					}
				}
			}

			updatedStandardAchievements[id] = ach
			continue
		}

		// Check repeat achievements if not found in standard
		if achConfig.IsRepeatable {
			if ach, ok := achievementList.RepeatAchievements[id]; ok {
				// Check if achievement has expired
				if s.isAchievementExpired(ach, achConfig, now) {
					logger.Info("Repeatable achievement %s has expired and cannot be progressed", id)
					continue
				}

				// Check if the reset time has passed and we need to reset
				if ach.ResetTimeSec > 0 && now > ach.ResetTimeSec {
					logger.Info("Repeatable achievement %s is being reset due to scheduled reset time", id)
					ach.Count = 0
					ach.ClaimTimeSec = 0
					ach.Reward = nil

					// Calculate next reset time
					if achConfig.ResetCronexpr != "" {
						nextResetTime, cronErr := s.calculateNextResetTime(achConfig.ResetCronexpr, time.Unix(now, 0))
						if cronErr == nil && nextResetTime > 0 {
							ach.ResetTimeSec = nextResetTime
						} else if cronErr != nil {
							logger.Error("Failed to parse CRON expression for achievement %s: %v", id, cronErr)
						}
					}
				}

				// Check preconditions
				if !s.checkPreconditions(achievementList, achConfig.PreconditionIDs) {
					logger.Info("Achievement %s preconditions not met, cannot progress", id)
					continue
				}

				needsSave = true
				ach.Count += updateAmount
				if ach.Count < 0 {
					ach.Count = 0
				}
				if ach.MaxCount > 0 && ach.Count > ach.MaxCount {
					ach.Count = ach.MaxCount
				}
				ach.CurrentTimeSec = now

				// Auto-claim for repeatable achievement
				if ach.Count >= ach.MaxCount && ach.ClaimTimeSec == 0 && achConfig.AutoClaim {
					achsToAutoClaim = append(achsToAutoClaim, id)
				}

				// For repeatable achievements that are already claimed, check if they should reset
				if ach.Count >= ach.MaxCount && ach.ClaimTimeSec > 0 {
					ach.Count = ach.Count % ach.MaxCount
					ach.ClaimTimeSec = 0
					ach.Reward = nil
				}

				updatedRepeatAchievements[id] = ach
			} else {
				// Check preconditions before creating a new achievement
				if !s.checkPreconditions(achievementList, achConfig.PreconditionIDs) {
					logger.Info("Achievement %s preconditions not met, cannot create", id)
					continue
				}

				needsSave = true
				newAch := &Achievement{
					Id:             id,
					Count:          updateAmount,
					MaxCount:       achConfig.MaxCount,
					CurrentTimeSec: now,
				}

				// Set initial count from config
				if achConfig.Count > 0 {
					newAch.Count += achConfig.Count
				}

				if newAch.Count < 0 {
					newAch.Count = 0
				}
				if newAch.MaxCount > 0 && newAch.Count > newAch.MaxCount {
					newAch.Count = newAch.MaxCount
				}

				// Set expiration time if duration is specified
				if achConfig.DurationSec > 0 {
					newAch.ExpireTimeSec = now + achConfig.DurationSec
				}

				// Calculate next reset time if CRON is specified
				if achConfig.ResetCronexpr != "" {
					nextResetTime, cronErr := s.calculateNextResetTime(achConfig.ResetCronexpr, time.Unix(now, 0))
					if cronErr == nil && nextResetTime > 0 {
						newAch.ResetTimeSec = nextResetTime
					} else if cronErr != nil {
						logger.Error("Failed to parse CRON expression for achievement %s: %v", id, cronErr)
					}
				}

				achievementList.RepeatAchievements[id] = newAch
				updatedRepeatAchievements[id] = newAch

				// Auto-claim for new repeatable achievement
				if newAch.Count >= newAch.MaxCount && achConfig.AutoClaim {
					achsToAutoClaim = append(achsToAutoClaim, id)
				}
			}
		} else {
			// Check preconditions before creating a new standard achievement
			if !s.checkPreconditions(achievementList, achConfig.PreconditionIDs) {
				logger.Info("Achievement %s preconditions not met, cannot create", id)
				continue
			}

			needsSave = true
			newAch := &Achievement{
				Id:              id,
				Count:           updateAmount,
				MaxCount:        achConfig.MaxCount,
				CurrentTimeSec:  now,
				SubAchievements: make(map[string]*SubAchievement),
			}

			// Set initial count from config
			if achConfig.Count > 0 {
				newAch.Count += achConfig.Count
			}

			if newAch.Count < 0 {
				newAch.Count = 0
			}
			if newAch.MaxCount > 0 && newAch.Count > newAch.MaxCount {
				newAch.Count = newAch.MaxCount
			}

			// Set expiration time if duration is specified
			if achConfig.DurationSec > 0 {
				newAch.ExpireTimeSec = now + achConfig.DurationSec
			}

			// Calculate next reset time if CRON is specified
			if achConfig.ResetCronexpr != "" {
				nextResetTime, cronErr := s.calculateNextResetTime(achConfig.ResetCronexpr, time.Unix(now, 0))
				if cronErr == nil && nextResetTime > 0 {
					newAch.ResetTimeSec = nextResetTime
				} else if cronErr != nil {
					logger.Error("Failed to parse CRON expression for achievement %s: %v", id, cronErr)
				}
			}

			achievementList.Achievements[id] = newAch
			updatedStandardAchievements[id] = newAch

			// Process sub-achievements for the newly created standard achievement
			if len(achConfig.SubAchievements) > 0 {
				for subID, subAchConfig_local := range achConfig.SubAchievements {
					userSubAch, subAchExists := newAch.SubAchievements[subID] // newAch is the parent
					if !subAchExists {
						userSubAch = &SubAchievement{Id: subID, Count: 0, CurrentTimeSec: now}

						// Set initial count from config
						if subAchConfig_local.Count > 0 {
							userSubAch.Count = subAchConfig_local.Count
						}

						// Set expiration time if duration is specified
						if subAchConfig_local.DurationSec > 0 {
							userSubAch.ExpireTimeSec = now + subAchConfig_local.DurationSec
						}

						// Calculate next reset time for sub-achievement if CRON is specified
						if subAchConfig_local.ResetCronexpr != "" {
							nextResetTime, cronErr := s.calculateNextResetTime(subAchConfig_local.ResetCronexpr, time.Unix(now, 0))
							if cronErr == nil && nextResetTime > 0 {
								userSubAch.ResetTimeSec = nextResetTime
							} else if cronErr != nil {
								logger.Error("Failed to parse CRON expression for sub-achievement %s: %v", subID, cronErr)
							}
						}

						newAch.SubAchievements[subID] = userSubAch
					}

					// Ensure sub-achievement count progresses if applicable
					if userSubAch.Count < subAchConfig_local.MaxCount {
						userSubAch.Count += updateAmount // Use parent's updateAmount
						if userSubAch.Count < 0 {
							userSubAch.Count = 0
						}
						if userSubAch.MaxCount > 0 && userSubAch.Count > subAchConfig_local.MaxCount {
							userSubAch.Count = subAchConfig_local.MaxCount
						}
						userSubAch.CurrentTimeSec = now
					}

					// Check for sub-achievement completion and trigger reward hook
					if userSubAch.Count >= subAchConfig_local.MaxCount && userSubAch.ClaimTimeSec == 0 {
						// Handle auto-claim for sub-achievements
						if subAchConfig_local.AutoClaim {
							userSubAch.ClaimTimeSec = now

							// Process sub-achievement reward
							rolledSubReward, err := s.processSubAchievementReward(ctx, logger, nk, userID, id, subID, subAchConfig_local, economySystem)
							if err == nil && rolledSubReward != nil {
								userSubAch.Reward = rolledSubReward
							}

							// Check if sub-achievement should auto-reset
							if subAchConfig_local.AutoReset {
								userSubAch.Count = 0
								userSubAch.ClaimTimeSec = 0
								userSubAch.Reward = nil

								// Calculate next reset time for sub-achievement
								if subAchConfig_local.ResetCronexpr != "" {
									nextResetTime, cronErr := s.calculateNextResetTime(subAchConfig_local.ResetCronexpr, time.Unix(now, 0))
									if cronErr == nil && nextResetTime > 0 {
										userSubAch.ResetTimeSec = nextResetTime
									} else if cronErr != nil {
										logger.Error("Failed to parse CRON expression for sub-achievement %s: %v", subID, cronErr)
									}
								}
							}
						} else if s.onSubAchievementReward != nil && subAchConfig_local.Reward != nil {
							// Just trigger the reward hook for processing, but don't auto-claim or grant
							rolledSubReward, errRoll := economySystem.RewardRoll(ctx, logger, nk, userID, subAchConfig_local.Reward)
							if errRoll != nil {
								logger.Error("Failed to roll sub-achievement reward for %s (new parent %s): %v", subID, id, errRoll)
							} else if rolledSubReward != nil {
								// We don't care about the returned reward since we're not granting it
								_, errHook := s.onSubAchievementReward(ctx, logger, nk, userID, subID, subAchConfig_local, subAchConfig_local.Reward, rolledSubReward)
								if errHook != nil {
									logger.Error("Error in onSubAchievementReward hook for %s (sub of %s): %v", subID, id, errHook)
								}
							}
						}
					}
				}
			}

			// Auto-claim for new standard achievement
			if newAch.Count >= newAch.MaxCount && achConfig.AutoClaim {
				achsToAutoClaim = append(achsToAutoClaim, id)
			}
		}
	}

	// Process auto-claim achievements
	if len(achsToAutoClaim) > 0 {
		claimAchievements, claimRepeatAchievements, errClaim := s.ClaimAchievements(ctx, logger, nk, userID, achsToAutoClaim, false)
		if errClaim != nil {
			logger.Error("Failed to auto-claim achievements: %v", errClaim)
		} else {
			// Add auto-claimed achievements to the updated maps
			for id, ach := range claimAchievements {
				updatedStandardAchievements[id] = ach
			}
			for id, ach := range claimRepeatAchievements {
				updatedRepeatAchievements[id] = ach
			}
		}
	}

	if needsSave {
		data, errMarshal := json.Marshal(achievementList)
		if errMarshal != nil {
			logger.Error("Failed to marshal updated achievements in UpdateAchievements: %v", errMarshal)
			return nil, nil, runtime.NewError("failed to serialize achievements update", 13) // INTERNAL
		}
		_, errWrite := nk.StorageWrite(ctx, []*runtime.StorageWrite{{
			Collection:      achievementStorageCollection,
			Key:             userAchievementsStorageKey,
			UserID:          userID,
			Value:           string(data),
			Version:         version,
			PermissionRead:  runtime.STORAGE_PERMISSION_OWNER_READ,
			PermissionWrite: runtime.STORAGE_PERMISSION_OWNER_WRITE,
		}})
		if errWrite != nil {
			logger.Error("Failed to write updated achievements in UpdateAchievements: %v", errWrite)
			return nil, nil, runtime.NewError("failed to save achievements update", 13) // INTERNAL
		}
	}

	return updatedStandardAchievements, updatedRepeatAchievements, nil
}

func (s *NakamaAchievementsSystem) SetOnAchievementReward(fn OnReward[*AchievementsConfigAchievement]) {
	s.onAchievementReward = fn
}

func (s *NakamaAchievementsSystem) SetOnSubAchievementReward(fn OnReward[*AchievementsConfigSubAchievement]) {
	s.onSubAchievementReward = fn
}

func (s *NakamaAchievementsSystem) SetOnAchievementTotalReward(fn OnReward[*AchievementsConfigAchievement]) {
	s.onAchievementTotalReward = fn
}

// SetPamlogix sets the Pamlogix instance for this achievements system.
func (s *NakamaAchievementsSystem) SetPamlogix(pl Pamlogix) {
	s.pamlogix = pl
}

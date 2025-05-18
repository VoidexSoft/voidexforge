package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
)

// RPC handler function placeholders for achievements
func rpcAchievementsClaim(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		achievementsSystem := p.GetAchievementsSystem()
		if achievementsSystem == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		var request struct {
			AchievementIds []string `json:"achievement_ids"`
			ClaimTotal     bool     `json:"claim_total,omitempty"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal AchievementsClaimRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Validate request
		if len(request.AchievementIds) == 0 {
			logger.Error("No achievement IDs provided in request")
			return "", runtime.NewError("achievement_ids is required and cannot be empty", 3) // INVALID_ARGUMENT
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Call the achievements system to claim achievements
		achievements, repeatAchievements, err := achievementsSystem.ClaimAchievements(ctx, logger, nk, userID, request.AchievementIds, request.ClaimTotal)
		if err != nil {
			logger.Error("Error claiming achievements: %v", err)
			return "", err
		}

		// Organize rewards for the response
		claimedRewards := make(map[string]*Reward)
		claimedTotalRewards := make(map[string]*Reward)

		// Extract rewards from completed achievements
		for id, ach := range achievements {
			if ach.Reward != nil {
				claimedRewards[id] = ach.Reward
			}
			if ach.TotalReward != nil && request.ClaimTotal {
				claimedTotalRewards[id] = ach.TotalReward
			}
		}

		// Extract rewards from repeat achievements
		for id, ach := range repeatAchievements {
			if ach.Reward != nil {
				claimedRewards[id] = ach.Reward
			}
		}

		// Prepare the response
		response := struct {
			Achievements        map[string]*Achievement `json:"achievements"`
			RepeatAchievements  map[string]*Achievement `json:"repeat_achievements,omitempty"`
			ClaimedRewards      map[string]*Reward      `json:"claimed_rewards,omitempty"`
			ClaimedTotalRewards map[string]*Reward      `json:"claimed_total_rewards,omitempty"`
			ClaimCount          int                     `json:"claim_count"`
		}{
			Achievements:        achievements,
			RepeatAchievements:  repeatAchievements,
			ClaimedRewards:      claimedRewards,
			ClaimedTotalRewards: claimedTotalRewards,
			ClaimCount:          len(achievements) + len(repeatAchievements),
		}

		// Encode the response
		responseData, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

func rpcAchievementsGet(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		achievementsSystem := p.GetAchievementsSystem()
		if achievementsSystem == nil {
			return "", ErrSystemNotFound
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Call the achievements system to get achievements
		achievements, repeatAchievements, err := achievementsSystem.GetAchievements(ctx, logger, nk, userID)
		if err != nil {
			logger.Error("Error getting achievements: %v", err)
			return "", err
		}

		// Prepare the response
		response := struct {
			Achievements       map[string]*Achievement `json:"achievements"`
			RepeatAchievements map[string]*Achievement `json:"repeat_achievements,omitempty"`
		}{
			Achievements:       achievements,
			RepeatAchievements: repeatAchievements,
		}

		// Encode the response
		responseData, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

func rpcAchievementsUpdate(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		achievementsSystem := p.GetAchievementsSystem()
		if achievementsSystem == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		var request struct {
			Updates map[string]int64 `json:"updates"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal AchievementsUpdateRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Call the achievements system to update progress
		achievements, repeatAchievements, err := achievementsSystem.UpdateAchievements(ctx, logger, nk, userID, request.Updates)
		if err != nil {
			logger.Error("Error updating achievement progress: %v", err)
			return "", err
		}

		// Prepare the response
		response := struct {
			Achievements       map[string]*Achievement `json:"achievements"`
			RepeatAchievements map[string]*Achievement `json:"repeat_achievements,omitempty"`
		}{
			Achievements:       achievements,
			RepeatAchievements: repeatAchievements,
		}

		// Encode the response
		responseData, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

// rpcAchievementsList handles the RPC to list achievements with optional filtering
func rpcAchievementsList(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		achievementsSystem := p.GetAchievementsSystem()
		if achievementsSystem == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request for potential filters
		var request struct {
			Category     string   `json:"category,omitempty"`
			Categories   []string `json:"categories,omitempty"`
			OnlyComplete bool     `json:"only_complete,omitempty"`
			OnlyProgress bool     `json:"only_progress,omitempty"`
		}

		// Try to parse optional filters, but continue even if parsing fails
		if payload != "" {
			if err := json.Unmarshal([]byte(payload), &request); err != nil {
				logger.Warn("Failed to unmarshal AchievementsListRequest, proceeding without filters: %v", err)
				// We don't return an error here, just proceed without filters
			}
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Call the achievements system to get achievements
		achievements, repeatAchievements, err := achievementsSystem.GetAchievements(ctx, logger, nk, userID)
		if err != nil {
			logger.Error("Error getting achievements: %v", err)
			return "", err
		}

		// Apply filters if specified
		filteredAchievements := make(map[string]*Achievement)
		filteredRepeatAchievements := make(map[string]*Achievement)

		// Filter regular achievements
		for id, ach := range achievements {
			if !passesFilter(ach, request.Category, request.Categories, request.OnlyComplete, request.OnlyProgress) {
				continue
			}
			filteredAchievements[id] = ach
		}

		// Filter repeat achievements
		for id, ach := range repeatAchievements {
			if !passesFilter(ach, request.Category, request.Categories, request.OnlyComplete, request.OnlyProgress) {
				continue
			}
			filteredRepeatAchievements[id] = ach
		}

		// Prepare the response
		response := struct {
			Achievements       map[string]*Achievement `json:"achievements"`
			RepeatAchievements map[string]*Achievement `json:"repeat_achievements,omitempty"`
		}{
			Achievements:       filteredAchievements,
			RepeatAchievements: filteredRepeatAchievements,
		}

		// Encode the response
		responseData, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

// Helper function to check if an achievement passes all specified filters
func passesFilter(ach *Achievement, category string, categories []string, onlyComplete bool, onlyProgress bool) bool {
	// Filter by category if specified
	if category != "" && ach.Category != category {
		return false
	}

	// Filter by categories if specified
	if len(categories) > 0 {
		categoryMatch := false
		for _, cat := range categories {
			if ach.Category == cat {
				categoryMatch = true
				break
			}
		}
		if !categoryMatch {
			return false
		}
	}

	// Filter by completion status if specified
	if onlyComplete && (ach.ClaimTimeSec == 0 || ach.Count < ach.MaxCount) {
		return false
	}

	// Filter by progress status if specified
	if onlyProgress && (ach.Count == 0 || ach.ClaimTimeSec > 0) {
		return false
	}

	return true
}

// rpcAchievementsProgress handles the RPC to update achievement progress
func rpcAchievementsProgress(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		achievementsSystem := p.GetAchievementsSystem()
		if achievementsSystem == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		var request struct {
			AchievementId string `json:"achievement_id"`
			Progress      int64  `json:"progress"`
			Absolute      bool   `json:"absolute,omitempty"` // If true, set progress to exact value rather than increment
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal AchievementsProgressRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Validate input
		if request.AchievementId == "" {
			logger.Error("Achievement ID is required")
			return "", runtime.NewError("achievement_id is required", 3) // INVALID_ARGUMENT
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// If this is an absolute update, we need to get the current progress first
		// to calculate the delta for UpdateAchievements
		var progressDelta int64 = request.Progress

		if request.Absolute {
			// Get current achievement state
			current, repeatCurrent, err := achievementsSystem.GetAchievements(ctx, logger, nk, userID)
			if err != nil {
				logger.Error("Error getting current achievement state: %v", err)
				return "", err
			}

			// Find the current count
			var currentCount int64 = 0
			if ach, ok := current[request.AchievementId]; ok {
				currentCount = ach.Count
			} else if ach, ok := repeatCurrent[request.AchievementId]; ok {
				currentCount = ach.Count
			}

			// Calculate the delta to reach the absolute value
			progressDelta = request.Progress - currentCount

			// If delta is negative or zero, we don't need to update
			if progressDelta <= 0 {
				logger.Info("No progress update needed for achievement %s (current: %d, requested: %d)",
					request.AchievementId, currentCount, request.Progress)

				// Return current state without updating
				var achievement *Achievement
				if ach, ok := current[request.AchievementId]; ok {
					achievement = ach
				} else if ach, ok := repeatCurrent[request.AchievementId]; ok {
					achievement = ach
				} else {
					logger.Error("Achievement not found: %s", request.AchievementId)
					return "", ErrSystemNotAvailable
				}

				response := struct {
					Achievement *Achievement `json:"achievement"`
					Reward      *Reward      `json:"reward,omitempty"`
					IsCompleted bool         `json:"is_completed"`
					IsUpdated   bool         `json:"is_updated"`
				}{
					Achievement: achievement,
					Reward:      nil,
					IsCompleted: achievement.Count >= achievement.MaxCount,
					IsUpdated:   false,
				}

				responseData, err := json.Marshal(response)
				if err != nil {
					logger.Error("Failed to marshal response: %v", err)
					return "", ErrPayloadEncode
				}

				return string(responseData), nil
			}
		}

		// Create a map with just the one achievement to update
		updates := map[string]int64{
			request.AchievementId: progressDelta,
		}

		// Call the achievements system to update progress
		achievements, repeatAchievements, err := achievementsSystem.UpdateAchievements(ctx, logger, nk, userID, updates)
		if err != nil {
			logger.Error("Error updating achievement progress: %v", err)
			return "", err
		}

		// Find the specific achievement
		var achievement *Achievement
		isCompleted := false

		// Look for the achievement in regular achievements
		if ach, ok := achievements[request.AchievementId]; ok {
			achievement = ach
			// Check if achievement is completed based on count and maxCount
			isCompleted = ach.Count >= ach.MaxCount
		} else if ach, ok := repeatAchievements[request.AchievementId]; ok {
			// Look for the achievement in repeat achievements
			achievement = ach
			isCompleted = ach.Count >= ach.MaxCount
		}

		if achievement == nil {
			logger.Error("Achievement not found after update: %s", request.AchievementId)
			return "", ErrSystemNotAvailable
		}

		// Calculate progress percentage for the response
		progressPercent := float64(0)
		if achievement.MaxCount > 0 {
			progressPercent = float64(achievement.Count) / float64(achievement.MaxCount) * 100
			if progressPercent > 100 {
				progressPercent = 100
			}
		}

		// Prepare the response
		response := struct {
			Achievement     *Achievement `json:"achievement"`
			Reward          *Reward      `json:"reward,omitempty"`
			IsCompleted     bool         `json:"is_completed"`
			IsUpdated       bool         `json:"is_updated"`
			ProgressPercent float64      `json:"progress_percent"`
			RemainingCount  int64        `json:"remaining_count"`
		}{
			Achievement:     achievement,
			Reward:          nil, // Rewards are only given when claimed
			IsCompleted:     isCompleted,
			IsUpdated:       true,
			ProgressPercent: progressPercent,
			RemainingCount:  max(0, achievement.MaxCount-achievement.Count),
		}

		// Encode the response
		responseData, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

// Helper function to get maximum of two int64 values
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// rpcAchievementDetails handles the RPC to get detailed information about a specific achievement
func rpcAchievementDetails(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		achievementsSystem := p.GetAchievementsSystem()
		if achievementsSystem == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		var request struct {
			AchievementId string `json:"achievement_id"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal AchievementDetailsRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Validate input
		if request.AchievementId == "" {
			logger.Error("Achievement ID is required")
			return "", runtime.NewError("achievement_id is required", 3) // INVALID_ARGUMENT
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Get all user achievements
		achievements, repeatAchievements, err := achievementsSystem.GetAchievements(ctx, logger, nk, userID)
		if err != nil {
			logger.Error("Error getting achievements: %v", err)
			return "", err
		}

		// Find the specified achievement
		var achievement *Achievement
		var isRepeat bool

		if ach, ok := achievements[request.AchievementId]; ok {
			achievement = ach
			isRepeat = false
		} else if ach, ok := repeatAchievements[request.AchievementId]; ok {
			achievement = ach
			isRepeat = true
		}

		if achievement == nil {
			logger.Error("Achievement not found: %s", request.AchievementId)
			return "", runtime.NewError("achievement not found", 5) // NOT_FOUND
		}

		// Calculate progress percentage
		progressPercent := float64(0)
		if achievement.MaxCount > 0 {
			progressPercent = float64(achievement.Count) / float64(achievement.MaxCount) * 100
			if progressPercent > 100 {
				progressPercent = 100
			}
		}

		// Check if it's completed
		isCompleted := achievement.Count >= achievement.MaxCount
		isClaimed := achievement.ClaimTimeSec > 0

		// Prepare additional metadata
		now := time.Now().Unix()
		timeUntilReset := int64(0)
		if achievement.ResetTimeSec > now {
			timeUntilReset = achievement.ResetTimeSec - now
		}

		timeUntilExpire := int64(0)
		isExpired := false
		if achievement.ExpireTimeSec > 0 {
			if achievement.ExpireTimeSec > now {
				timeUntilExpire = achievement.ExpireTimeSec - now
			} else {
				isExpired = true
			}
		}

		// Prepare the response
		response := struct {
			Achievement     *Achievement               `json:"achievement"`
			IsRepeat        bool                       `json:"is_repeat"`
			IsCompleted     bool                       `json:"is_completed"`
			IsClaimed       bool                       `json:"is_claimed"`
			IsExpired       bool                       `json:"is_expired"`
			ProgressPercent float64                    `json:"progress_percent"`
			RemainingCount  int64                      `json:"remaining_count"`
			SubAchievements map[string]*SubAchievement `json:"sub_achievements,omitempty"`
			TimeUntilReset  int64                      `json:"time_until_reset,omitempty"`
			TimeUntilExpire int64                      `json:"time_until_expire,omitempty"`
		}{
			Achievement:     achievement,
			IsRepeat:        isRepeat,
			IsCompleted:     isCompleted,
			IsClaimed:       isClaimed,
			IsExpired:       isExpired,
			ProgressPercent: progressPercent,
			RemainingCount:  max(0, achievement.MaxCount-achievement.Count),
			SubAchievements: achievement.SubAchievements,
			TimeUntilReset:  timeUntilReset,
			TimeUntilExpire: timeUntilExpire,
		}

		// Encode the response
		responseData, err := json.Marshal(response)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

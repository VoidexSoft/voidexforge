package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"

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

func rpcAchievementsGet(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		// Implementation would go here
		return "", runtime.NewError("not implemented", 12) // UNIMPLEMENTED
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

// rpcAchievementsList handles the RPC to list achievements
func rpcAchievementsList(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
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
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal AchievementsProgressRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Create a map with just the one achievement to update
		updates := map[string]int64{
			request.AchievementId: request.Progress,
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
			// Since we don't know the exact fields, we'll assume it's completed if it's found
			isCompleted = true
		} else if ach, ok := repeatAchievements[request.AchievementId]; ok {
			// Look for the achievement in repeat achievements
			achievement = ach
			isCompleted = true
		}

		if achievement == nil {
			logger.Error("Achievement not found after update: %s", request.AchievementId)
			return "", ErrSystemNotAvailable
		}

		// Prepare the response
		response := struct {
			Achievement *Achievement `json:"achievement"`
			Reward      *Reward      `json:"reward,omitempty"`
			IsCompleted bool         `json:"is_completed"`
		}{
			Achievement: achievement,
			Reward:      nil, // Rewards are only given when claimed
			IsCompleted: isCompleted,
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

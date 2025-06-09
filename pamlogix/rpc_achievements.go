package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/heroiclabs/nakama-common/runtime"
	"google.golang.org/protobuf/proto"
)

// RPC handler function placeholders for achievements
func rpcAchievementsClaim(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		achievementsSystem := p.GetAchievementsSystem()
		if achievementsSystem == nil {
			return "", ErrSystemNotFound
		}

		request := &AchievementsClaimRequest{}
		if err := proto.Unmarshal([]byte(payload), request); err != nil {
			logger.Error("Failed to unmarshal AchievementsClaimRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Validate request
		if len(request.Ids) == 0 {
			logger.Error("No achievement IDs provided in request")
			return "", runtime.NewError("ids is required and cannot be empty", INVALID_ARGUMENT_ERROR_CODE)
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Call the achievements system to claim achievements
		achievements, repeatAchievements, err := achievementsSystem.ClaimAchievements(ctx, logger, nk, userID, request.Ids, request.ClaimTotalReward)
		if err != nil {
			logger.Error("Error claiming achievements: %v", err)
			return "", err
		}

		// Prepare the response using protobuf
		response := &AchievementsUpdateAck{
			Achievements:       achievements,
			RepeatAchievements: repeatAchievements,
		}

		responseData, err := proto.Marshal(response)
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

		// Prepare the response using protobuf
		response := &AchievementList{
			Achievements:       achievements,
			RepeatAchievements: repeatAchievements,
		}

		// Encode the response using protobuf
		responseData, err := proto.Marshal(response)
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

		// Parse the input request using protobuf
		request := &AchievementsUpdateRequest{}
		if err := proto.Unmarshal([]byte(payload), request); err != nil {
			logger.Error("Failed to unmarshal AchievementsUpdateRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Convert protobuf request to the expected format for the achievements system
		var updates map[string]int64
		if request.Achievements != nil && len(request.Achievements) > 0 {
			updates = request.Achievements
		} else if len(request.Ids) > 0 && request.Amount > 0 {
			// Support legacy format where all IDs get the same amount
			updates = make(map[string]int64)
			for _, id := range request.Ids {
				updates[id] = request.Amount
			}
		} else {
			logger.Error("No achievement updates provided in request")
			return "", runtime.NewError("achievements or ids with amount is required", INVALID_ARGUMENT_ERROR_CODE)
		}

		// Call the achievements system to update progress
		achievements, repeatAchievements, err := achievementsSystem.UpdateAchievements(ctx, logger, nk, userID, updates)
		if err != nil {
			logger.Error("Error updating achievement progress: %v", err)
			return "", err
		}

		// Prepare the response using protobuf
		response := &AchievementsUpdateAck{
			Achievements:       achievements,
			RepeatAchievements: repeatAchievements,
		}

		// Encode the response using protobuf
		responseData, err := proto.Marshal(response)
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

		// For this endpoint, we'll continue using JSON for the request since there's no specific protobuf message
		// But we'll use protobuf for the response
		var request struct {
			Category     string   `json:"category,omitempty"`
			Categories   []string `json:"categories,omitempty"`
			OnlyComplete bool     `json:"only_complete,omitempty"`
			OnlyProgress bool     `json:"only_progress,omitempty"`
		}

		// Try to parse optional filters, but continue even if parsing fails
		if payload != "" {
			// Since this is a custom endpoint without a defined protobuf message,
			// we'll keep JSON parsing for backward compatibility
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

		// Prepare the response using protobuf
		response := &AchievementList{
			Achievements:       filteredAchievements,
			RepeatAchievements: filteredRepeatAchievements,
		}

		// Encode the response using protobuf
		responseData, err := proto.Marshal(response)
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

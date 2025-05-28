package pamlogix

import (
	"context"
	"encoding/json"

	"github.com/heroiclabs/nakama-common/runtime"
)

// rpcEventLeaderboardsList handles the list event leaderboards RPC
func rpcEventLeaderboardsList(pamlogix Pamlogix) func(context.Context, runtime.Logger, runtime.NakamaModule, string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, payload string) (string, error) {
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok {
			return "", ErrNoSessionUser
		}

		eventLeaderboardsSystem := pamlogix.GetEventLeaderboardsSystem()
		if eventLeaderboardsSystem == nil {
			return "", ErrSystemNotAvailable
		}

		// Parse request
		var req EventLeaderboardList
		if payload != "" {
			if err := json.Unmarshal([]byte(payload), &req); err != nil {
				logger.Error("Failed to unmarshal event leaderboard list request: %v", err)
				return "", ErrPayloadDecode
			}
		}

		// List event leaderboards
		eventLeaderboards, err := eventLeaderboardsSystem.ListEventLeaderboard(ctx, logger, nk, userID, req.WithScores, req.Categories)
		if err != nil {
			logger.Error("Failed to list event leaderboards: %v", err)
			return "", err
		}

		// Build response
		resp := &EventLeaderboards{
			EventLeaderboards: eventLeaderboards,
		}

		respBytes, err := json.Marshal(resp)
		if err != nil {
			logger.Error("Failed to marshal event leaderboards response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(respBytes), nil
	}
}

// rpcEventLeaderboardsGet handles the get event leaderboard RPC
func rpcEventLeaderboardsGet(pamlogix Pamlogix) func(context.Context, runtime.Logger, runtime.NakamaModule, string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, payload string) (string, error) {
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok {
			return "", ErrNoSessionUser
		}

		eventLeaderboardsSystem := pamlogix.GetEventLeaderboardsSystem()
		if eventLeaderboardsSystem == nil {
			return "", ErrSystemNotAvailable
		}

		// Parse request
		var req EventLeaderboardGet
		if err := json.Unmarshal([]byte(payload), &req); err != nil {
			logger.Error("Failed to unmarshal event leaderboard get request: %v", err)
			return "", ErrPayloadDecode
		}

		if req.Id == "" {
			return "", ErrBadInput
		}

		// Get event leaderboard
		eventLeaderboard, err := eventLeaderboardsSystem.GetEventLeaderboard(ctx, logger, nk, userID, req.Id)
		if err != nil {
			logger.Error("Failed to get event leaderboard: %v", err)
			return "", err
		}

		respBytes, err := json.Marshal(eventLeaderboard)
		if err != nil {
			logger.Error("Failed to marshal event leaderboard response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(respBytes), nil
	}
}

// rpcEventLeaderboardsUpdate handles the update event leaderboard RPC
func rpcEventLeaderboardsUpdate(pamlogix Pamlogix) func(context.Context, runtime.Logger, runtime.NakamaModule, string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, payload string) (string, error) {
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok {
			return "", ErrNoSessionUser
		}

		username, ok := ctx.Value(runtime.RUNTIME_CTX_USERNAME).(string)
		if !ok {
			return "", ErrNoSessionUsername
		}

		eventLeaderboardsSystem := pamlogix.GetEventLeaderboardsSystem()
		if eventLeaderboardsSystem == nil {
			return "", ErrSystemNotAvailable
		}

		// Parse request
		var req EventLeaderboardUpdate
		if err := json.Unmarshal([]byte(payload), &req); err != nil {
			logger.Error("Failed to unmarshal event leaderboard update request: %v", err)
			return "", ErrPayloadDecode
		}

		if req.Id == "" {
			return "", ErrBadInput
		}

		// Parse metadata from JSON string to map
		var metadata map[string]interface{}
		if req.Metadata != "" {
			if err := json.Unmarshal([]byte(req.Metadata), &metadata); err != nil {
				logger.Error("Failed to unmarshal metadata: %v", err)
				return "", ErrPayloadDecode
			}
		}

		// Update event leaderboard
		eventLeaderboard, err := eventLeaderboardsSystem.UpdateEventLeaderboard(ctx, logger, nk, userID, username, req.Id, req.Score, req.Subscore, metadata)
		if err != nil {
			logger.Error("Failed to update event leaderboard: %v", err)
			return "", err
		}

		respBytes, err := json.Marshal(eventLeaderboard)
		if err != nil {
			logger.Error("Failed to marshal event leaderboard response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(respBytes), nil
	}
}

// rpcEventLeaderboardsClaim handles the claim event leaderboard RPC
func rpcEventLeaderboardsClaim(pamlogix Pamlogix) func(context.Context, runtime.Logger, runtime.NakamaModule, string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, payload string) (string, error) {
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok {
			return "", ErrNoSessionUser
		}

		eventLeaderboardsSystem := pamlogix.GetEventLeaderboardsSystem()
		if eventLeaderboardsSystem == nil {
			return "", ErrSystemNotAvailable
		}

		// Parse request
		var req EventLeaderboardClaim
		if err := json.Unmarshal([]byte(payload), &req); err != nil {
			logger.Error("Failed to unmarshal event leaderboard claim request: %v", err)
			return "", ErrPayloadDecode
		}

		if req.Id == "" {
			return "", ErrBadInput
		}

		// Claim event leaderboard
		eventLeaderboard, err := eventLeaderboardsSystem.ClaimEventLeaderboard(ctx, logger, nk, userID, req.Id)
		if err != nil {
			logger.Error("Failed to claim event leaderboard: %v", err)
			return "", err
		}

		respBytes, err := json.Marshal(eventLeaderboard)
		if err != nil {
			logger.Error("Failed to marshal event leaderboard response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(respBytes), nil
	}
}

// rpcEventLeaderboardsRoll handles the roll event leaderboard RPC
func rpcEventLeaderboardsRoll(pamlogix Pamlogix) func(context.Context, runtime.Logger, runtime.NakamaModule, string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, payload string) (string, error) {
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok {
			return "", ErrNoSessionUser
		}

		eventLeaderboardsSystem := pamlogix.GetEventLeaderboardsSystem()
		if eventLeaderboardsSystem == nil {
			return "", ErrSystemNotAvailable
		}

		// Parse request
		var req EventLeaderboardRoll
		if err := json.Unmarshal([]byte(payload), &req); err != nil {
			logger.Error("Failed to unmarshal event leaderboard roll request: %v", err)
			return "", ErrPayloadDecode
		}

		if req.Id == "" {
			return "", ErrBadInput
		}

		// Roll event leaderboard
		eventLeaderboard, err := eventLeaderboardsSystem.RollEventLeaderboard(ctx, logger, nk, userID, req.Id, nil, nil)
		if err != nil {
			logger.Error("Failed to roll event leaderboard: %v", err)
			return "", err
		}

		respBytes, err := json.Marshal(eventLeaderboard)
		if err != nil {
			logger.Error("Failed to marshal event leaderboard response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(respBytes), nil
	}
}

// rpcEventLeaderboardsDebugFill handles the debug fill event leaderboard RPC
func rpcEventLeaderboardsDebugFill(pamlogix Pamlogix) func(context.Context, runtime.Logger, runtime.NakamaModule, string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, payload string) (string, error) {
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok {
			return "", ErrNoSessionUser
		}

		eventLeaderboardsSystem := pamlogix.GetEventLeaderboardsSystem()
		if eventLeaderboardsSystem == nil {
			return "", ErrSystemNotAvailable
		}

		// Parse request
		var req EventLeaderboardDebugFillRequest
		if err := json.Unmarshal([]byte(payload), &req); err != nil {
			logger.Error("Failed to unmarshal event leaderboard debug fill request: %v", err)
			return "", ErrPayloadDecode
		}

		if req.Id == "" {
			return "", ErrBadInput
		}

		// Debug fill event leaderboard
		eventLeaderboard, err := eventLeaderboardsSystem.DebugFill(ctx, logger, nk, userID, req.Id, int(req.TargetCount))
		if err != nil {
			logger.Error("Failed to debug fill event leaderboard: %v", err)
			return "", err
		}

		respBytes, err := json.Marshal(eventLeaderboard)
		if err != nil {
			logger.Error("Failed to marshal event leaderboard response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(respBytes), nil
	}
}

// rpcEventLeaderboardsDebugRandomScores handles the debug random scores event leaderboard RPC
func rpcEventLeaderboardsDebugRandomScores(pamlogix Pamlogix) func(context.Context, runtime.Logger, runtime.NakamaModule, string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, payload string) (string, error) {
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok {
			return "", ErrNoSessionUser
		}

		eventLeaderboardsSystem := pamlogix.GetEventLeaderboardsSystem()
		if eventLeaderboardsSystem == nil {
			return "", ErrSystemNotAvailable
		}

		// Parse request
		var req EventLeaderboardDebugRandomScoresRequest
		if err := json.Unmarshal([]byte(payload), &req); err != nil {
			logger.Error("Failed to unmarshal event leaderboard debug random scores request: %v", err)
			return "", ErrPayloadDecode
		}

		if req.Id == "" {
			return "", ErrBadInput
		}

		var operator *int
		if req.Operator != nil && req.Operator.Value != 0 {
			operatorValue := int(req.Operator.Value)
			operator = &operatorValue
		}

		// Debug random scores event leaderboard
		eventLeaderboard, err := eventLeaderboardsSystem.DebugRandomScores(ctx, logger, nk, userID, req.Id, req.Min, req.Max, req.SubscoreMin, req.SubscoreMax, operator)
		if err != nil {
			logger.Error("Failed to debug random scores event leaderboard: %v", err)
			return "", err
		}

		respBytes, err := json.Marshal(eventLeaderboard)
		if err != nil {
			logger.Error("Failed to marshal event leaderboard response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(respBytes), nil
	}
}

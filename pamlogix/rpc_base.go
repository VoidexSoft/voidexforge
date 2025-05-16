package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/heroiclabs/nakama-common/runtime"
)

// rpcBaseRateApp handles the RPC to submit app ratings
func rpcBaseRateApp(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		baseSystem := p.GetBaseSystem()
		if baseSystem == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		var request struct {
			Score   uint32 `json:"score"`
			Message string `json:"message,omitempty"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal RateAppRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID and username from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		username, ok := ctx.Value(runtime.RUNTIME_CTX_USERNAME).(string)
		if !ok || username == "" {
			logger.Error("No username in context")
			return "", ErrNoSessionUsername
		}

		// Call the base system to process the rating
		err := baseSystem.RateApp(ctx, logger, nk, userID, username, request.Score, request.Message)
		if err != nil {
			logger.Error("Error processing app rating: %v", err)
			return "", err
		}

		// Prepare the response
		response := struct {
			Success bool `json:"success"`
		}{
			Success: true,
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

// rpcBaseSetDevicePrefs handles the RPC to set device preferences
func rpcBaseSetDevicePrefs(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		baseSystem := p.GetBaseSystem()
		if baseSystem == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		var request struct {
			DeviceID         string          `json:"device_id"`
			PushTokenAndroid string          `json:"push_token_android,omitempty"`
			PushTokenIos     string          `json:"push_token_ios,omitempty"`
			Preferences      map[string]bool `json:"preferences,omitempty"`
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal SetDevicePrefsRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Call the base system to set device preferences
		err := baseSystem.SetDevicePrefs(ctx, logger, db, userID, request.DeviceID, request.PushTokenAndroid, request.PushTokenIos, request.Preferences)
		if err != nil {
			logger.Error("Error setting device preferences: %v", err)
			return "", err
		}

		// Prepare the response
		response := struct {
			Success bool `json:"success"`
		}{
			Success: true,
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

// rpcBaseSync handles the RPC to sync server with offline state changes
func rpcBaseSync(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		baseSystem := p.GetBaseSystem()
		if baseSystem == nil {
			return "", ErrSystemNotFound
		}

		// Parse the input request
		var request SyncRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal SyncRequest: %v", err)
			return "", ErrPayloadDecode
		}

		// Extract user ID from session
		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			logger.Error("No user ID in context")
			return "", ErrNoSessionUser
		}

		// Call the base system to sync
		resp, err := baseSystem.Sync(ctx, logger, nk, userID, &request)
		if err != nil {
			logger.Error("Error syncing: %v", err)
			return "", err
		}

		// Encode the response
		responseData, err := json.Marshal(resp)
		if err != nil {
			logger.Error("Failed to marshal response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

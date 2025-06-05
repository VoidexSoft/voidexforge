package pamlogix

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
)

const (
	statsStorageCollection = "stats"
	userStatsStorageKey    = "user_stats"
)

// NakamaStatsSystem implements the StatsSystem interface using Nakama as the backend.
type NakamaStatsSystem struct {
	config   *StatsConfig
	pamlogix Pamlogix
}

// NewNakamaStatsSystem creates a new instance of the stats system with the given configuration.
func NewStatsSystem(config *StatsConfig) *NakamaStatsSystem {
	return &NakamaStatsSystem{
		config: config,
	}
}

// SetPamlogix sets the Pamlogix instance for this stats system
func (s *NakamaStatsSystem) SetPamlogix(pl Pamlogix) {
	s.pamlogix = pl
}

// GetType returns the system type for the stats system.
func (s *NakamaStatsSystem) GetType() SystemType {
	return SystemTypeStats
}

// GetConfig returns the configuration for the stats system.
func (s *NakamaStatsSystem) GetConfig() any {
	return s.config
}

// validateStatName checks if a stat name is allowed based on the whitelist configuration
func (s *NakamaStatsSystem) validateStatName(name string, isPublic bool) error {
	// If whitelist is configured and not empty, enforce it
	if s.config != nil && s.config.Whitelist != nil && len(s.config.Whitelist) > 0 {
		for _, allowed := range s.config.Whitelist {
			if allowed == name {
				return nil // Found in whitelist
			}
		}
		return runtime.NewError(fmt.Sprintf("stat '%s' is not in the configured whitelist", name), 3) // INVALID_ARGUMENT
	}
	return nil // No whitelist configured, allow all stats
}

// List all private stats for one or more users.
func (s *NakamaStatsSystem) List(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, userIDs []string) (map[string]*StatList, error) {
	result := make(map[string]*StatList, len(userIDs))

	// Early return for empty user list
	if len(userIDs) == 0 {
		return result, nil
	}

	// Create batch storage read operations for all users
	storageReads := make([]*runtime.StorageRead, len(userIDs))
	for i, uid := range userIDs {
		storageReads[i] = &runtime.StorageRead{
			Collection: statsStorageCollection,
			Key:        userStatsStorageKey,
			UserID:     uid,
		}
	}

	// Perform single batched storage read
	objects, err := nk.StorageRead(ctx, storageReads)
	if err != nil {
		logger.Error("Failed to batch read user stats: %v", err)
		return nil, err
	}

	// Create a map for efficient lookup of results by userID
	objectsByUserID := make(map[string]string)
	for _, obj := range objects {
		if obj != nil && obj.Value != "" {
			objectsByUserID[obj.UserId] = obj.Value
		}
	}

	// Process results for each requested user
	for _, uid := range userIDs {
		var stats *StatList

		if data, exists := objectsByUserID[uid]; exists {
			// Parse existing stats data
			var parsedStats StatList
			if err := json.Unmarshal([]byte(data), &parsedStats); err != nil {
				logger.Error("Failed to unmarshal user stats for user %s: %v", uid, err)
				return nil, err
			}
			stats = &parsedStats
		} else {
			// No stats found for this user, create empty stats
			stats = &StatList{
				Public:  make(map[string]*Stat),
				Private: make(map[string]*Stat),
			}
		}

		result[uid] = stats
	}

	return result, nil
}

// Update private stats for a particular user.
func (s *NakamaStatsSystem) Update(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, publicStats []*StatUpdate, privateStats []*StatUpdate) (*StatList, error) {
	stats, err := s.getUserStats(ctx, logger, nk, userID)
	if err != nil {
		logger.Error("Failed to get stats for user %s: %v", userID, err)
		return nil, err
	}
	if stats == nil {
		stats = &StatList{
			Public:  make(map[string]*Stat),
			Private: make(map[string]*Stat),
		}
	}
	now := time.Now().Unix()
	// Helper to apply a StatUpdate to a Stat
	applyUpdate := func(stat *Stat, upd *StatUpdate) {
		oldValue := stat.Value
		switch upd.Operator {
		case StatUpdateOperator_STAT_UPDATE_OPERATOR_SET:
			stat.Value = upd.Value
		case StatUpdateOperator_STAT_UPDATE_OPERATOR_DELTA:
			stat.Value += upd.Value
		case StatUpdateOperator_STAT_UPDATE_OPERATOR_MIN:
			if stat.Count == 0 || upd.Value < stat.Min {
				stat.Value = upd.Value
			}
		case StatUpdateOperator_STAT_UPDATE_OPERATOR_MAX:
			if stat.Count == 0 || upd.Value > stat.Max {
				stat.Value = upd.Value
			}
		default:
			stat.Value = upd.Value
		}
		stat.Count++
		// For delta operations, add the delta to total. For other operations, add the applied value delta.
		if upd.Operator == StatUpdateOperator_STAT_UPDATE_OPERATOR_DELTA {
			stat.Total += upd.Value
		} else {
			stat.Total += (stat.Value - oldValue)
		}
		if stat.Count == 1 {
			stat.Min = stat.Value
			stat.Max = stat.Value
			stat.First = upd.Value
		} else {
			// Update min/max based on the update value, not the final stat.Value
			if upd.Value < stat.Min {
				stat.Min = upd.Value
			}
			if upd.Value > stat.Max {
				stat.Max = upd.Value
			}
		}
		stat.Last = upd.Value
		stat.UpdateTimeSec = now
	}
	// Apply public stat updates
	for _, upd := range publicStats {
		// Validate stat name against whitelist
		if err := s.validateStatName(upd.Name, true); err != nil {
			logger.Error("Failed to validate public stat name '%s': %v", upd.Name, err)
			return nil, err
		}

		stat, ok := stats.Public[upd.Name]
		if !ok {
			stat = &Stat{Name: upd.Name, Public: true}
			stats.Public[upd.Name] = stat
		}
		applyUpdate(stat, upd)
	}
	// Apply private stat updates
	for _, upd := range privateStats {
		// Validate stat name against whitelist
		if err := s.validateStatName(upd.Name, false); err != nil {
			logger.Error("Failed to validate private stat name '%s': %v", upd.Name, err)
			return nil, err
		}

		stat, ok := stats.Private[upd.Name]
		if !ok {
			stat = &Stat{Name: upd.Name, Public: false}
			stats.Private[upd.Name] = stat
		}
		applyUpdate(stat, upd)
	}
	if err := s.saveUserStats(ctx, logger, nk, userID, stats); err != nil {
		logger.Error("Failed to save stats for user %s: %v", userID, err)
		return nil, err
	}
	return stats, nil
}

// Helper: getUserStats fetches the stored stats data for a user from Nakama storage.
func (s *NakamaStatsSystem) getUserStats(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (*StatList, error) {
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: statsStorageCollection,
			Key:        userStatsStorageKey,
			UserID:     userID,
		},
	})
	if err != nil {
		logger.Error("Failed to read user stats: %v", err)
		return nil, err
	}
	if len(objects) == 0 || objects[0] == nil || objects[0].Value == "" {
		return nil, nil // No stats found
	}
	var stats StatList
	if err := json.Unmarshal([]byte(objects[0].Value), &stats); err != nil {
		logger.Error("Failed to unmarshal user stats: %v", err)
		return nil, err
	}
	return &stats, nil
}

// Helper: saveUserStats stores the updated stats data for a user in Nakama storage.
func (s *NakamaStatsSystem) saveUserStats(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, stats *StatList) error {
	data, err := json.Marshal(stats)
	if err != nil {
		logger.Error("Failed to marshal user stats: %v", err)
		return err
	}
	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{
		{
			Collection:      statsStorageCollection,
			Key:             userStatsStorageKey,
			UserID:          userID,
			Value:           string(data),
			PermissionRead:  runtime.STORAGE_PERMISSION_OWNER_READ,
			PermissionWrite: runtime.STORAGE_PERMISSION_OWNER_WRITE,
		},
	})
	if err != nil {
		logger.Error("Failed to write user stats: %v", err)
		return err
	}
	return nil
}

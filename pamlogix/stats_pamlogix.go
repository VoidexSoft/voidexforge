package pamlogix

import (
	"context"
	"encoding/json"
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
func NewNakamaStatsSystem(config *StatsConfig) *NakamaStatsSystem {
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

// List all private stats for one or more users.
func (s *NakamaStatsSystem) List(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, userIDs []string) (map[string]*StatList, error) {
	result := make(map[string]*StatList, len(userIDs))
	for _, uid := range userIDs {
		stats, err := s.getUserStats(ctx, logger, nk, uid)
		if err != nil {
			logger.Error("Failed to get stats for user %s: %v", uid, err)
			return nil, err
		}
		if stats == nil {
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
		switch upd.Operator {
		case StatUpdateOperator_STAT_UPDATE_OPERATOR_SET:
			stat.Value = upd.Value
		case StatUpdateOperator_STAT_UPDATE_OPERATOR_DELTA:
			stat.Value += upd.Value
		case StatUpdateOperator_STAT_UPDATE_OPERATOR_MIN:
			if stat.Count == 0 || upd.Value < stat.Value {
				stat.Value = upd.Value
			}
		case StatUpdateOperator_STAT_UPDATE_OPERATOR_MAX:
			if stat.Count == 0 || upd.Value > stat.Value {
				stat.Value = upd.Value
			}
		default:
			stat.Value = upd.Value
		}
		stat.Count++
		stat.Total += upd.Value
		if stat.Count == 1 {
			stat.Min = stat.Value
			stat.Max = stat.Value
			stat.First = upd.Value
		} else {
			if stat.Value < stat.Min {
				stat.Min = stat.Value
			}
			if stat.Value > stat.Max {
				stat.Max = stat.Value
			}
		}
		stat.Last = upd.Value
		stat.UpdateTimeSec = now
	}
	// Apply public stat updates
	for _, upd := range publicStats {
		stat, ok := stats.Public[upd.Name]
		if !ok {
			stat = &Stat{Name: upd.Name, Public: true}
			stats.Public[upd.Name] = stat
		}
		applyUpdate(stat, upd)
	}
	// Apply private stat updates
	for _, upd := range privateStats {
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

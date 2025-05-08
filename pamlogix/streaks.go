package pamlogix

import (
	"context"

	"github.com/heroiclabs/nakama-common/runtime"
)

var ErrStreakResetInvalid = runtime.NewError("streak reset schedule invalid", 13)

// StreaksConfig is the data definition for a StreaksSystem type.
type StreaksConfig struct {
	Streaks map[string]*StreaksConfigStreak `json:"streaks,omitempty"`
}

type StreaksConfigStreak struct {
	Name                 string                       `json:"name,omitempty"`
	Description          string                       `json:"description,omitempty"`
	Count                int64                        `json:"count,omitempty"`
	MaxCount             int64                        `json:"max_count,omitempty"`
	MaxCountCurrentReset int64                        `json:"max_count_current_reset,omitempty"`
	IdleCountDecayReset  int64                        `json:"idle_count_decay_reset,omitempty"`
	MaxIdleCountDecay    int64                        `json:"max_idle_count_decay,omitempty"`
	ResetCronexpr        string                       `json:"reset_cronexpr,omitempty"`
	Rewards              []*StreaksConfigStreakReward `json:"rewards,omitempty"`
	StartTimeSec         int64                        `json:"start_time_sec,omitempty"`
	EndTimeSec           int64                        `json:"end_time_sec,omitempty"`
	Disabled             bool                         `json:"disabled,omitempty"`
}

type StreaksConfigStreakReward struct {
	CountMin int64                `json:"count_min,omitempty"`
	CountMax int64                `json:"count_max,omitempty"`
	Reward   *EconomyConfigReward `json:"reward,omitempty"`
}

type StreaksSystem interface {
	System

	// List all streaks and their current state and progress for a given user.
	List(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (streaks map[string]*Streak, err error)

	// Update one or more streaks with the indicated counts for the given user.
	Update(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, streakIDs map[string]int64) (streaks map[string]*Streak, err error)

	// Claim rewards for one or more streaks for the given user.
	Claim(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, streakIDs []string) (streaks map[string]*Streak, err error)

	// Reset progress on selected streaks for the given user.
	Reset(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, streakIDs []string) (streaks map[string]*Streak, err error)

	// SetOnClaimReward sets a custom reward function which will run after a streak's reward is rolled.
	SetOnClaimReward(fn OnReward[*StreaksConfigStreak])
}

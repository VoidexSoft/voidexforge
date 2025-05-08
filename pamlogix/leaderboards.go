package pamlogix

import (
	"context"
	"database/sql"

	"github.com/heroiclabs/nakama-common/api"
	"github.com/heroiclabs/nakama-common/runtime"
)

// LeaderboardsConfig is the data definition for the LeaderboardsSystem type.
type LeaderboardsConfig struct {
	Leaderboards []*LeaderboardsConfigLeaderboard `json:"leaderboards,omitempty"`
}

type LeaderboardsConfigLeaderboard struct {
	Id            string   `json:"id,omitempty"`
	SortOrder     string   `json:"sort_order,omitempty"`
	Operator      string   `json:"operator,omitempty"`
	ResetSchedule string   `json:"reset_schedule,omitempty"`
	Authoritative bool     `json:"authoritative,omitempty"`
	Regions       []string `json:"regions,omitempty"`
}

// The LeaderboardsSystem defines a collection of leaderboards which can be defined as global or regional with Nakama
// server.
type LeaderboardsSystem interface {
	System
}

// ValidateWriteScoreFn is a function used to validate the leaderboard score input.
type ValidateWriteScoreFn func(context.Context, runtime.Logger, *sql.DB, runtime.NakamaModule, *api.WriteLeaderboardRecordRequest) *runtime.Error

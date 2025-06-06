package pamlogix

import (
	"context"
	"database/sql"

	"github.com/heroiclabs/nakama-common/runtime"
)

// TeamsConfig is the data definition for a TeamsSystem type.
type TeamsConfig struct {
	MaxTeamSize int `json:"max_team_size,omitempty"`
}

// A TeamsSystem is a gameplay system which wraps the groups system in Nakama server.
type TeamsSystem interface {
	System

	// Create makes a new team (i.e. Nakama group) with additional metadata which configures the team.
	Create(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, req *TeamCreateRequest) (team *Team, err error)

	// List will return a list of teams which the user can join.
	List(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, req *TeamListRequest) (teams *TeamList, err error)

	// Search for teams based on given criteria.
	Search(ctx context.Context, db *sql.DB, logger runtime.Logger, nk runtime.NakamaModule, req *TeamSearchRequest) (teams *TeamList, err error)

	// WriteChatMessage sends a message to the user's team even when they're not connected on a realtime socket.
	WriteChatMessage(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, req *TeamWriteChatMessageRequest) (resp *ChannelMessageAck, err error)
}

// ValidateCreateTeamFn allows custom rules or velocity checks to be added as a precondition on whether a team is
// created or not.
type ValidateCreateTeamFn func(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, req *TeamCreateRequest) *runtime.Error

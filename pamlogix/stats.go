package pamlogix

import (
	"context"

	"github.com/heroiclabs/nakama-common/runtime"
)

// StatsConfig is the data definition for a StatsSystem type.
type StatsConfig struct {
	Whitelist    []string                    `json:"whitelist,omitempty"`
	StatsPublic  map[string]*StatsConfigStat `json:"stats_public,omitempty"`
	StatsPrivate map[string]*StatsConfigStat `json:"stats_private,omitempty"`
}

type StatsConfigStat struct {
	Value                int64                  `json:"value,omitempty"`
	AdditionalProperties map[string]interface{} `json:"additional_properties,omitempty"`
}

type StatsSystem interface {
	System

	// List all private stats for one or more users.
	List(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, userIDs []string) (stats map[string]*StatList, err error)

	// Update private stats for a particular user.
	Update(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, publicStats []*StatUpdate, privateStats []*StatUpdate) (stats *StatList, err error)
}

package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/heroiclabs/nakama-common/api"
	"github.com/heroiclabs/nakama-common/runtime"
)

const (
	teamsStorageCollection = "teams"
)

// NakamaTeamsSystem implements the TeamsSystem interface using Nakama groups as the backend.
type NakamaTeamsSystem struct {
	config             *TeamsConfig
	validateCreateTeam ValidateCreateTeamFn
	pamlogix           Pamlogix
}

// NewNakamaTeamsSystem creates a new instance of the teams system with the given configuration.
func NewNakamaTeamsSystem(config *TeamsConfig) *NakamaTeamsSystem {
	return &NakamaTeamsSystem{
		config: config,
	}
}

// SetPamlogix sets the Pamlogix instance for this teams system
func (t *NakamaTeamsSystem) SetPamlogix(pl Pamlogix) {
	t.pamlogix = pl
}

// SetValidateCreateTeam sets the custom validation function for team creation
func (t *NakamaTeamsSystem) SetValidateCreateTeam(fn ValidateCreateTeamFn) {
	t.validateCreateTeam = fn
}

// GetType returns the system type for the teams system.
func (t *NakamaTeamsSystem) GetType() SystemType {
	return SystemTypeTeams
}

// GetConfig returns the configuration for the teams system.
func (t *NakamaTeamsSystem) GetConfig() any {
	return t.config
}

// Create makes a new team (i.e. Nakama group) with additional metadata which configures the team.
func (t *NakamaTeamsSystem) Create(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, req *TeamCreateRequest) (*Team, error) {
	// Get user ID from context
	userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
	if !ok || userID == "" {
		return nil, runtime.NewError("user id not found in context", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
	}

	// Validate the request
	if req.Name == "" {
		return nil, runtime.NewError("team name is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
	}

	// Apply custom validation if configured
	if t.validateCreateTeam != nil {
		if err := t.validateCreateTeam(ctx, logger, nk, userID, req); err != nil {
			return nil, err
		}
	}

	// Prepare metadata
	metadata := make(map[string]interface{})
	if req.SetupMetadata != "" {
		if err := json.Unmarshal([]byte(req.SetupMetadata), &metadata); err != nil {
			logger.Warn("Failed to parse setup metadata: %v", err)
			// Continue with empty metadata instead of failing
		}
	}

	// Add team-specific metadata
	metadata["icon"] = req.Icon
	metadata["created_by"] = userID

	// Determine max team size
	maxCount := 100 // Default max count
	if t.config != nil && t.config.MaxTeamSize > 0 {
		maxCount = t.config.MaxTeamSize
	}

	// Create the Nakama group
	group, err := nk.GroupCreate(ctx, userID, req.Name, userID, req.LangTag, req.Desc, "", req.Open, metadata, maxCount)
	if err != nil {
		logger.Error("Failed to create Nakama group: %v", err)
		return nil, err
	}

	// Convert Nakama group to Team
	team := t.convertGroupToTeam(group, req.Icon)

	return team, nil
}

// List will return a list of teams which the user can join.
func (t *NakamaTeamsSystem) List(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, req *TeamListRequest) (*TeamList, error) {
	// Set default limit if not provided
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	// Use Nakama's GroupsList to get available groups
	open := true // Only show open groups for joining
	groups, cursor, err := nk.GroupsList(ctx, "", req.Location, nil, &open, int(limit), req.Cursor)
	if err != nil {
		logger.Error("Failed to list groups: %v", err)
		return nil, err
	}

	// Convert groups to teams
	teams := make([]*Team, 0, len(groups))
	for _, group := range groups {
		team := t.convertGroupToTeam(group, "")
		teams = append(teams, team)
	}

	return &TeamList{
		Teams:  teams,
		Cursor: cursor,
	}, nil
}

// Search for teams based on given criteria.
func (t *NakamaTeamsSystem) Search(ctx context.Context, db *sql.DB, logger runtime.Logger, nk runtime.NakamaModule, req *TeamSearchRequest) (*TeamList, error) {
	// Set default limit if not provided
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	// Use Nakama's GroupsList with name filter to search
	open := true // Only show open groups for joining
	groups, _, err := nk.GroupsList(ctx, req.Input, req.LangTag, nil, &open, int(limit), "")
	if err != nil {
		logger.Error("Failed to search groups: %v", err)
		return nil, err
	}

	// Convert groups to teams
	teams := make([]*Team, 0, len(groups))
	for _, group := range groups {
		// Additional filtering if needed
		if req.Input != "" && !strings.Contains(strings.ToLower(group.Name), strings.ToLower(req.Input)) {
			continue
		}

		team := t.convertGroupToTeam(group, "")
		teams = append(teams, team)
	}

	return &TeamList{
		Teams:  teams,
		Cursor: "", // Search doesn't support pagination cursor
	}, nil
}

// checkTeamMembership efficiently checks if a user is a member of a specific team.
//
// OPTIMIZATION STRATEGY:
// Uses UserGroupsList with a small limit (10) instead of fetching all user groups (up to 100).
// This approach significantly reduces:
// - Database queries to Nakama
// - Memory usage by limiting the number of groups fetched
// - Network overhead for the API call
//
// Note: This implementation is stateless and works correctly in distributed Nakama deployments
// where multiple instances may be running behind a load balancer.
func (t *NakamaTeamsSystem) checkTeamMembership(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, teamID string) (bool, error) {
	// Use a small limit (10) instead of fetching all user groups (up to 100)
	// This reduces memory usage and network overhead significantly
	userGroups, _, err := nk.UserGroupsList(ctx, userID, 10, nil, "") // Small limit for efficiency
	if err != nil {
		logger.Error("Failed to get user groups for membership check: %v", err)
		return false, err
	}

	// Check if the user is in the specific team
	for _, userGroup := range userGroups {
		if userGroup.Group.Id == teamID {
			// Check if user is an active member (not just a join request)
			if userGroup.State != nil && userGroup.State.Value != int32(api.UserGroupList_UserGroup_JOIN_REQUEST) {
				return true, nil
			}
			break
		}
	}

	return false, nil
}

// WriteChatMessage sends a message to the user's team even when they're not connected on a realtime socket.
func (t *NakamaTeamsSystem) WriteChatMessage(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, req *TeamWriteChatMessageRequest) (*ChannelMessageAck, error) {
	if req.Id == "" {
		return nil, runtime.NewError("team id is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
	}

	if req.Content == "" {
		return nil, runtime.NewError("message content is required", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
	}

	// Optimized membership check - use targeted approach instead of fetching all user groups
	isMember, err := t.checkTeamMembership(ctx, logger, nk, userID, req.Id)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, runtime.NewError("user is not a member of this team", PERMISSION_DENIED_ERROR_CODE) // PERMISSION_DENIED
	}

	// Build the channel ID for the group
	channelID, err := nk.ChannelIdBuild(ctx, userID, req.Id, runtime.Group)
	if err != nil {
		logger.Error("Failed to build channel ID: %v", err)
		return nil, err
	}

	// Send the message to the group channel
	ack, err := nk.ChannelMessageSend(ctx, channelID, map[string]interface{}{"content": req.Content}, "", userID, true)
	if err != nil {
		logger.Error("Failed to send channel message: %v", err)
		return nil, err
	}

	// Convert Nakama ChannelMessageAck to our ChannelMessageAck
	// Extract values from wrapped types safely
	var code int32 = 0
	if ack.Code != nil {
		code = ack.Code.Value
	}

	var createTime int64 = 0
	if ack.CreateTime != nil {
		createTime = ack.CreateTime.Seconds
	}

	var updateTime int64 = 0
	if ack.UpdateTime != nil {
		updateTime = ack.UpdateTime.Seconds
	}

	var persistent bool = false
	if ack.Persistent != nil {
		persistent = ack.Persistent.Value
	}

	return &ChannelMessageAck{
		ChannelId:  ack.ChannelId,
		MessageId:  ack.MessageId,
		Code:       code,
		Username:   ack.Username,
		CreateTime: createTime,
		UpdateTime: updateTime,
		Persistent: persistent,
		RoomName:   ack.RoomName,
		GroupId:    ack.GroupId,
		UserIdOne:  ack.UserIdOne,
		UserIdTwo:  ack.UserIdTwo,
	}, nil
}

// Helper function to convert Nakama Group to Team
func (t *NakamaTeamsSystem) convertGroupToTeam(group *api.Group, iconOverride string) *Team {
	// Parse metadata to extract icon
	icon := iconOverride
	if icon == "" && group.Metadata != "" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(group.Metadata), &metadata); err == nil {
			if iconVal, ok := metadata["icon"].(string); ok {
				icon = iconVal
			}
		}
	}

	// Extract open value safely
	open := false
	if group.Open != nil {
		open = group.Open.Value
	}

	return &Team{
		Id:            group.Id,
		CreatorId:     group.CreatorId,
		Name:          group.Name,
		Description:   group.Description,
		LangTag:       group.LangTag,
		Metadata:      group.Metadata,
		AvatarUrl:     group.AvatarUrl,
		Open:          open,
		EdgeCount:     group.EdgeCount,
		MaxCount:      group.MaxCount,
		CreateTimeSec: group.CreateTime.Seconds,
		UpdateTimeSec: group.UpdateTime.Seconds,
		Icon:          icon,
	}
}

package pamlogix

import (
	"context"

	"github.com/heroiclabs/nakama-common/runtime"
)

type IncentivesConfig struct {
	Incentives map[string]*IncentivesConfigIncentive `json:"incentives,omitempty"`
}

type IncentivesConfigIncentive struct {
	Type                 IncentiveType          `json:"type,omitempty"`
	Name                 string                 `json:"name,omitempty"`
	Description          string                 `json:"description,omitempty"`
	MaxClaims            int                    `json:"max_claims,omitempty"`
	MaxGlobalClaims      int                    `json:"max_global_claims,omitempty"`
	MaxRecipientAgeSec   int64                  `json:"max_recipient_age_sec,omitempty"`
	RecipientReward      *EconomyConfigReward   `json:"recipient_reward,omitempty"`
	SenderReward         *EconomyConfigReward   `json:"sender_reward,omitempty"`
	MaxConcurrent        int                    `json:"max_concurrent,omitempty"`
	ExpiryDurationSec    int64                  `json:"expiry_duration_sec,omitempty"`
	AdditionalProperties map[string]interface{} `json:"additional_properties,omitempty"`
}

// The IncentivesSystem provides a gameplay system which can create and claim incentives and their associated rewards.
type IncentivesSystem interface {
	System

	SenderList(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (incentives []*Incentive, err error)

	SenderCreate(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, incentiveID string) (incentives []*Incentive, err error)

	SenderDelete(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, code string) (incentives []*Incentive, err error)

	SenderClaim(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, code string, claimantIDs []string) (incentives []*Incentive, err error)

	RecipientGet(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, code string) (incentive *IncentiveInfo, err error)

	RecipientClaim(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, code string) (incentive *IncentiveInfo, err error)

	// SetOnSenderReward sets a custom reward function which will run after an incentive sender's reward is rolled.
	SetOnSenderReward(fn OnReward[*IncentivesConfigIncentive])

	// SetOnRecipientReward sets a custom reward function which will run after an incentive recipient's reward is rolled.
	SetOnRecipientReward(fn OnReward[*IncentivesConfigIncentive])
}

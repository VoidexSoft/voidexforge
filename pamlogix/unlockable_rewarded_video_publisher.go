package pamlogix

import (
	"context"

	"github.com/heroiclabs/nakama-common/runtime"
)

// UnlockableRewardedVideoPublisher listens for rewarded video unlock events and unlocks the unlockable.
type UnlockableRewardedVideoPublisher struct {
	Unlockables UnlockablesSystem
}

func (p *UnlockableRewardedVideoPublisher) Authenticate(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, created bool) {
	// No-op
}

func (p *UnlockableRewardedVideoPublisher) Send(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, events []*PublisherEvent) {
	for _, event := range events {
		if event.Name == "placement_success" {
			// Check if this placement is for unlockables (by placement_id or other metadata)
			placementID := ""
			instanceID := ""
			if event.Metadata != nil {
				placementID = event.Metadata["placement_id"]
				instanceID = event.Metadata["instance_id"]
			}
			// You may want to check placementID against a known unlockable placement ID
			if placementID == "unlockable_rewarded_video" && instanceID != "" && p.Unlockables != nil {
				_, err := p.Unlockables.PurchaseUnlock(ctx, logger, nk, userID, instanceID)
				if err != nil {
					logger.Error("Failed to instantly unlock unlockable via rewarded video: %v", err)
				}
			}
		}
	}
}

package examples

import (
	"context"
	"database/sql"
	"encoding/json"

	"voidexforge/pamlogix"

	"github.com/heroiclabs/nakama-common/runtime"
	"go.uber.org/zap"
)

// BattlePassManager handles all battle pass operations
type BattlePassManager struct {
	progressionSystem pamlogix.ProgressionSystem
	economySystem     pamlogix.EconomySystem
}

// NewBattlePassManager creates a new battle pass manager
func NewBattlePassManager(progressionSystem pamlogix.ProgressionSystem, economySystem pamlogix.EconomySystem) *BattlePassManager {
	return &BattlePassManager{
		progressionSystem: progressionSystem,
		economySystem:     economySystem,
	}
}

// AwardBattlePassXP awards XP to a player and checks for tier unlocks
func (bpm *BattlePassManager) AwardBattlePassXP(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, xpAmount int64, reason string) error {
	// Award XP
	_, err := bpm.progressionSystem.Update(ctx, logger, nk, userID, "battle_pass_xp_tracker", map[string]int64{
		"battle_pass_xp": xpAmount,
	})
	if err != nil {
		return err
	}

	// Check for newly unlocked tiers
	progressions, deltas, err := bpm.progressionSystem.Get(ctx, logger, nk, userID, nil)
	if err != nil {
		return err
	}

	// Process newly unlocked tiers
	for tierID, delta := range deltas {
		if delta.State == pamlogix.ProgressionDeltaState_PROGRESSION_DELTA_STATE_UNLOCKED {
			// Check if this is a battle pass tier
			if progression, exists := progressions[tierID]; exists {
				if progression.Category == "battle_pass" {
					logger.Info("Battle pass tier unlocked", zap.String("userID", userID), zap.String("tierID", tierID), zap.String("reason", reason))

					// Automatically grant rewards
					_, _, err := bpm.progressionSystem.Complete(ctx, logger, nk, userID, tierID)
					if err != nil {
						logger.Error("Failed to complete battle pass tier", zap.Error(err))
					}
				}
			}
		}
	}

	return nil
}

// PurchasePremiumPass handles premium battle pass purchase
func (bpm *BattlePassManager) PurchasePremiumPass(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) error {
	// Grant premium pass item
	reward := &pamlogix.Reward{
		Items: map[string]int64{"battle_pass_premium": 1},
	}

	_, _, _, err := bpm.economySystem.RewardGrant(ctx, logger, nk, userID, reward, map[string]interface{}{
		"source": "premium_purchase",
		"season": "winter_2024",
	}, false)
	if err != nil {
		return err
	}

	// Check for retroactive premium rewards
	return bpm.GrantRetroactivePremiumRewards(ctx, logger, nk, userID)
}

// GrantRetroactivePremiumRewards grants premium rewards for already completed tiers
func (bpm *BattlePassManager) GrantRetroactivePremiumRewards(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) error {
	progressions, _, err := bpm.progressionSystem.Get(ctx, logger, nk, userID, nil)
	if err != nil {
		return err
	}

	for tierID, progression := range progressions {
		// Check if this is a premium tier that's already unlocked
		if progression.Unlocked && progression.Category == "battle_pass" {
			if passType, exists := progression.AdditionalProperties["pass_type"]; exists && passType == "premium" {
				// Grant premium rewards retroactively
				_, _, err := bpm.progressionSystem.Complete(ctx, logger, nk, userID, tierID)
				if err != nil {
					logger.Error("Failed to grant retroactive premium reward", zap.Error(err), zap.String("tierID", tierID))
				} else {
					logger.Info("Granted retroactive premium reward", zap.String("userID", userID), zap.String("tierID", tierID))
				}
			}
		}
	}

	return nil
}

// CompleteDailyChallenge handles daily challenge completion
func (bpm *BattlePassManager) CompleteDailyChallenge(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, challengeType string) error {
	// Update daily challenge count
	_, err := bpm.progressionSystem.Update(ctx, logger, nk, userID, "daily_challenges", map[string]int64{
		"daily_logins": 1,
	})
	if err != nil {
		return err
	}

	// Complete the daily challenge progression to get XP reward
	_, _, err = bpm.progressionSystem.Complete(ctx, logger, nk, userID, "battle_pass_daily_login")
	return err
}

// CompleteWeeklyChallenge handles weekly challenge completion
func (bpm *BattlePassManager) CompleteWeeklyChallenge(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) error {
	// Update weekly challenge count
	_, err := bpm.progressionSystem.Update(ctx, logger, nk, userID, "weekly_challenges", map[string]int64{
		"weekly_challenges_completed": 1,
	})
	if err != nil {
		return err
	}

	// Check if weekly challenge progression is complete
	progressions, _, err := bpm.progressionSystem.Get(ctx, logger, nk, userID, nil)
	if err != nil {
		return err
	}

	if progression, exists := progressions["battle_pass_weekly_challenge"]; exists && progression.Unlocked {
		// Complete weekly challenge to get XP bonus
		_, _, err = bpm.progressionSystem.Complete(ctx, logger, nk, userID, "battle_pass_weekly_challenge")
		return err
	}

	return nil
}

// GetBattlePassStatus returns the current battle pass status for a user
func (bpm *BattlePassManager) GetBattlePassStatus(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (*BattlePassStatus, error) {
	progressions, _, err := bpm.progressionSystem.Get(ctx, logger, nk, userID, nil)
	if err != nil {
		return nil, err
	}

	status := &BattlePassStatus{
		Season:       "winter_2024",
		HasPremium:   false,
		CurrentXP:    0,
		FreeTiers:    make([]TierStatus, 0),
		PremiumTiers: make([]TierStatus, 0),
	}

	// Check if user has premium pass
	if progression, exists := progressions["battle_pass_premium_check"]; exists && progression.Unlocked {
		status.HasPremium = true
	}

	// Get current XP from progression counts
	if xpProgression, exists := progressions["battle_pass_xp_tracker"]; exists {
		if xp, exists := xpProgression.Counts["battle_pass_xp"]; exists {
			status.CurrentXP = xp
		}
	}

	// Process all battle pass tiers
	for tierID, progression := range progressions {
		if progression.Category == "battle_pass" {
			tierStatus := TierStatus{
				TierID:   tierID,
				Name:     progression.Name,
				Unlocked: progression.Unlocked,
			}

			if tier, exists := progression.AdditionalProperties["tier"]; exists {
				tierStatus.Tier = tier
			}

			if passType, exists := progression.AdditionalProperties["pass_type"]; exists {
				if passType == "premium" {
					status.PremiumTiers = append(status.PremiumTiers, tierStatus)
				} else {
					status.FreeTiers = append(status.FreeTiers, tierStatus)
				}
			}
		}
	}

	return status, nil
}

// ResetSeason resets all battle pass progress for a new season
func (bpm *BattlePassManager) ResetSeason(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) error {
	// Get all battle pass progressions
	progressions, _, err := bpm.progressionSystem.Get(ctx, logger, nk, userID, nil)
	if err != nil {
		return err
	}

	battlePassTiers := make([]string, 0)
	for tierID, progression := range progressions {
		if progression.Category == "battle_pass" {
			battlePassTiers = append(battlePassTiers, tierID)
		}
	}

	// Reset all battle pass progressions
	_, err = bpm.progressionSystem.Reset(ctx, logger, nk, userID, battlePassTiers)
	return err
}

// Data structures for battle pass status
type BattlePassStatus struct {
	Season       string       `json:"season"`
	HasPremium   bool         `json:"has_premium"`
	CurrentXP    int64        `json:"current_xp"`
	FreeTiers    []TierStatus `json:"free_tiers"`
	PremiumTiers []TierStatus `json:"premium_tiers"`
}

type TierStatus struct {
	TierID    string `json:"tier_id"`
	Name      string `json:"name"`
	Tier      string `json:"tier"`
	Unlocked  bool   `json:"unlocked"`
	Completed bool   `json:"completed"`
}

// RPC handlers for battle pass operations

// rpcBattlePassStatus returns the current battle pass status
func rpcBattlePassStatus(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
	if !ok {
		return "", runtime.NewError("No user ID found", 3)
	}

	// Get battle pass manager from context or initialize
	battlePassManager := getBattlePassManager(ctx)

	status, err := battlePassManager.GetBattlePassStatus(ctx, logger, nk, userID)
	if err != nil {
		return "", runtime.NewError("Failed to get battle pass status", 13)
	}

	statusJSON, err := json.Marshal(status)
	if err != nil {
		return "", runtime.NewError("Failed to marshal status", 13)
	}

	return string(statusJSON), nil
}

// rpcBattlePassPurchasePremium handles premium pass purchase
func rpcBattlePassPurchasePremium(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
	if !ok {
		return "", runtime.NewError("No user ID found", 3)
	}

	battlePassManager := getBattlePassManager(ctx)

	err := battlePassManager.PurchasePremiumPass(ctx, logger, nk, userID)
	if err != nil {
		return "", runtime.NewError("Failed to purchase premium pass", 13)
	}

	return `{"success": true}`, nil
}

// rpcBattlePassCompleteDaily handles daily challenge completion
func rpcBattlePassCompleteDaily(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
	if !ok {
		return "", runtime.NewError("No user ID found", 3)
	}

	var request struct {
		ChallengeType string `json:"challenge_type"`
	}

	if err := json.Unmarshal([]byte(payload), &request); err != nil {
		return "", runtime.NewError("Invalid payload", 3)
	}

	battlePassManager := getBattlePassManager(ctx)

	err := battlePassManager.CompleteDailyChallenge(ctx, logger, nk, userID, request.ChallengeType)
	if err != nil {
		return "", runtime.NewError("Failed to complete daily challenge", 13)
	}

	return `{"success": true}`, nil
}

// Helper function to get battle pass manager from context
func getBattlePassManager(ctx context.Context) *BattlePassManager {
	// In a real implementation, you would retrieve this from your dependency injection system
	// or initialize it with your progression and economy systems
	return nil // Placeholder
}

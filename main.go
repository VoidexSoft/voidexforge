package main

import (
	"context"
	"database/sql"
	"time"
	"voidexforge/pamalyze"
	"voidexforge/pamlogix"

	"github.com/heroiclabs/nakama-common/runtime"
)

// noinspection GoUnusedExportedFunction
func InitModule(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, initializer runtime.Initializer) error {
	initStart := time.Now()

	logger.Info("Loading Voidexforge Nakama plugin...")

	er := initializer.RegisterRpc("hello_world", func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		return `{"message": "Hello, World!"}`, nil
	})
	if er != nil {
		logger.Error("Failed to register hello_world RPC: %v", er)
		return er
	}
	ping := pamalyze.NewPingService()
	//register ping service from pamalyze
	er = initializer.RegisterRpc("ping", ping.Ping)
	if er != nil {
		logger.Error("Failed to register ping RPC: %v", er)
		return er
	}

	_, err := pamlogix.Init(ctx, logger, nk, initializer,
		pamlogix.WithBaseSystem("configs/base.json", true),
		pamlogix.WithAchievementsSystem("configs/achievements.json", true),
		pamlogix.WithAuctionsSystem("configs/auctions.json", true),
		pamlogix.WithEconomySystem("configs/economy.json", true),
		pamlogix.WithEnergySystem("configs/energy.json", true),
		pamlogix.WithEventLeaderboardsSystem("configs/event_leaderboards.json", true),
		pamlogix.WithIncentivesSystem("configs/incentives.json", true),
		pamlogix.WithInventorySystem("configs/inventory.json", true),
		pamlogix.WithLeaderboardsSystem("configs/leaderboards.json", true),
		pamlogix.WithProgressionSystem("configs/progression.json", true),
		pamlogix.WithStatsSystem("configs/stats.json", true),
		pamlogix.WithStreaksSystem("configs/streaks.json", true),
		pamlogix.WithTeamsSystem("configs/teams.json", true),
		pamlogix.WithTutorialsSystem("configs/tutorials.json", true),
		pamlogix.WithUnlockablesSystem("configs/unlockables.json", true))
	if err != nil {
		return err
	}

	logger.Info("Voidexforge Nakama plugin loaded in '%d' msec.", time.Now().Sub(initStart).Milliseconds())
	return nil
}

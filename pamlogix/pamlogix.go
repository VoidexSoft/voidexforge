package pamlogix

import (
	"context"
	"encoding/json"
	"io"

	"github.com/heroiclabs/nakama-common/runtime"
)

// pamlogixImpl implements the Pamlogix interface
type pamlogixImpl struct {
	personalizers      []Personalizer
	publishers         []Publisher
	afterAuthenticate  AfterAuthenticateFn
	collectionResolver CollectionResolverFn

	// Store systems in a map by type
	systems map[SystemType]System
}

// Init initializes a Pamlogix type with the configurations provided.
func Init(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, initializer runtime.Initializer, configs ...SystemConfig) (Pamlogix, error) {
	// Create a new pamlogix implementation
	pl := &pamlogixImpl{
		personalizers:      make([]Personalizer, 0),
		publishers:         make([]Publisher, 0),
		collectionResolver: nil,
		afterAuthenticate:  nil,
		systems:            make(map[SystemType]System),
	}

	// Initialize systems based on provided configs
	for _, config := range configs {
		if err := pl.initSystem(ctx, logger, nk, initializer, config); err != nil {
			return nil, err
		}
	}

	// Register UnlockableRewardedVideoPublisher if Unlockables system is present
	if unlockables, ok := pl.systems[SystemTypeUnlockables].(UnlockablesSystem); ok {
		pl.AddPublisher(&UnlockableRewardedVideoPublisher{Unlockables: unlockables})
	}

	return pl, nil
}

// initSystem initializes a specific system based on its type
func (p *pamlogixImpl) initSystem(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, initializer runtime.Initializer, config SystemConfig) error {
	// Log the initialization
	logger.Info("Initializing system type: %v, config file: %s", config.GetType(), config.GetConfigFile())

	// 1. Load and parse the config file
	configData, err := nk.ReadFile(config.GetConfigFile())
	if err != nil {
		logger.Error("Failed to read config file %s: %v", config.GetConfigFile(), err)
		return err
	}

	// Read file contents
	configBytes, err := io.ReadAll(configData)
	if err != nil {
		logger.Error("Failed to read config file contents: %v", err)
		return err
	}
	defer configData.Close()

	// 2. Create the appropriate system instance based on system type
	var system System

	switch config.GetType() {
	case SystemTypeBase:
		baseConfig := &BaseSystemConfig{}
		if err := json.Unmarshal(configBytes, baseConfig); err != nil {
			logger.Error("Failed to parse Base system config: %v", err)
			return err
		}
		system = &BasePamlogix{}

	case SystemTypeEnergy:
		energyConfig := &EnergyConfig{}
		if err := json.Unmarshal(configBytes, energyConfig); err != nil {
			logger.Error("Failed to parse Energy system config: %v", err)
			return err
		}
		system = NewNakamaEnergySystem(energyConfig)

	case SystemTypeInventory:
		inventoryConfig := &InventoryConfig{}
		if err := json.Unmarshal(configBytes, inventoryConfig); err != nil {
			logger.Error("Failed to parse Inventory system config: %v", err)
			return err
		}
		system = NewNakamaInventorySystem(inventoryConfig)

	case SystemTypeEconomy:
		economyConfig := &EconomyConfig{}
		if err := json.Unmarshal(configBytes, economyConfig); err != nil {
			logger.Error("Failed to parse Economy system config: %v", err)
			return err
		}
		system = NewNakamaEconomySystem(economyConfig)

	case SystemTypeAchievements:
		achievementsConfig := &AchievementsConfig{}
		if err := json.Unmarshal(configBytes, achievementsConfig); err != nil {
			logger.Error("Failed to parse Achievements system config: %v", err)
			return err
		}
		system = NewNakamaAchievementsSystem(achievementsConfig)

	case SystemTypeLeaderboards:
		leaderboardConfig := &LeaderboardConfig{}
		if err := json.Unmarshal(configBytes, leaderboardConfig); err != nil {
			logger.Error("Failed to parse Leaderboards system config: %v", err)
			return err
		}
		// Create leaderboards system instance

	case SystemTypeStats:
		statsConfig := &StatsConfig{}
		if err := json.Unmarshal(configBytes, statsConfig); err != nil {
			logger.Error("Failed to parse Stats system config: %v", err)
			return err
		}
		system = NewStatsSystem(statsConfig)

	case SystemTypeTeams:
		teamsConfig := &TeamsConfig{}
		if err := json.Unmarshal(configBytes, teamsConfig); err != nil {
			logger.Error("Failed to parse Teams system config: %v", err)
			return err
		}
		teamsSystem := NewNakamaTeamsSystem(teamsConfig)

		// Set custom validation function if provided
		if extra := config.GetExtra(); extra != nil {
			if validateCreateTeamFns, ok := extra.([]ValidateCreateTeamFn); ok && len(validateCreateTeamFns) > 0 {
				teamsSystem.SetValidateCreateTeam(validateCreateTeamFns[0])
				logger.Info("Set custom validateCreateTeam function for teams system")
			}
		}

		system = teamsSystem

	case SystemTypeTutorials:
		tutorialsConfig := &TutorialsConfig{}
		if err := json.Unmarshal(configBytes, tutorialsConfig); err != nil {
			logger.Error("Failed to parse Tutorials system config: %v", err)
			return err
		}
		system = NewNakamaTutorialsSystem(tutorialsConfig)

	case SystemTypeUnlockables:
		unlockablesConfig := &UnlockablesConfig{}
		if err := json.Unmarshal(configBytes, unlockablesConfig); err != nil {
			logger.Error("Failed to parse Unlockables system config: %v", err)
			return err
		}
		system = NewUnlockablesSystem(unlockablesConfig)

	case SystemTypeEventLeaderboards:
		eventLeaderboardsConfig := &EventLeaderboardsConfig{}
		if err := json.Unmarshal(configBytes, eventLeaderboardsConfig); err != nil {
			logger.Error("Failed to parse EventLeaderboards system config: %v", err)
			return err
		}
		system = NewNakamaEventLeaderboardsSystem(eventLeaderboardsConfig)

	case SystemTypeProgression:
		progressionConfig := &ProgressionConfig{}
		if err := json.Unmarshal(configBytes, progressionConfig); err != nil {
			logger.Error("Failed to parse Progression system config: %v", err)
			return err
		}
		system = NewNakamaProgressionSystem(progressionConfig)

	case SystemTypeIncentives:
		incentivesConfig := &IncentivesConfig{}
		if err := json.Unmarshal(configBytes, incentivesConfig); err != nil {
			logger.Error("Failed to parse Incentives system config: %v", err)
			return err
		}
		system = NewNakamaIncentivesSystem(incentivesConfig)

	case SystemTypeAuctions:
		auctionsConfig := &AuctionsConfig{}
		if err := json.Unmarshal(configBytes, auctionsConfig); err != nil {
			logger.Error("Failed to parse Auctions system config: %v", err)
			return err
		}
		system = NewNakamaAuctionsSystem(auctionsConfig)

	case SystemTypeStreaks:
		streaksConfig := &StreaksConfig{}
		if err := json.Unmarshal(configBytes, streaksConfig); err != nil {
			logger.Error("Failed to parse Streaks system config: %v", err)
			return err
		}
		system = NewNakamaStreaksSystem(streaksConfig)

	default:
		logger.Error("Unknown system type: %v", config.GetType())
		return runtime.NewError("unknown system type", 3) // INVALID_ARGUMENT
	}

	// 3. Store the system in the systems map if it was created
	if system != nil {
		// Apply any personalizers to the system
		for _, personalizer := range p.personalizers {
			personalizedConfig, err := personalizer.GetValue(ctx, logger, nk, system, "")
			if err != nil {
				logger.Warn("Failed to get personalized config: %v", err)
				// Continue despite error
			}

			// If personalization was successful and returned a config, we don't need to do anything
			// since the personalizer would have modified the system's config directly
			if personalizedConfig != nil {
				logger.Info("Applied personalization to system type: %v", system.GetType())
			}
		}

		// Store the system
		p.systems[config.GetType()] = system

		// For energy system, set the Pamlogix reference to enable cross-system communication
		if energySystem, ok := system.(*NakamaEnergySystem); ok {
			energySystem.SetPamlogix(p)
			logger.Info("Set Pamlogix reference in energy system for cross-system communication")
		}

		// For achievement system, set the Pamlogix reference to enable cross-system communication
		if achievementSystem, ok := system.(*NakamaAchievementsSystem); ok {
			achievementSystem.SetPamlogix(p)
			logger.Info("Set Pamlogix reference in achievement system for cross-system communication")
		}

		// For tutorials system, set the Pamlogix reference to enable cross-system communication
		if tutorialsSystem, ok := system.(*NakamaTutorialsSystem); ok {
			tutorialsSystem.SetPamlogix(p)
			logger.Info("Set Pamlogix reference in tutorials system for cross-system communication")
		}

		// For unlockables system, set the Pamlogix reference to enable cross-system communication
		if unlockablesSystem, ok := system.(*UnlockablesPamlogix); ok {
			unlockablesSystem.SetPamlogix(p)
			logger.Info("Set Pamlogix reference in unlockables system for cross-system communication")
		}

		// For streaks system, set the Pamlogix reference to enable cross-system communication
		if streaksSystem, ok := system.(*NakamaStreaksSystem); ok {
			streaksSystem.SetPamlogix(p)
			logger.Info("Set Pamlogix reference in streaks system for cross-system communication")
		}

		// For auctions system, set the Pamlogix reference to enable cross-system communication
		if auctionsSystem, ok := system.(*AuctionsPamlogix); ok {
			auctionsSystem.SetPamlogix(p)
			logger.Info("Set Pamlogix reference in auctions system for cross-system communication")
		}

		// For teams system, set the Pamlogix reference to enable cross-system communication
		if teamsSystem, ok := system.(*NakamaTeamsSystem); ok {
			teamsSystem.SetPamlogix(p)
			logger.Info("Set Pamlogix reference in teams system for cross-system communication")
		}

		// For progression system, set the Pamlogix reference to enable cross-system communication
		if progressionSystem, ok := system.(*NakamaProgressionSystem); ok {
			progressionSystem.SetPamlogix(p)
			logger.Info("Set Pamlogix reference in progression system for cross-system communication")
		}

		// For event leaderboards system, set the Pamlogix reference to enable cross-system communication
		if eventLeaderboardsSystem, ok := system.(*NakamaEventLeaderboardsSystem); ok {
			eventLeaderboardsSystem.SetPamlogix(p)
			logger.Info("Set Pamlogix reference in event leaderboards system for cross-system communication")
		}
	}

	// 4. Register RPCs if requested
	if config.GetRegister() {
		if err := p.registerSystemRpcs(initializer, config.GetType()); err != nil {
			return err
		}
		if err := p.registerSystemRpcs_Json(initializer, config.GetType()); err != nil {
			return err
		}
	}

	return nil
}

// registerSystemRpcs registers the appropriate RPCs for a given system type
func (p *pamlogixImpl) registerSystemRpcs(initializer runtime.Initializer, systemType SystemType) error {
	switch systemType {
	case SystemTypeAchievements:
		// Register Achievements system RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ACHIEVEMENTS_CLAIM.String(), rpcAchievementsClaim(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ACHIEVEMENTS_GET.String(), rpcAchievementsGet(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ACHIEVEMENTS_UPDATE.String(), rpcAchievementsUpdate(p)); err != nil {
			return err
		}
		//// Register additional achievement RPCs using the same RPC IDs with different endpoints
		//if err := initializer.RegisterRpc("achievements_list", rpcAchievementsList(p)); err != nil {
		//	return err
		//}

	case SystemTypeBase:
		// Register Base system RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_BASE_RATE_APP.String(), rpcBaseRateApp(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_BASE_SET_DEVICE_PREFS.String(), rpcBaseSetDevicePrefs(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_BASE_SYNC.String(), rpcBaseSync(p)); err != nil {
			return err
		}

	case SystemTypeEconomy:
		// Register Economy system RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_DONATION_CLAIM.String(), rpcEconomyDonationClaim(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_DONATION_GIVE.String(), rpcEconomyDonationGive(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_DONATION_GET.String(), rpcEconomyDonationGet(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_DONATION_REQUEST.String(), rpcEconomyDonationRequest(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_STORE_GET.String(), rpcEconomyStoreGet(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_GRANT.String(), rpcEconomyGrant(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_PURCHASE_INTENT.String(), rpcEconomyPurchaseIntent(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_PURCHASE_ITEM.String(), rpcEconomyPurchaseItem(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_PURCHASE_RESTORE.String(), rpcEconomyPurchaseRestore(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_PLACEMENT_STATUS.String(), rpcEconomyPlacementStatus(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_PLACEMENT_START.String(), rpcEconomyPlacementStart(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_PLACEMENT_SUCCESS.String(), rpcEconomyPlacementSuccess(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_PLACEMENT_FAIL.String(), rpcEconomyPlacementFail(p)); err != nil {
			return err
		}
	case SystemTypeEventLeaderboards:
		// Register EventLeaderboards system RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_EVENT_LEADERBOARD_LIST.String(), rpcEventLeaderboardsList(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_EVENT_LEADERBOARD_GET.String(), rpcEventLeaderboardsGet(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_EVENT_LEADERBOARD_UPDATE.String(), rpcEventLeaderboardsUpdate(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_EVENT_LEADERBOARD_CLAIM.String(), rpcEventLeaderboardsClaim(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_EVENT_LEADERBOARD_ROLL.String(), rpcEventLeaderboardsRoll(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_EVENT_LEADERBOARD_DEBUG_FILL.String(), rpcEventLeaderboardsDebugFill(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_EVENT_LEADERBOARD_DEBUG_RANDOM_SCORES.String(), rpcEventLeaderboardsDebugRandomScores(p)); err != nil {
			return err
		}

	case SystemTypeEnergy:
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ENERGY_GET.String(), rpcEnergyGet(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ENERGY_SPEND.String(), rpcEnergySpend(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ENERGY_GRANT.String(), rpcEnergyGrant(p)); err != nil {
			return err
		}

	case SystemTypeInventory:
		// Register Inventory system RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_INVENTORY_LIST.String(), rpcInventoryList(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_INVENTORY_LIST_INVENTORY.String(), rpcInventoryListInventory(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_INVENTORY_CONSUME.String(), rpcInventoryConsume(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_INVENTORY_GRANT.String(), rpcInventoryGrant(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_INVENTORY_UPDATE.String(), rpcInventoryUpdate(p)); err != nil {
			return err
		}

	case SystemTypeStats:
		if err := initializer.RegisterRpc(RpcId_RPC_ID_STATS_GET.String(), rpcStatsGet(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_STATS_UPDATE.String(), rpcStatsUpdate(p)); err != nil {
			return err
		}

	case SystemTypeTutorials:
		// Register Tutorials system RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_TUTORIALS_GET.String(), rpcTutorialsGet(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_TUTORIALS_ACCEPT.String(), rpcTutorialsAccept(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_TUTORIALS_DECLINE.String(), rpcTutorialsDecline(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_TUTORIALS_ABANDON.String(), rpcTutorialsAbandon(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_TUTORIALS_UPDATE.String(), rpcTutorialsUpdate(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_TUTORIALS_RESET.String(), rpcTutorialsReset(p)); err != nil {
			return err
		}

	case SystemTypeUnlockables:
		// Register Unlockables system RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_UNLOCKABLES_CREATE.String(), rpcUnlockablesCreate(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_UNLOCKABLES_GET.String(), rpcUnlockablesGet(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_UNLOCKABLES_UNLOCK_START.String(), rpcUnlockablesUnlockStart(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_UNLOCKABLES_PURCHASE_UNLOCK.String(), rpcUnlockablesPurchaseUnlock(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_UNLOCKABLES_PURCHASE_SLOT.String(), rpcUnlockablesPurchaseSlot(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_UNLOCKABLES_CLAIM.String(), rpcUnlockablesClaim(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_UNLOCKABLES_QUEUE_ADD.String(), rpcUnlockablesQueueAdd(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_UNLOCKABLES_QUEUE_REMOVE.String(), rpcUnlockablesQueueRemove(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_UNLOCKABLES_QUEUE_SET.String(), rpcUnlockablesQueueSet(p)); err != nil {
			return err
		}

	case SystemTypeAuctions:
		// Register Auctions system RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_AUCTIONS_GET_TEMPLATES.String(), rpcAuctionsGetTemplates(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_AUCTIONS_LIST.String(), rpcAuctionsList(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_AUCTIONS_BID.String(), rpcAuctionsBid(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_AUCTIONS_CLAIM_BID.String(), rpcAuctionsClaimBid(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_AUCTIONS_CLAIM_CREATED.String(), rpcAuctionsClaimCreated(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_AUCTIONS_CANCEL.String(), rpcAuctionsCancel(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_AUCTIONS_CREATE.String(), rpcAuctionsCreate(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_AUCTIONS_LIST_BIDS.String(), rpcAuctionsListBids(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_AUCTIONS_LIST_CREATED.String(), rpcAuctionsListCreated(p)); err != nil {
			return err
		}

		if err := initializer.RegisterRpc(RpcSocketId_RPC_SOCKET_ID_AUCTIONS_FOLLOW.String(), rpcAuctionsFollow(p)); err != nil {
			return err
		}
		//
		//// Optionally register a real-time message handler for auction-related messages
		//// This allows intercepting and processing real-time auction messages if needed
		//if err := initializer.RegisterBeforeRt("rpc", func(ctx context.Context, rtLogger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, in *rtapi.Envelope) (*rtapi.Envelope, error) {
		//	// Check if this is an auction-related RPC call over websocket
		//	if rpc := in.GetRpc(); rpc != nil {
		//		if rpc.Id == RpcSocketId_RPC_SOCKET_ID_AUCTIONS_FOLLOW.String() {
		//			// This RPC is being called over websocket - allow it to proceed
		//			// The real-time updates will be handled by the stream system
		//			rtLogger.Debug("Auction follow RPC called over websocket for real-time updates")
		//		}
		//	}
		//	return in, nil
		//}); err != nil {
		//	// Failed to register real-time message handler for auctions - this is optional
		//	// Don't return error as this is optional
		//}

	case SystemTypeStreaks:
		// Register Streaks system RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_STREAKS_LIST.String(), rpcStreaksList(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_STREAKS_UPDATE.String(), rpcStreaksUpdate(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_STREAKS_CLAIM.String(), rpcStreaksClaim(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_STREAKS_RESET.String(), rpcStreaksReset(p)); err != nil {
			return err
		}

	case SystemTypeProgression:
		// Register Progression system RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_PROGRESSIONS_GET.String(), rpcProgressionsGet(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_PROGRESSIONS_PURCHASE.String(), rpcProgressionsPurchase(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_PROGRESSIONS_UPDATE.String(), rpcProgressionsUpdate(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_PROGRESSIONS_RESET.String(), rpcProgressionsReset(p)); err != nil {
			return err
		}

	case SystemTypeTeams:
		// Register Teams system RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_TEAMS_CREATE.String(), rpcTeamsCreate(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_TEAMS_LIST.String(), rpcTeamsList(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_TEAMS_SEARCH.String(), rpcTeamsSearch(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_TEAMS_WRITE_CHAT_MESSAGE.String(), rpcTeamsWriteChatMessage(p)); err != nil {
			return err
		}

	// Add other system types as needed...

	default:
		// Unknown system type, no RPCs to register
	}

	return nil
}

// SetPersonalizer is deprecated in favor of AddPersonalizer function to compose a chain of configuration personalization.
func (p *pamlogixImpl) SetPersonalizer(personalizer Personalizer) {
	p.personalizers = []Personalizer{personalizer}
}

// AddPersonalizer adds a personalizer to the chain
func (p *pamlogixImpl) AddPersonalizer(personalizer Personalizer) {
	p.personalizers = append(p.personalizers, personalizer)
}

// AddPublisher adds a publisher to the chain
func (p *pamlogixImpl) AddPublisher(publisher Publisher) {
	p.publishers = append(p.publishers, publisher)
}

// SetAfterAuthenticate sets the after authenticate function
func (p *pamlogixImpl) SetAfterAuthenticate(fn AfterAuthenticateFn) {
	p.afterAuthenticate = fn
}

// SetCollectionResolver sets a function that may change the storage collection target for Pamlogix systems.
func (p *pamlogixImpl) SetCollectionResolver(fn CollectionResolverFn) {
	p.collectionResolver = fn
}

// System getter implementations
func (p *pamlogixImpl) GetAchievementsSystem() AchievementsSystem {
	if sys, ok := p.systems[SystemTypeAchievements].(AchievementsSystem); ok {
		return sys
	}
	return nil
}

func (p *pamlogixImpl) GetBaseSystem() BaseSystem {
	if sys, ok := p.systems[SystemTypeBase].(BaseSystem); ok {
		return sys
	}
	return nil
}

func (p *pamlogixImpl) GetEconomySystem() EconomySystem {
	if sys, ok := p.systems[SystemTypeEconomy].(EconomySystem); ok {
		return sys
	}
	return nil
}

func (p *pamlogixImpl) GetEnergySystem() EnergySystem {
	if sys, ok := p.systems[SystemTypeEnergy].(EnergySystem); ok {
		return sys
	}
	return nil
}

func (p *pamlogixImpl) GetInventorySystem() InventorySystem {
	if sys, ok := p.systems[SystemTypeInventory].(InventorySystem); ok {
		return sys
	}
	return nil
}

func (p *pamlogixImpl) GetLeaderboardsSystem() LeaderboardsSystem {
	if sys, ok := p.systems[SystemTypeLeaderboards].(LeaderboardsSystem); ok {
		return sys
	}
	return nil
}

func (p *pamlogixImpl) GetStatsSystem() StatsSystem {
	if sys, ok := p.systems[SystemTypeStats].(StatsSystem); ok {
		return sys
	}
	return nil
}

func (p *pamlogixImpl) GetTeamsSystem() TeamsSystem {
	if sys, ok := p.systems[SystemTypeTeams].(TeamsSystem); ok {
		return sys
	}
	return nil
}

func (p *pamlogixImpl) GetTutorialsSystem() TutorialsSystem {
	if sys, ok := p.systems[SystemTypeTutorials].(TutorialsSystem); ok {
		return sys
	}
	return nil
}

func (p *pamlogixImpl) GetUnlockablesSystem() UnlockablesSystem {
	if sys, ok := p.systems[SystemTypeUnlockables].(UnlockablesSystem); ok {
		return sys
	}
	return nil
}

func (p *pamlogixImpl) GetEventLeaderboardsSystem() EventLeaderboardsSystem {
	if sys, ok := p.systems[SystemTypeEventLeaderboards].(EventLeaderboardsSystem); ok {
		return sys
	}
	return nil
}

func (p *pamlogixImpl) GetProgressionSystem() ProgressionSystem {
	if sys, ok := p.systems[SystemTypeProgression].(ProgressionSystem); ok {
		return sys
	}
	return nil
}

func (p *pamlogixImpl) GetIncentivesSystem() IncentivesSystem {
	if sys, ok := p.systems[SystemTypeIncentives].(IncentivesSystem); ok {
		return sys
	}
	return nil
}

func (p *pamlogixImpl) GetAuctionsSystem() AuctionsSystem {
	if sys, ok := p.systems[SystemTypeAuctions].(AuctionsSystem); ok {
		return sys
	}
	return nil
}

func (p *pamlogixImpl) GetStreaksSystem() StreaksSystem {
	if sys, ok := p.systems[SystemTypeStreaks].(StreaksSystem); ok {
		return sys
	}
	return nil
}

func (p *pamlogixImpl) GetChallengesSystem() ChallengesSystem {
	if sys, ok := p.systems[SystemTypeChallenges].(ChallengesSystem); ok {
		return sys
	}
	return nil
}

// SendPublisherEvents broadcasts events to all registered publishers
func (p *pamlogixImpl) SendPublisherEvents(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, events []*PublisherEvent) {
	if len(p.publishers) == 0 || len(events) == 0 {
		return
	}

	for _, publisher := range p.publishers {
		publisher.Send(ctx, logger, nk, userID, events)
	}
}

// BroadcastAuthEvent notifies all publishers about user authentication
func (p *pamlogixImpl) BroadcastAuthEvent(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, created bool) {
	if len(p.publishers) == 0 {
		return
	}

	for _, publisher := range p.publishers {
		publisher.Authenticate(ctx, logger, nk, userID, created)
	}
}

func (p *pamlogixImpl) registerSystemRpcs_Json(initializer runtime.Initializer, systemType SystemType) error {
	switch systemType {
	case SystemTypeAchievements:
		// Register Achievements system JSON RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ACHIEVEMENTS_CLAIM.String()+"_Json", rpcAchievementsClaim_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ACHIEVEMENTS_GET.String()+"_Json", rpcAchievementsGet_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ACHIEVEMENTS_UPDATE.String()+"_Json", rpcAchievementsUpdate_Json(p)); err != nil {
			return err
		}

	case SystemTypeBase:
		// Register Base system JSON RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_BASE_RATE_APP.String()+"_Json", rpcBaseRateApp(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_BASE_SET_DEVICE_PREFS.String()+"_Json", rpcBaseSetDevicePrefs(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_BASE_SYNC.String()+"_Json", rpcBaseSync(p)); err != nil {
			return err
		}

	case SystemTypeEconomy:
		// Register Economy system JSON RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_DONATION_CLAIM.String()+"_Json", rpcEconomyDonationClaim_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_DONATION_GIVE.String()+"_Json", rpcEconomyDonationGive_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_DONATION_GET.String()+"_Json", rpcEconomyDonationGet_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_DONATION_REQUEST.String()+"_Json", rpcEconomyDonationRequest_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_STORE_GET.String()+"_Json", rpcEconomyStoreGet_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_GRANT.String()+"_Json", rpcEconomyGrant_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_PURCHASE_INTENT.String()+"_Json", rpcEconomyPurchaseIntent_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_PURCHASE_ITEM.String()+"_Json", rpcEconomyPurchaseItem_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_PURCHASE_RESTORE.String()+"_Json", rpcEconomyPurchaseRestore_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_PLACEMENT_STATUS.String()+"_Json", rpcEconomyPlacementStatus_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_PLACEMENT_START.String()+"_Json", rpcEconomyPlacementStart_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_PLACEMENT_SUCCESS.String()+"_Json", rpcEconomyPlacementSuccess_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ECONOMY_PLACEMENT_FAIL.String()+"_Json", rpcEconomyPlacementFail_Json(p)); err != nil {
			return err
		}

	case SystemTypeEventLeaderboards:
		// Register EventLeaderboards system JSON RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_EVENT_LEADERBOARD_LIST.String()+"_Json", rpcEventLeaderboardsList(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_EVENT_LEADERBOARD_GET.String()+"_Json", rpcEventLeaderboardsGet(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_EVENT_LEADERBOARD_UPDATE.String()+"_Json", rpcEventLeaderboardsUpdate(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_EVENT_LEADERBOARD_CLAIM.String()+"_Json", rpcEventLeaderboardsClaim(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_EVENT_LEADERBOARD_ROLL.String()+"_Json", rpcEventLeaderboardsRoll(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_EVENT_LEADERBOARD_DEBUG_FILL.String()+"_Json", rpcEventLeaderboardsDebugFill(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_EVENT_LEADERBOARD_DEBUG_RANDOM_SCORES.String()+"_Json", rpcEventLeaderboardsDebugRandomScores(p)); err != nil {
			return err
		}

	case SystemTypeEnergy:
		// Register Energy system JSON RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ENERGY_GET.String()+"_Json", rpcEnergyGet_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ENERGY_SPEND.String()+"_Json", rpcEnergySpend_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_ENERGY_GRANT.String()+"_Json", rpcEnergyGrant_Json(p)); err != nil {
			return err
		}

	case SystemTypeInventory:
		// Register Inventory system JSON RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_INVENTORY_LIST.String()+"_Json", rpcInventoryList_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_INVENTORY_LIST_INVENTORY.String()+"_Json", rpcInventoryListInventory_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_INVENTORY_CONSUME.String()+"_Json", rpcInventoryConsume_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_INVENTORY_GRANT.String()+"_Json", rpcInventoryGrant_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_INVENTORY_UPDATE.String()+"_Json", rpcInventoryUpdate_Json(p)); err != nil {
			return err
		}

	case SystemTypeStats:
		// Register Stats system JSON RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_STATS_GET.String()+"_Json", rpcStatsGet_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_STATS_UPDATE.String()+"_Json", rpcStatsUpdate_Json(p)); err != nil {
			return err
		}

	case SystemTypeTutorials:
		// Register Tutorials system JSON RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_TUTORIALS_GET.String()+"_Json", rpcTutorialsGetJson(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_TUTORIALS_ACCEPT.String()+"_Json", rpcTutorialsAcceptJson(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_TUTORIALS_DECLINE.String()+"_Json", rpcTutorialsDeclineJson(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_TUTORIALS_ABANDON.String()+"_Json", rpcTutorialsAbandonJson(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_TUTORIALS_UPDATE.String()+"_Json", rpcTutorialsUpdateJson(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_TUTORIALS_RESET.String()+"_Json", rpcTutorialsResetJson(p)); err != nil {
			return err
		}

	case SystemTypeUnlockables:
		// Register Unlockables system JSON RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_UNLOCKABLES_CREATE.String()+"_Json", rpcUnlockablesCreateJson(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_UNLOCKABLES_GET.String()+"_Json", rpcUnlockablesGetJson(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_UNLOCKABLES_UNLOCK_START.String()+"_Json", rpcUnlockablesUnlockStartJson(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_UNLOCKABLES_PURCHASE_UNLOCK.String()+"_Json", rpcUnlockablesPurchaseUnlockJson(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_UNLOCKABLES_PURCHASE_SLOT.String()+"_Json", rpcUnlockablesPurchaseSlotJson(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_UNLOCKABLES_CLAIM.String()+"_Json", rpcUnlockablesClaimJson(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_UNLOCKABLES_QUEUE_ADD.String()+"_Json", rpcUnlockablesQueueAddJson(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_UNLOCKABLES_QUEUE_REMOVE.String()+"_Json", rpcUnlockablesQueueRemoveJson(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_UNLOCKABLES_QUEUE_SET.String()+"_Json", rpcUnlockablesQueueSetJson(p)); err != nil {
			return err
		}

	case SystemTypeAuctions:
		// Register Auctions system JSON RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_AUCTIONS_GET_TEMPLATES.String()+"_Json", rpcAuctionsGetTemplates_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_AUCTIONS_LIST.String()+"_Json", rpcAuctionsList_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_AUCTIONS_BID.String()+"_Json", rpcAuctionsBid_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_AUCTIONS_CLAIM_BID.String()+"_Json", rpcAuctionsClaimBid_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_AUCTIONS_CLAIM_CREATED.String()+"_Json", rpcAuctionsClaimCreated_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_AUCTIONS_CANCEL.String()+"_Json", rpcAuctionsCancel_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_AUCTIONS_CREATE.String()+"_Json", rpcAuctionsCreate_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_AUCTIONS_LIST_BIDS.String()+"_Json", rpcAuctionsListBids_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_AUCTIONS_LIST_CREATED.String()+"_Json", rpcAuctionsListCreated_Json(p)); err != nil {
			return err
		}

		// Register socket RPC with JSON suffix
		if err := initializer.RegisterRpc(RpcSocketId_RPC_SOCKET_ID_AUCTIONS_FOLLOW.String()+"_Json", rpcAuctionsFollow_Json(p)); err != nil {
			return err
		}

	case SystemTypeStreaks:
		// Register Streaks system JSON RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_STREAKS_LIST.String()+"_Json", rpcStreaksList_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_STREAKS_UPDATE.String()+"_Json", rpcStreaksUpdate_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_STREAKS_CLAIM.String()+"_Json", rpcStreaksClaim_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_STREAKS_RESET.String()+"_Json", rpcStreaksReset_Json(p)); err != nil {
			return err
		}

	case SystemTypeProgression:
		// Register Progression system JSON RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_PROGRESSIONS_GET.String()+"_Json", rpcProgressionsGet_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_PROGRESSIONS_PURCHASE.String()+"_Json", rpcProgressionsPurchase_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_PROGRESSIONS_UPDATE.String()+"_Json", rpcProgressionsUpdate_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_PROGRESSIONS_RESET.String()+"_Json", rpcProgressionsReset_Json(p)); err != nil {
			return err
		}

	case SystemTypeTeams:
		// Register Teams system JSON RPCs
		if err := initializer.RegisterRpc(RpcId_RPC_ID_TEAMS_CREATE.String()+"_Json", rpcTeamsCreate_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_TEAMS_LIST.String()+"_Json", rpcTeamsList_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_TEAMS_SEARCH.String()+"_Json", rpcTeamsSearch_Json(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc(RpcId_RPC_ID_TEAMS_WRITE_CHAT_MESSAGE.String()+"_Json", rpcTeamsWriteChatMessage_Json(p)); err != nil {
			return err
		}

	// Add other system types as needed...

	default:
		// Unknown system type, no RPCs to register
	}

	return nil
}

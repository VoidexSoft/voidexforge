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
		// Register additional achievement RPCs using the same RPC IDs with different endpoints
		if err := initializer.RegisterRpc("achievements_list", rpcAchievementsList(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc("achievements_progress", rpcAchievementsProgress(p)); err != nil {
			return err
		}
		if err := initializer.RegisterRpc("achievement_details", rpcAchievementDetails(p)); err != nil {
			return err
		}

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

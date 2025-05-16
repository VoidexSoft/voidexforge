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
		// Create achievements system instance
		// Would implement with NewNakamaAchievementsSystem(achievementsConfig)
		// For now, using a placeholder
		logger.Warn("Achievements system not fully implemented yet")

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
		// Create stats system instance

	case SystemTypeTeams:
		teamsConfig := &TeamsConfig{}
		if err := json.Unmarshal(configBytes, teamsConfig); err != nil {
			logger.Error("Failed to parse Teams system config: %v", err)
			return err
		}
		// Create teams system instance

	case SystemTypeTutorials:
		tutorialsConfig := &TutorialsConfig{}
		if err := json.Unmarshal(configBytes, tutorialsConfig); err != nil {
			logger.Error("Failed to parse Tutorials system config: %v", err)
			return err
		}
		// Create tutorials system instance

	case SystemTypeUnlockables:
		unlockablesConfig := &UnlockablesConfig{}
		if err := json.Unmarshal(configBytes, unlockablesConfig); err != nil {
			logger.Error("Failed to parse Unlockables system config: %v", err)
			return err
		}
		// Create unlockables system instance

	case SystemTypeEventLeaderboards:
		eventLeaderboardsConfig := &EventLeaderboardsConfig{}
		if err := json.Unmarshal(configBytes, eventLeaderboardsConfig); err != nil {
			logger.Error("Failed to parse EventLeaderboards system config: %v", err)
			return err
		}
		// Create event leaderboards system instance

	case SystemTypeProgression:
		progressionConfig := &ProgressionConfig{}
		if err := json.Unmarshal(configBytes, progressionConfig); err != nil {
			logger.Error("Failed to parse Progression system config: %v", err)
			return err
		}
		// Create progression system instance

	case SystemTypeIncentives:
		incentivesConfig := &IncentivesConfig{}
		if err := json.Unmarshal(configBytes, incentivesConfig); err != nil {
			logger.Error("Failed to parse Incentives system config: %v", err)
			return err
		}
		// Create incentives system instance

	case SystemTypeAuctions:
		auctionsConfig := &AuctionsConfig{}
		if err := json.Unmarshal(configBytes, auctionsConfig); err != nil {
			logger.Error("Failed to parse Auctions system config: %v", err)
			return err
		}
		// Create auctions system instance

	case SystemTypeStreaks:
		streaksConfig := &StreaksConfig{}
		if err := json.Unmarshal(configBytes, streaksConfig); err != nil {
			logger.Error("Failed to parse Streaks system config: %v", err)
			return err
		}
		// Create streaks system instance

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

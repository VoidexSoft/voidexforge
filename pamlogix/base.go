package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"

	"github.com/heroiclabs/nakama-common/api"
	"github.com/heroiclabs/nakama-common/runtime"
)

var (
	ErrInternal           = runtime.NewError("internal error occurred", 13) // INTERNAL
	ErrBadInput           = runtime.NewError("bad input", 3)                // INVALID_ARGUMENT
	ErrFileNotFound       = runtime.NewError("file not found", 3)
	ErrNoSessionUser      = runtime.NewError("no user ID in session", 3)       // INVALID_ARGUMENT
	ErrNoSessionID        = runtime.NewError("no session ID in session", 3)    // INVALID_ARGUMENT
	ErrNoSessionUsername  = runtime.NewError("no username in session", 3)      // INVALID_ARGUMENT
	ErrPayloadDecode      = runtime.NewError("cannot decode json", 13)         // INTERNAL
	ErrPayloadEmpty       = runtime.NewError("payload should not be empty", 3) // INVALID_ARGUMENT
	ErrPayloadEncode      = runtime.NewError("cannot encode json", 13)         // INTERNAL
	ErrPayloadInvalid     = runtime.NewError("payload is invalid", 3)          // INVALID_ARGUMENT
	ErrSessionUser        = runtime.NewError("user ID in session", 3)          // INVALID_ARGUMENT
	ErrSystemNotAvailable = runtime.NewError("system not available", 13)       // INTERNAL
	ErrSystemNotFound     = runtime.NewError("system not found", 13)           // INTERNAL
)

// The BaseSystem provides various small features which aren't large enough to be in their own gameplay systems.
type BaseSystem interface {
	System

	// RateApp uses the SMTP configuration to receive feedback from players via email.
	RateApp(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, username string, score uint32, message string) (err error)

	// SetDevicePrefs sets push notification tokens on a user's account so push messages can be received.
	SetDevicePrefs(ctx context.Context, logger runtime.Logger, db *sql.DB, userID, deviceID, pushTokenAndroid, pushTokenIos string, preferences map[string]bool) (err error)

	// Sync processes an operation to update the server with offline state changes.
	Sync(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, req *SyncRequest) (resp *SyncResponse, err error)
}

// BaseSystemConfig is the data definition for the BaseSystem type.
type BaseSystemConfig struct {
	RateAppSmtpAddr          string `json:"rate_app_smtp_addr,omitempty"`            // "smtp.gmail.com"
	RateAppSmtpUsername      string `json:"rate_app_smtp_username,omitempty"`        // "email@domain"
	RateAppSmtpPassword      string `json:"rate_app_smtp_password,omitempty"`        // "password"
	RateAppSmtpEmailFrom     string `json:"rate_app_smtp_email_from,omitempty"`      // "gamename-server@mmygamecompany.com"
	RateAppSmtpEmailFromName string `json:"rate_app_smtp_email_from_name,omitempty"` // My Game Company
	RateAppSmtpEmailSubject  string `json:"rate_app_smtp_email_subject,omitempty"`   // "RateApp Feedback"
	RateAppSmtpEmailTo       string `json:"rate_app_smtp_email_to,omitempty"`        // "gamename-rateapp@mygamecompany.com"
	RateAppSmtpPort          int    `json:"rate_app_smtp_port,omitempty"`            // 587

	RateAppTemplate string `json:"rate_app_template"` // HTML email template
}

type AfterAuthenticateFn func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, session *api.Session) error

type CollectionResolverFn func(ctx context.Context, systemType SystemType, collection string) (string, error)

// Pamlogix provides a type which combines all gameplay systems.
type Pamlogix interface {
	// SetPersonalizer is deprecated in favor of AddPersonalizer function to compose a chain of configuration personalization.
	SetPersonalizer(Personalizer)
	AddPersonalizer(personalizer Personalizer)

	AddPublisher(publisher Publisher)

	SetAfterAuthenticate(fn AfterAuthenticateFn)

	// SetCollectionResolver sets a function that may change the storage collection target for Pamlogix systems. Not typically used.
	SetCollectionResolver(fn CollectionResolverFn)

	GetAchievementsSystem() AchievementsSystem
	GetBaseSystem() BaseSystem
	GetEconomySystem() EconomySystem
	GetEnergySystem() EnergySystem
	GetInventorySystem() InventorySystem
	GetLeaderboardsSystem() LeaderboardsSystem
	GetStatsSystem() StatsSystem
	GetTeamsSystem() TeamsSystem
	GetTutorialsSystem() TutorialsSystem
	GetUnlockablesSystem() UnlockablesSystem
	GetEventLeaderboardsSystem() EventLeaderboardsSystem
	GetProgressionSystem() ProgressionSystem
	GetIncentivesSystem() IncentivesSystem
	GetAuctionsSystem() AuctionsSystem
	GetStreaksSystem() StreaksSystem
}

// The SystemType identifies each of the gameplay systems.
type SystemType uint

const (
	SystemTypeUnknown SystemType = iota
	SystemTypeBase
	SystemTypeEnergy
	SystemTypeUnlockables
	SystemTypeTutorials
	SystemTypeLeaderboards
	SystemTypeStats
	SystemTypeTeams
	SystemTypeInventory
	SystemTypeAchievements
	SystemTypeEconomy
	SystemTypeEventLeaderboards
	SystemTypeProgression
	SystemTypeIncentives
	SystemTypeAuctions
	SystemTypeStreaks
)

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

// pamlogixImpl implements the Pamlogix interface
type pamlogixImpl struct {
	personalizers      []Personalizer
	publishers         []Publisher
	afterAuthenticate  AfterAuthenticateFn
	collectionResolver CollectionResolverFn

	// Store systems in a map by type
	systems map[SystemType]System
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
		// Implement all economy RPCs here...

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

// RPC handler function placeholders
func rpcAchievementsClaim(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		// Implementation would go here
		return "", runtime.NewError("not implemented", 12) // UNIMPLEMENTED
	}
}

func rpcAchievementsGet(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		// Implementation would go here
		return "", runtime.NewError("not implemented", 12) // UNIMPLEMENTED
	}
}

func rpcAchievementsUpdate(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		// Implementation would go here
		return "", runtime.NewError("not implemented", 12) // UNIMPLEMENTED
	}
}

func rpcBaseRateApp(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		// Implementation would go here
		return "", runtime.NewError("not implemented", 12) // UNIMPLEMENTED
	}
}

func rpcBaseSetDevicePrefs(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		// Implementation would go here
		return "", runtime.NewError("not implemented", 12) // UNIMPLEMENTED
	}
}

func rpcBaseSync(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		// Implementation would go here
		return "", runtime.NewError("not implemented", 12) // UNIMPLEMENTED
	}
}

func rpcEnergyGet(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		energySystem := p.GetEnergySystem()
		if energySystem == nil {
			return "", runtime.NewError("energy system not available", 12) // UNIMPLEMENTED
		}

		userId, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userId == "" {
			return "", runtime.NewError("user id not found in context", 3) // INVALID_ARGUMENT
		}

		energies, err := energySystem.Get(ctx, logger, nk, userId)
		if err != nil {
			return "", err
		}

		data, err := json.Marshal(energies)
		if err != nil {
			logger.Error("Failed to marshal energies: %v", err)
			return "", runtime.NewError("failed to marshal energies", 13) // INTERNAL
		}

		return string(data), nil
	}
}

func rpcEnergySpend(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		// Implementation would go here
		return "", runtime.NewError("not implemented", 12) // UNIMPLEMENTED
	}
}

func rpcEnergyGrant(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		// Implementation would go here
		return "", runtime.NewError("not implemented", 12) // UNIMPLEMENTED
	}
}

func rpcInventoryList(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		// Implementation would go here
		return "", runtime.NewError("not implemented", 12) // UNIMPLEMENTED
	}
}

func rpcInventoryListInventory(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		// Implementation would go here
		return "", runtime.NewError("not implemented", 12) // UNIMPLEMENTED
	}
}

func rpcInventoryConsume(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		// Implementation would go here
		return "", runtime.NewError("not implemented", 12) // UNIMPLEMENTED
	}
}

func rpcInventoryGrant(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		// Implementation would go here
		return "", runtime.NewError("not implemented", 12) // UNIMPLEMENTED
	}
}

func rpcInventoryUpdate(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		// Implementation would go here
		return "", runtime.NewError("not implemented", 12) // UNIMPLEMENTED
	}
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

// The SystemConfig describes the configuration that each gameplay system must use to configure itself.
type SystemConfig interface {
	// GetType returns the runtime type of the gameplay system.
	GetType() SystemType

	// GetConfigFile returns the configuration file used for the data definitions in the gameplay system.
	GetConfigFile() string

	// GetRegister returns true if the gameplay system's RPCs should be registered with the game server.
	GetRegister() bool

	// GetExtra returns the extra parameter used to configure the gameplay system.
	GetExtra() any
}

var _ SystemConfig = &systemConfig{}

type systemConfig struct {
	systemType SystemType
	configFile string
	register   bool

	extra any
}

func (sc *systemConfig) GetType() SystemType {
	return sc.systemType
}
func (sc *systemConfig) GetConfigFile() string {
	return sc.configFile
}
func (sc *systemConfig) GetRegister() bool {
	return sc.register
}
func (sc *systemConfig) GetExtra() any {
	return sc.extra
}

// OnReward is a function which can be used by each gameplay system to provide an override reward.
type OnReward[T any] func(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, sourceID string, source T, rewardConfig *EconomyConfigReward, reward *Reward) (*Reward, error)

// A System is a base type for a gameplay system.
type System interface {
	// GetType provides the runtime type of the gameplay system.
	GetType() SystemType

	// GetConfig returns the configuration type of the gameplay system.
	GetConfig() any
}

// UsernameOverrideFn can be used to provide a different username generation strategy from the default in Nakama server.
// Requested username indicates what the username would otherwise be set to, if the incoming request specified a value.
// The function is always expected to return a value, and returning "" defers to Nakama's built-in behaviour.
type UsernameOverrideFn func(requestedUsername string) string

// WithAchievementsSystem configures an AchievementsSystem type and optionally registers its RPCs with the game server.
func WithAchievementsSystem(configFile string, register bool) SystemConfig {
	return &systemConfig{
		systemType: SystemTypeAchievements,
		configFile: configFile,
		register:   register,
	}
}

// WithBaseSystem configures a BaseSystem type and optionally registers its RPCs with the game server.
func WithBaseSystem(configFile string, register bool, usernameOverride ...UsernameOverrideFn) SystemConfig {
	return &systemConfig{
		systemType: SystemTypeBase,
		configFile: configFile,
		register:   register,

		extra: usernameOverride,
	}
}

// WithEconomySystem configures an EconomySystem type and optionally registers its RPCs with the game server.
func WithEconomySystem(configFile string, register bool, ironSrcPrivKey ...string) SystemConfig {
	return &systemConfig{
		systemType: SystemTypeEconomy,
		configFile: configFile,
		register:   register,

		extra: ironSrcPrivKey,
	}
}

// WithEnergySystem configures an EnergySystem type and optionally registers its RPCs with the game server.
func WithEnergySystem(configFile string, register bool) SystemConfig {
	return &systemConfig{
		systemType: SystemTypeEnergy,
		configFile: configFile,
		register:   register,
	}
}

// WithInventorySystem configures an InventorySystem type and optionally registers its RPCs with the game server.
func WithInventorySystem(configFile string, register bool) SystemConfig {
	return &systemConfig{
		systemType: SystemTypeInventory,
		configFile: configFile,
		register:   register,
	}
}

// WithLeaderboardsSystem configures a LeaderboardsSystem type.
func WithLeaderboardsSystem(configFile string, register bool, validateWriteScore ...ValidateWriteScoreFn) SystemConfig {
	return &systemConfig{
		systemType: SystemTypeLeaderboards,
		configFile: configFile,
		register:   register,

		extra: validateWriteScore,
	}
}

// WithStatsSystem configures a StatsSystem type and optionally registers its RPCs with the game server.
func WithStatsSystem(configFile string, register bool) SystemConfig {
	return &systemConfig{
		systemType: SystemTypeStats,
		configFile: configFile,
		register:   register,
	}
}

// WithTeamsSystem configures a TeamsSystem type and optionally registers its RPCs with the game server.
func WithTeamsSystem(configFile string, register bool, validateCreateTeam ...ValidateCreateTeamFn) SystemConfig {
	return &systemConfig{
		systemType: SystemTypeTeams,
		configFile: configFile,
		register:   register,

		extra: validateCreateTeam,
	}
}

// WithTutorialsSystem configures a TutorialsSystem type and optionally registers its RPCs with the game server.
func WithTutorialsSystem(configFile string, register bool) SystemConfig {
	return &systemConfig{
		systemType: SystemTypeTutorials,
		configFile: configFile,
		register:   register,
	}
}

// WithUnlockablesSystem configures an UnlockablesSystem type and optionally registers its RPCs with the game server.
func WithUnlockablesSystem(configFile string, register bool) SystemConfig {
	return &systemConfig{
		systemType: SystemTypeUnlockables,
		configFile: configFile,
		register:   register,
	}
}

// WithEventLeaderboardsSystem configures an EventLeaderboardsSystem type and optionally registers its RPCs with the game server.
func WithEventLeaderboardsSystem(configFile string, register bool) SystemConfig {
	return &systemConfig{
		systemType: SystemTypeEventLeaderboards,
		configFile: configFile,
		register:   register,
	}
}

// WithProgressionSystem configures a ProgressionSystem type and optionally registers its RPCs with the game server.
func WithProgressionSystem(configFile string, register bool) SystemConfig {
	return &systemConfig{
		systemType: SystemTypeProgression,
		configFile: configFile,
		register:   register,
	}
}

// WithIncentivesSystem configures a IncentivesSystem type and optionally registers its RPCs with the game server.
func WithIncentivesSystem(configFile string, register bool) SystemConfig {
	return &systemConfig{
		systemType: SystemTypeIncentives,
		configFile: configFile,
		register:   register,
	}
}

// WithAuctionsSystem configures a AuctionsSystem type and optionally registers its RPCs with the game server.
func WithAuctionsSystem(configFile string, register bool) SystemConfig {
	return &systemConfig{
		systemType: SystemTypeAuctions,
		configFile: configFile,
		register:   register,
	}
}

// WithStreaksSystem configures a StreaksSystem type and optionally registers its RPCs with the game server.
func WithStreaksSystem(configFile string, register bool) SystemConfig {
	return &systemConfig{
		systemType: SystemTypeStreaks,
		configFile: configFile,
		register:   register,
	}
}

// UnregisterRpc clears the implementation of one or more RPCs registered in Nakama by Pamlogix gameplay systems with a
// no-op version (http response 404). This is useful to remove individual RPCs which you do not want to be callable by
// game clients:
//
//	pamlogix.UnregisterRpc(initializer, pamlogix.RpcId_RPC_ID_ECONOMY_GRANT, pamlogix.RpcId_RPC_ID_INVENTORY_GRANT)
//
// The behaviour of `initializer.RegisterRpc` in Nakama is last registration wins. It's recommended to use UnregisterRpc
// only after `pamlogix.Init` has been executed.
func UnregisterRpc(initializer runtime.Initializer, ids ...RpcId) error {
	noopFn := func(context.Context, runtime.Logger, *sql.DB, runtime.NakamaModule, string) (string, error) {
		return "", runtime.NewError("not found", 12) // GRPC - UNIMPLEMENTED
	}
	for _, id := range ids {
		if err := initializer.RegisterRpc(id.String(), noopFn); err != nil {
			return err
		}
	}
	return nil
}

// UnregisterDebugRpc clears the implementation of ALL debug RPCs registered in Nakama by Pamlogix gameplay systems with
// a no-op version (http response 404). This is useful to remove debug RPCs if you do not want them to be callable
// by game clients:
//
//	pamlogix.UnregisterDebugRpc(initializer)
//
// The behaviour of `initializer.RegisterRpc` in Nakama is last registration wins. It's recommended to use
// UnregisterDebugRpc only after `pamlogix.Init` has been executed.
func UnregisterDebugRpc(initializer runtime.Initializer) error {
	ids := []RpcId{
		RpcId_RPC_ID_EVENT_LEADERBOARD_DEBUG_FILL,
		RpcId_RPC_ID_EVENT_LEADERBOARD_DEBUG_RANDOM_SCORES,
	}
	return UnregisterRpc(initializer, ids...)
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

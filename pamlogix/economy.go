package pamlogix

import (
	"context"
	"database/sql"

	"github.com/heroiclabs/nakama-common/api"
	"github.com/heroiclabs/nakama-common/runtime"
)

var (
	ErrEconomyNoItem            = runtime.NewError("item not found", INVALID_ARGUMENT_ERROR_CODE)                        // INVALID_ARGUMENT
	ErrEconomyItemUnavailable   = runtime.NewError("item unavailable", INVALID_ARGUMENT_ERROR_CODE)                      // INVALID_ARGUMENT
	ErrEconomyNoSku             = runtime.NewError("sku not found", INVALID_ARGUMENT_ERROR_CODE)                         // INVALID_ARGUMENT
	ErrEconomySkuInvalid        = runtime.NewError("invalid sku", INVALID_ARGUMENT_ERROR_CODE)                           // INVALID_ARGUMENT
	ErrEconomyNotEnoughCurrency = runtime.NewError("not enough currency for purchase", INVALID_ARGUMENT_ERROR_CODE)      // INVALID_ARGUMENT
	ErrEconomyNotEnoughItem     = runtime.NewError("not enough item", INVALID_ARGUMENT_ERROR_CODE)                       // INVALID_ARGUMENT
	ErrEconomyReceiptInvalid    = runtime.NewError("invalid receipt", INVALID_ARGUMENT_ERROR_CODE)                       // INVALID_ARGUMENT
	ErrEconomyReceiptDuplicate  = runtime.NewError("duplicate receipt", INVALID_ARGUMENT_ERROR_CODE)                     // INVALID_ARGUMENT
	ErrEconomyReceiptMismatch   = runtime.NewError("mismatched product receipt", INVALID_ARGUMENT_ERROR_CODE)            // INVALID_ARGUMENT
	ErrEconomyNoPlacement       = runtime.NewError("placement not found", INVALID_ARGUMENT_ERROR_CODE)                   // INVALID_ARGUMENT
	ErrEconomyNoDonation        = runtime.NewError("donation not found", INVALID_ARGUMENT_ERROR_CODE)                    // INVALID_ARGUMENT
	ErrEconomyMaxDonation       = runtime.NewError("donation maximum contribution reached", INVALID_ARGUMENT_ERROR_CODE) // INVALID_ARGUMENT
	ErrEconomyClaimedDonation   = runtime.NewError("donation already claimed", INVALID_ARGUMENT_ERROR_CODE)              // INVALID_ARGUMENT

	ErrInventoryNotInitialized = runtime.NewError("inventory not initialized for batch", INTERNAL_ERROR_CODE) // INTERNAL
	ErrItemsNotConsumable      = runtime.NewError("items not consumable", INVALID_ARGUMENT_ERROR_CODE)        // INVALID_ARGUMENT
	ErrItemsInsufficient       = runtime.NewError("insufficient items", FAILED_PRECONDITION_ERROR_CODE)       // FAILED_PRECONDITION
	ErrCurrencyInsufficient    = runtime.NewError("insufficient currency", FAILED_PRECONDITION_ERROR_CODE)    // FAILED_PRECONDITION
)

// EconomyConfig is the data definition for the EconomySystem type.
type EconomyConfig struct {
	InitializeUser    *EconomyConfigInitializeUser       `json:"initialize_user,omitempty"`
	Donations         map[string]*EconomyConfigDonation  `json:"donations,omitempty"`
	StoreItems        map[string]*EconomyConfigStoreItem `json:"store_items,omitempty"`
	Placements        map[string]*EconomyConfigPlacement `json:"placements,omitempty"`
	AllowFakeReceipts bool                               `json:"allow_fake_receipts,omitempty"`
}

type EconomyConfigDonation struct {
	Cost                     *EconomyConfigDonationCost `json:"cost,omitempty"`
	Count                    int64                      `json:"count,omitempty"`
	Description              string                     `json:"description,omitempty"`
	DurationSec              int64                      `json:"duration_sec,omitempty"`
	MaxCount                 int64                      `json:"max_count,omitempty"`
	Name                     string                     `json:"name,omitempty"`
	RecipientReward          *EconomyConfigReward       `json:"recipient_reward,omitempty"`
	ContributorReward        *EconomyConfigReward       `json:"contributor_reward,omitempty"`
	UserContributionMaxCount int64                      `json:"user_contribution_max_count,omitempty"`
	AdditionalProperties     map[string]string          `json:"additional_properties,omitempty"`
}

type EconomyConfigDonationCost struct {
	Currencies map[string]int64 `json:"currencies,omitempty"`
	Items      map[string]int64 `json:"items,omitempty"`
}

type EconomyConfigInitializeUser struct {
	Currencies map[string]int64 `json:"currencies,omitempty"`
	Items      map[string]int64 `json:"items,omitempty"`
}

type EconomyConfigPlacement struct {
	Reward               *EconomyConfigReward `json:"reward,omitempty"`
	AdditionalProperties map[string]string    `json:"additional_properties,omitempty"`
}

type EconomyConfigReward struct {
	Guaranteed     *EconomyConfigRewardContents   `json:"guaranteed,omitempty"`
	Weighted       []*EconomyConfigRewardContents `json:"weighted,omitempty"`
	MaxRolls       int64                          `json:"max_rolls,omitempty"`
	MaxRepeatRolls int64                          `json:"max_repeat_rolls,omitempty"`
	TotalWeight    int64                          `json:"total_weight,omitempty"`
}

type EconomyConfigRewardContents struct {
	Items           map[string]*EconomyConfigRewardItem     `json:"items,omitempty"`
	ItemSets        []*EconomyConfigRewardItemSet           `json:"item_sets,omitempty"`
	Currencies      map[string]*EconomyConfigRewardCurrency `json:"currencies,omitempty"`
	Energies        map[string]*EconomyConfigRewardEnergy   `json:"energies,omitempty"`
	EnergyModifiers []*EconomyConfigRewardEnergyModifier    `json:"energy_modifiers,omitempty"`
	RewardModifiers []*EconomyConfigRewardRewardModifier    `json:"reward_modifiers,omitempty"`
	Weight          int64                                   `json:"weight,omitempty"`
}

type EconomyConfigRewardCurrency struct {
	EconomyConfigRewardRangeInt64
}

type EconomyConfigRewardEnergy struct {
	EconomyConfigRewardRangeInt32
}

type EconomyConfigRewardEnergyModifier struct {
	Id          string                          `json:"id,omitempty"`
	Operator    string                          `json:"operator,omitempty"`
	Value       *EconomyConfigRewardRangeInt64  `json:"value,omitempty"`
	DurationSec *EconomyConfigRewardRangeUInt64 `json:"duration_sec,omitempty"`
}

type EconomyConfigRewardItem struct {
	EconomyConfigRewardRangeInt64
	StringProperties  map[string]*EconomyConfigRewardStringProperty `json:"string_properties,omitempty"`
	NumericProperties map[string]*EconomyConfigRewardRangeFloat64   `json:"numeric_properties,omitempty"`
}

type EconomyConfigRewardItemSet struct {
	EconomyConfigRewardRangeInt64

	MaxRepeats int64    `json:"max_repeats,omitempty"`
	Set        []string `json:"set,omitempty"`
}

type EconomyConfigRewardRangeInt32 struct {
	Min      int32 `json:"min,omitempty"`
	Max      int32 `json:"max,omitempty"`
	Multiple int32 `json:"multiple,omitempty"`
}

type EconomyConfigRewardRangeInt64 struct {
	Min      int64 `json:"min,omitempty"`
	Max      int64 `json:"max,omitempty"`
	Multiple int64 `json:"multiple,omitempty"`
}

type EconomyConfigRewardRangeUInt64 struct {
	Min      uint64 `json:"min,omitempty"`
	Max      uint64 `json:"max,omitempty"`
	Multiple uint64 `json:"multiple,omitempty"`
}

type EconomyConfigRewardRangeFloat64 struct {
	Min      float64 `json:"min,omitempty"`
	Max      float64 `json:"max,omitempty"`
	Multiple float64 `json:"multiple,omitempty"`
}

type EconomyConfigRewardRewardModifier struct {
	Id          string                          `json:"id,omitempty"`
	Type        string                          `json:"type,omitempty"`
	Operator    string                          `json:"operator,omitempty"`
	Value       *EconomyConfigRewardRangeInt64  `json:"value,omitempty"`
	DurationSec *EconomyConfigRewardRangeUInt64 `json:"duration_sec,omitempty"`
}

type EconomyConfigRewardStringProperty struct {
	TotalWeight int64                                               `json:"total_weight,omitempty"`
	Options     map[string]*EconomyConfigRewardStringPropertyOption `json:"options,omitempty"`
}

type EconomyConfigRewardStringPropertyOption struct {
	Weight int64 `json:"weight,omitempty"`
}

type EconomyConfigStoreItem struct {
	Category             string                      `json:"category,omitempty"`
	Cost                 *EconomyConfigStoreItemCost `json:"cost,omitempty"`
	Description          string                      `json:"description,omitempty"`
	Name                 string                      `json:"name,omitempty"`
	Reward               *EconomyConfigReward        `json:"reward,omitempty"`
	AdditionalProperties map[string]string           `json:"additional_properties,omitempty"`
	Disabled             bool                        `json:"disabled,omitempty"`
	Unavailable          bool                        `json:"unavailable,omitempty"`
}

type EconomyConfigStoreItemCost struct {
	Currencies map[string]int64 `json:"currencies,omitempty"`
	Sku        string           `json:"sku,omitempty"`
}

// EconomyPlacementInfo contains information about a placement instance.
type EconomyPlacementInfo struct {
	// Placement configuration.
	Placement *EconomyConfigPlacement `json:"placement,omitempty"`
	// Metadata, if any was set when the placement was started.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// The EconomySystem is the foundation of a game's economy.
//
// It provides functionality for 4 different reward types: basic, gacha, weighted table, and custom. These rolled
// rewards are available to generate in all other gameplay systems and can be generated manually as well.
type EconomySystem interface {
	System

	// RewardCreate prepares a new reward configuration to be filled in and used later.
	RewardCreate() (rewardConfig *EconomyConfigReward)

	// RewardConvert transforms a wire representation of a reward into an equivalent configuration representation.
	RewardConvert(contents *AvailableRewards) (rewardConfig *EconomyConfigReward)

	// RewardRoll takes a reward configuration and rolls an actual reward from it, applying all appropriate rules.
	RewardRoll(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, rewardConfig *EconomyConfigReward) (reward *Reward, err error)

	// RewardGrant updates a user's economy, inventory, and/or energy models with the contents of a rolled reward.
	RewardGrant(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, reward *Reward, metadata map[string]interface{}, ignoreLimits bool) (newItems map[string]*InventoryItem, updatedItems map[string]*InventoryItem, notGrantedItemIDs map[string]int64, err error)

	// DonationClaim will claim donation rewards for a user and the given donation IDs.
	DonationClaim(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, donationClaims map[string]*EconomyDonationClaimRequestDetails) (donationsList *EconomyDonationsList, err error)

	// DonationGet will get all donations for the given list of user IDs.
	DonationGet(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userIDs []string) (donationsList *EconomyDonationsByUserList, err error)

	// DonationGive will contribute to a particular donation for a user ID.
	DonationGive(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, donationID, fromUserID string) (donation *EconomyDonation, updatedWallet map[string]int64, updatedInventory *Inventory, rewardModifiers []*ActiveRewardModifier, contributorReward *Reward, timestamp int64, err error)

	// DonationRequest will create a donation request for a given donation ID and user ID.
	DonationRequest(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, donationID string) (donation *EconomyDonation, success bool, err error)

	// List will get the defined store items and placements within the economy system.
	List(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (storeItems map[string]*EconomyConfigStoreItem, placements map[string]*EconomyConfigPlacement, rewardModifiers []*ActiveRewardModifier, timestamp int64, err error)

	// Grant will add currencies, and reward modifiers to a user's economy by ID.
	Grant(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, currencies map[string]int64, items map[string]int64, modifiers []*RewardModifier, walletMetadata map[string]interface{}) (updatedWallet map[string]int64, rewardModifiers []*ActiveRewardModifier, timestamp int64, err error)

	// UnmarshalWallet unmarshals and returns the account's wallet as a map[string]int64.
	UnmarshalWallet(account *api.Account) (wallet map[string]int64, err error)

	// PurchaseIntent will create a purchase intent for a particular store item for a user ID.
	PurchaseIntent(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, itemID string, store EconomyStoreType, sku string) (err error)

	// PurchaseItem will validate a purchase and give the user ID the appropriate rewards.
	PurchaseItem(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, userID, itemID string, store EconomyStoreType, receipt string) (updatedWallet map[string]int64, updatedInventory *Inventory, reward *Reward, isSandboxPurchase bool, err error)

	// PurchaseRestore will process a restore attempt for the given user, based on a set of restore receipts.
	PurchaseRestore(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, store EconomyStoreType, receipts []string) (err error)

	// PlacementStatus will get the status of a specified placement.
	PlacementStatus(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, rewardID, placementID string, retryCount int) (resp *EconomyPlacementStatus, err error)

	// PlacementStart will indicate that a user ID has begun viewing an ad placement.
	PlacementStart(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, placementID string, metadata map[string]string) (resp *EconomyPlacementStatus, err error)

	// PlacementSuccess will indicate that the user ID has successfully viewed an ad placement and provide the appropriate reward.
	PlacementSuccess(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, rewardID, placementID string) (reward *Reward, placementMetadata map[string]string, err error)

	// PlacementFail will indicate that the user ID has failed to successfully view the ad placement.
	PlacementFail(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, rewardID, placementID string) (placementMetadata map[string]string, err error)

	// SetOnDonationClaimReward sets a custom reward function which will run after a donation's reward is rolled.
	SetOnDonationClaimReward(fn OnReward[*EconomyConfigDonation])

	// SetOnDonationContributorReward sets a custom reward function which will run after a donation's sender reward is rolled.
	SetOnDonationContributorReward(fn OnReward[*EconomyConfigDonation])

	// SetOnPlacementReward sets a custom reward function which will run after a placement's reward is rolled.
	SetOnPlacementReward(fn OnReward[*EconomyPlacementInfo])

	// SetOnStoreItemReward sets a custom reward function which will run after store item's reward is rolled.
	SetOnStoreItemReward(fn OnReward[*EconomyConfigStoreItem])
}

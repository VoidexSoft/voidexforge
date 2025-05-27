package pamlogix

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/heroiclabs/nakama-common/runtime"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	AuctionCollectionKey  = "auctions"
	AuctionIndexKey       = "auction_index"
	AuctionBidsKey        = "auction_bids"
	AuctionUserCreatedKey = "auction_user_created"
	AuctionUserBidsKey    = "auction_user_bids"
)

// AuctionsPamlogix implements the AuctionsSystem interface
type AuctionsPamlogix struct {
	config   *AuctionsConfig
	pamlogix Pamlogix

	onClaimBid           OnAuctionReward[*AuctionReward]
	onClaimCreated       OnAuctionReward[*AuctionBidAmount]
	onClaimCreatedFailed OnAuctionReward[*AuctionReward]
	onCancel             OnAuctionReward[*AuctionReward]
}

// NewNakamaAuctionsSystem creates a new auctions system instance
func NewNakamaAuctionsSystem(config *AuctionsConfig) AuctionsSystem {
	return &AuctionsPamlogix{
		config: config,
	}
}

// GetType returns the system type
func (a *AuctionsPamlogix) GetType() SystemType {
	return SystemTypeAuctions
}

// GetConfig returns the system configuration
func (a *AuctionsPamlogix) GetConfig() any {
	return a.config
}

// SetPamlogix sets the Pamlogix instance for this auctions system
func (a *AuctionsPamlogix) SetPamlogix(pl Pamlogix) {
	a.pamlogix = pl
}

// GetTemplates lists all available auction configurations that can be used to create auction listings
func (a *AuctionsPamlogix) GetTemplates(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string) (*AuctionTemplates, error) {
	templates := &AuctionTemplates{
		Templates: make(map[string]*AuctionTemplate),
	}

	for templateID, auctionConfig := range a.config.Auctions {
		template := &AuctionTemplate{
			Items:           auctionConfig.Items,
			ItemSets:        auctionConfig.ItemSets,
			Conditions:      make(map[string]*AuctionTemplateCondition),
			BidHistoryCount: int32(auctionConfig.BidHistoryCount),
		}

		for conditionID, condition := range auctionConfig.Conditions {
			templateCondition := &AuctionTemplateCondition{
				DurationSec:           condition.DurationSec,
				ExtensionThresholdSec: condition.ExtensionThresholdSec,
				ExtensionSec:          condition.ExtensionSec,
				ExtensionMaxSec:       condition.ExtensionMaxSec,
			}

			if condition.ListingCost != nil {
				templateCondition.ListingCost = &AuctionTemplateConditionListingCost{
					Currencies: condition.ListingCost.Currencies,
					Items:      condition.ListingCost.Items,
					Energies:   condition.ListingCost.Energies,
				}
			}

			if condition.BidStart != nil {
				templateCondition.BidStart = &AuctionBidAmount{
					Currencies: condition.BidStart.Currencies,
				}
			}

			if condition.BidIncrement != nil {
				templateCondition.BidIncrement = &AuctionTemplateConditionBidIncrement{
					Percentage: condition.BidIncrement.Percentage,
				}
				if condition.BidIncrement.Fixed != nil {
					templateCondition.BidIncrement.Fixed = &AuctionBidAmount{
						Currencies: condition.BidIncrement.Fixed.Currencies,
					}
				}
			}

			if condition.Fee != nil {
				templateCondition.Fee = &AuctionFee{
					Percentage: condition.Fee.Percentage,
				}
				if condition.Fee.Fixed != nil {
					templateCondition.Fee.Fixed = &AuctionBidAmount{
						Currencies: condition.Fee.Fixed.Currencies,
					}
				}
			}

			template.Conditions[conditionID] = templateCondition
		}

		templates.Templates[templateID] = template
	}

	return templates, nil
}

// List auctions based on provided criteria
func (a *AuctionsPamlogix) List(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, query string, sort []string, limit int, cursor string) (*AuctionList, error) {
	// Parse cursor for pagination
	var offset int
	if cursor != "" {
		var err error
		offset, err = strconv.Atoi(cursor)
		if err != nil {
			return nil, ErrBadInput
		}
	}

	// Read auction index to get list of active auctions
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: AuctionCollectionKey,
			Key:        AuctionIndexKey,
			UserID:     "",
		},
	})
	if err != nil {
		logger.Error("Failed to read auction index: %v", err)
		return nil, ErrInternal
	}

	var auctionIDs []string
	if len(objects) > 0 {
		var index map[string]bool
		if err := json.Unmarshal([]byte(objects[0].Value), &index); err != nil {
			logger.Error("Failed to unmarshal auction index: %v", err)
			return nil, ErrInternal
		}

		for auctionID := range index {
			auctionIDs = append(auctionIDs, auctionID)
		}
	}

	// Apply pagination
	if offset >= len(auctionIDs) {
		return &AuctionList{
			Auctions: []*Auction{},
			Cursor:   "",
		}, nil
	}

	end := offset + limit
	if end > len(auctionIDs) {
		end = len(auctionIDs)
	}

	paginatedIDs := auctionIDs[offset:end]

	// Read auction data
	var auctions []*Auction
	if len(paginatedIDs) > 0 {
		reads := make([]*runtime.StorageRead, len(paginatedIDs))
		for i, auctionID := range paginatedIDs {
			reads[i] = &runtime.StorageRead{
				Collection: AuctionCollectionKey,
				Key:        auctionID,
				UserID:     "",
			}
		}

		objects, err := nk.StorageRead(ctx, reads)
		if err != nil {
			logger.Error("Failed to read auctions: %v", err)
			return nil, ErrInternal
		}

		currentTime := time.Now().Unix()
		for _, obj := range objects {
			var auction Auction
			if err := json.Unmarshal([]byte(obj.Value), &auction); err != nil {
				logger.Error("Failed to unmarshal auction %s: %v", obj.Key, err)
				continue
			}

			// Update auction state based on current time
			a.updateAuctionState(&auction, currentTime, userID)
			auctions = append(auctions, &auction)
		}
	}

	// Determine next cursor
	var nextCursor string
	if end < len(auctionIDs) {
		nextCursor = strconv.Itoa(end)
	}

	return &AuctionList{
		Auctions: auctions,
		Cursor:   nextCursor,
	}, nil
}

// Bid on an active auction
func (a *AuctionsPamlogix) Bid(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, sessionID, auctionID, version string, bid *AuctionBidAmount, marshaler *protojson.MarshalOptions) (*Auction, error) {
	// Read current auction state
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: AuctionCollectionKey,
			Key:        auctionID,
			UserID:     "",
		},
	})
	if err != nil {
		logger.Error("Failed to read auction %s: %v", auctionID, err)
		return nil, ErrInternal
	}

	if len(objects) == 0 {
		return nil, ErrAuctionNotFound
	}

	var auction Auction
	if err := json.Unmarshal([]byte(objects[0].Value), &auction); err != nil {
		logger.Error("Failed to unmarshal auction %s: %v", auctionID, err)
		return nil, ErrInternal
	}

	// Check version
	if auction.Version != version {
		return nil, ErrAuctionVersionMismatch
	}

	currentTime := time.Now().Unix()
	a.updateAuctionState(&auction, currentTime, userID)

	// Validate bid
	if err := a.validateBid(&auction, userID, bid, currentTime); err != nil {
		return nil, err
	}

	// Check if user has sufficient funds using the economy system
	if err := a.checkUserFunds(ctx, logger, nk, userID, bid); err != nil {
		return nil, err
	}

	// Process the bid
	if err := a.processBid(ctx, logger, nk, &auction, userID, bid, currentTime, nil); err != nil {
		return nil, err
	}

	// Save updated auction
	if err := a.saveAuction(ctx, nk, &auction); err != nil {
		logger.Error("Failed to save auction after bid: %v", err)
		return nil, ErrInternal
	}

	// Add to user's bid auctions index
	if err := a.addToUserBidsIndex(ctx, nk, userID, auctionID); err != nil {
		logger.Error("Failed to add auction to user bids index: %v", err)
		// Don't return error as the bid was placed successfully
	}

	// Automatically follow the auction for the bidder to receive future updates
	a.followAuctionForUser(ctx, logger, nk, userID, sessionID, auctionID)

	// Send real-time notification to followers
	a.sendBidNotification(ctx, logger, nk, &auction, sessionID)

	return &auction, nil
}

// ClaimBid claims a completed auction as the successful bidder
func (a *AuctionsPamlogix) ClaimBid(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, auctionID string) (*AuctionClaimBid, error) {
	// Read auction
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: AuctionCollectionKey,
			Key:        auctionID,
			UserID:     "",
		},
	})
	if err != nil {
		logger.Error("Failed to read auction %s: %v", auctionID, err)
		return nil, ErrInternal
	}

	if len(objects) == 0 {
		return nil, ErrAuctionNotFound
	}

	var auction Auction
	if err := json.Unmarshal([]byte(objects[0].Value), &auction); err != nil {
		logger.Error("Failed to unmarshal auction %s: %v", auctionID, err)
		return nil, ErrInternal
	}

	currentTime := time.Now().Unix()
	a.updateAuctionState(&auction, currentTime, userID)

	// Validate claim
	if !auction.HasEnded {
		return nil, ErrAuctionNotStarted
	}

	if auction.Bid == nil || auction.Bid.UserId != userID {
		return nil, ErrAuctionCannotClaim
	}

	if auction.WinnerClaimSec > 0 {
		return nil, ErrAuctionCannotClaim
	}

	// Mark as claimed
	auction.WinnerClaimSec = currentTime
	auction.CanClaim = false

	// Grant items to winner
	reward := auction.Reward
	if a.onClaimBid != nil {
		customReward, err := a.onClaimBid(ctx, logger, nk, userID, auctionID, &auction, reward)
		if err != nil {
			logger.Error("Custom claim bid reward failed: %v", err)
			return nil, err
		}
		reward = customReward
	}

	// Save updated auction
	if err := a.saveAuction(ctx, nk, &auction); err != nil {
		logger.Error("Failed to save auction after claim: %v", err)
		return nil, ErrInternal
	}

	return &AuctionClaimBid{
		Auction: &auction,
		Reward:  reward,
	}, nil
}

// ClaimCreated claims a completed auction as the auction creator
func (a *AuctionsPamlogix) ClaimCreated(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, auctionID string) (*AuctionClaimCreated, error) {
	// Read auction
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: AuctionCollectionKey,
			Key:        auctionID,
			UserID:     "",
		},
	})
	if err != nil {
		logger.Error("Failed to read auction %s: %v", auctionID, err)
		return nil, ErrInternal
	}

	if len(objects) == 0 {
		return nil, ErrAuctionNotFound
	}

	var auction Auction
	if err := json.Unmarshal([]byte(objects[0].Value), &auction); err != nil {
		logger.Error("Failed to unmarshal auction %s: %v", auctionID, err)
		return nil, ErrInternal
	}

	currentTime := time.Now().Unix()
	a.updateAuctionState(&auction, currentTime, userID)

	// Validate claim
	if auction.UserId != userID {
		return nil, ErrAuctionCannotClaim
	}

	if !auction.HasEnded {
		return nil, ErrAuctionNotStarted
	}

	if auction.OwnerClaimSec > 0 {
		return nil, ErrAuctionCannotClaim
	}

	// Mark as claimed
	auction.OwnerClaimSec = currentTime
	auction.CanClaim = false

	var reward *AuctionBidAmount
	var fee *AuctionBidAmount
	var returnedItems []*InventoryItem

	if auction.Bid != nil {
		// Successful auction - calculate reward and fee
		reward = auction.Bid.Bid
		fee = a.calculateFee(auction.Bid.Bid, auction.Fee)

		if a.onClaimCreated != nil {
			customReward, err := a.onClaimCreated(ctx, logger, nk, userID, auctionID, &auction, reward)
			if err != nil {
				logger.Error("Custom claim created reward failed: %v", err)
				return nil, err
			}
			reward = customReward
		}
	} else {
		// Failed auction - return items
		returnedItems = auction.Reward.Items

		if a.onClaimCreatedFailed != nil {
			customReward, err := a.onClaimCreatedFailed(ctx, logger, nk, userID, auctionID, &auction, auction.Reward)
			if err != nil {
				logger.Error("Custom claim created failed reward failed: %v", err)
				return nil, err
			}
			returnedItems = customReward.Items
		}
	}

	// Save updated auction
	if err := a.saveAuction(ctx, nk, &auction); err != nil {
		logger.Error("Failed to save auction after claim: %v", err)
		return nil, ErrInternal
	}

	return &AuctionClaimCreated{
		Auction:       &auction,
		Reward:        reward,
		Fee:           fee,
		ReturnedItems: returnedItems,
	}, nil
}

// Cancel an active auction before it reaches its scheduled end time
func (a *AuctionsPamlogix) Cancel(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, auctionID string) (*AuctionCancel, error) {
	// Read auction
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: AuctionCollectionKey,
			Key:        auctionID,
			UserID:     "",
		},
	})
	if err != nil {
		logger.Error("Failed to read auction %s: %v", auctionID, err)
		return nil, ErrInternal
	}

	if len(objects) == 0 {
		return nil, ErrAuctionNotFound
	}

	var auction Auction
	if err := json.Unmarshal([]byte(objects[0].Value), &auction); err != nil {
		logger.Error("Failed to unmarshal auction %s: %v", auctionID, err)
		return nil, ErrInternal
	}

	currentTime := time.Now().Unix()
	a.updateAuctionState(&auction, currentTime, userID)

	// Validate cancellation
	if auction.UserId != userID {
		return nil, ErrAuctionCannotCancel
	}

	if !auction.CanCancel {
		return nil, ErrAuctionCannotCancel
	}

	// Cancel the auction
	auction.CancelTimeSec = currentTime
	auction.HasEnded = true
	auction.CanCancel = false
	auction.CanBid = false

	// Return bid to current bidder if any
	if auction.Bid != nil {
		if err := a.returnBidToUser(ctx, logger, nk, auction.Bid.UserId, auction.Bid.Bid); err != nil {
			logger.Error("Failed to return bid to user %s when cancelling auction: %v", auction.Bid.UserId, err)
			// Continue with cancellation despite error, but log it
		} else {
			logger.Info("Returned bid to user %s when cancelling auction %s", auction.Bid.UserId, auction.Id)
		}

		// Remove the auction from the bidder's bids index since the auction is cancelled
		if err := a.removeFromUserBidsIndex(ctx, nk, auction.Bid.UserId, auction.Id); err != nil {
			logger.Error("Failed to remove auction from bidder's index when cancelling: %v", err)
		}
	}

	// Return items to creator
	reward := auction.Reward
	if a.onCancel != nil {
		customReward, err := a.onCancel(ctx, logger, nk, userID, auctionID, &auction, reward)
		if err != nil {
			logger.Error("Custom cancel reward failed: %v", err)
			return nil, err
		}
		reward = customReward
	}

	// Save updated auction
	if err := a.saveAuction(ctx, nk, &auction); err != nil {
		logger.Error("Failed to save auction after cancel: %v", err)
		return nil, ErrInternal
	}

	// Remove from active auctions index
	if err := a.removeFromIndex(ctx, nk, auctionID); err != nil {
		logger.Error("Failed to remove auction from index: %v", err)
	}

	// Remove from user's created auctions index
	if err := a.removeFromUserCreatedIndex(ctx, nk, userID, auctionID); err != nil {
		logger.Error("Failed to remove auction from user created index: %v", err)
	}

	return &AuctionCancel{
		Auction: &auction,
		Reward:  reward,
	}, nil
}

// Create a new auction based on supplied parameters and available configuration
func (a *AuctionsPamlogix) Create(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, templateID, conditionID string, instanceIDs []string, startTimeSec int64, items []*InventoryItem, overrideConfig *AuctionsConfigAuction) (*Auction, error) {
	// Get template configuration
	var config *AuctionsConfigAuction
	if overrideConfig != nil {
		config = overrideConfig
	} else {
		var exists bool
		config, exists = a.config.Auctions[templateID]
		if !exists {
			return nil, ErrAuctionTemplateNotFound
		}
	}

	// Get condition configuration
	condition, exists := config.Conditions[conditionID]
	if !exists {
		return nil, ErrAuctionConditionNotFound
	}

	// Validate items
	if len(items) == 0 {
		return nil, ErrAuctionItemsInvalid
	}

	// Validate items against template
	if err := a.validateItems(items, config); err != nil {
		return nil, err
	}

	// Create auction
	auctionID := uuid.New().String()
	currentTime := time.Now().Unix()

	if startTimeSec == 0 {
		startTimeSec = currentTime
	}

	endTimeSec := startTimeSec + condition.DurationSec

	auction := &Auction{
		Id:                    auctionID,
		UserId:                userID,
		Reward:                &AuctionReward{Items: items},
		Version:               a.generateVersion(),
		DurationSec:           condition.DurationSec,
		OriginalDurationSec:   condition.DurationSec,
		ExtensionThresholdSec: condition.ExtensionThresholdSec,
		ExtensionSec:          condition.ExtensionSec,
		ExtensionMaxSec:       condition.ExtensionMaxSec,
		ExtensionRemainingSec: condition.ExtensionMaxSec, // Initialize with max extension time
		CreateTimeSec:         currentTime,
		StartTimeSec:          startTimeSec,
		EndTimeSec:            endTimeSec,
		OriginalEndTimeSec:    endTimeSec,
		CurrentTimeSec:        currentTime,
		HasStarted:            startTimeSec <= currentTime,
		HasEnded:              false,
		CanBid:                startTimeSec <= currentTime,
		CanClaim:              false,
		CanCancel:             true,
	}

	// Set bid start amount
	if condition.BidStart != nil {
		auction.BidNext = &AuctionBidAmount{
			Currencies: make(map[string]int64),
		}
		// Copy the bid start currencies
		for currency, amount := range condition.BidStart.Currencies {
			auction.BidNext.Currencies[currency] = amount
		}
	} else {
		// Default minimum bid if no start amount specified
		auction.BidNext = &AuctionBidAmount{
			Currencies: make(map[string]int64),
		}
	}

	// Set fee structure
	if condition.Fee != nil {
		auction.Fee = &AuctionFee{
			Percentage: condition.Fee.Percentage,
		}
		if condition.Fee.Fixed != nil {
			auction.Fee.Fixed = &AuctionBidAmount{
				Currencies: make(map[string]int64),
			}
			// Copy the fixed fee currencies
			for currency, amount := range condition.Fee.Fixed.Currencies {
				auction.Fee.Fixed.Currencies[currency] = amount
			}
		}
	}

	// Update state
	a.updateAuctionState(auction, currentTime, userID)

	// Save auction
	if err := a.saveAuction(ctx, nk, auction); err != nil {
		logger.Error("Failed to save new auction: %v", err)
		return nil, ErrInternal
	}

	// Add to index
	if err := a.addToIndex(ctx, nk, auctionID); err != nil {
		logger.Error("Failed to add auction to index: %v", err)
		return nil, ErrInternal
	}

	// Add to user's created auctions index
	if err := a.addToUserCreatedIndex(ctx, nk, userID, auctionID); err != nil {
		logger.Error("Failed to add auction to user created index: %v", err)
		// Don't return error as the auction was created successfully
	}

	return auction, nil
}

// ListBids returns auctions the user has successfully bid on
func (a *AuctionsPamlogix) ListBids(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, limit int, cursor string) (*AuctionList, error) {
	// Parse cursor for pagination
	var offset int
	if cursor != "" {
		var err error
		offset, err = strconv.Atoi(cursor)
		if err != nil {
			return nil, ErrBadInput
		}
	}

	// Read user's bid auctions index
	userBidsKey := fmt.Sprintf("%s_%s", AuctionUserBidsKey, userID)
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: AuctionCollectionKey,
			Key:        userBidsKey,
			UserID:     "",
		},
	})
	if err != nil {
		logger.Error("Failed to read user bid auctions index: %v", err)
		return nil, ErrInternal
	}

	var auctionIDs []string
	if len(objects) > 0 {
		var index map[string]bool
		if err := json.Unmarshal([]byte(objects[0].Value), &index); err != nil {
			logger.Error("Failed to unmarshal user bid auctions index: %v", err)
			return nil, ErrInternal
		}

		for auctionID := range index {
			auctionIDs = append(auctionIDs, auctionID)
		}
	}

	// Apply pagination
	if offset >= len(auctionIDs) {
		return &AuctionList{
			Auctions: []*Auction{},
			Cursor:   "",
		}, nil
	}

	end := offset + limit
	if end > len(auctionIDs) {
		end = len(auctionIDs)
	}

	paginatedIDs := auctionIDs[offset:end]

	// Read auction data
	var auctions []*Auction
	if len(paginatedIDs) > 0 {
		reads := make([]*runtime.StorageRead, len(paginatedIDs))
		for i, auctionID := range paginatedIDs {
			reads[i] = &runtime.StorageRead{
				Collection: AuctionCollectionKey,
				Key:        auctionID,
				UserID:     "",
			}
		}

		objects, err := nk.StorageRead(ctx, reads)
		if err != nil {
			logger.Error("Failed to read user bid auctions: %v", err)
			return nil, ErrInternal
		}

		currentTime := time.Now().Unix()
		for _, obj := range objects {
			var auction Auction
			if err := json.Unmarshal([]byte(obj.Value), &auction); err != nil {
				logger.Error("Failed to unmarshal auction %s: %v", obj.Key, err)
				continue
			}

			// Update auction state based on current time
			a.updateAuctionState(&auction, currentTime, userID)
			auctions = append(auctions, &auction)
		}
	}

	// Determine next cursor
	var nextCursor string
	if end < len(auctionIDs) {
		nextCursor = strconv.Itoa(end)
	}

	return &AuctionList{
		Auctions: auctions,
		Cursor:   nextCursor,
	}, nil
}

// ListCreated returns auctions the user has created
func (a *AuctionsPamlogix) ListCreated(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, limit int, cursor string) (*AuctionList, error) {
	// Parse cursor for pagination
	var offset int
	if cursor != "" {
		var err error
		offset, err = strconv.Atoi(cursor)
		if err != nil {
			return nil, ErrBadInput
		}
	}

	// Read user's created auctions index
	userCreatedKey := fmt.Sprintf("%s_%s", AuctionUserCreatedKey, userID)
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: AuctionCollectionKey,
			Key:        userCreatedKey,
			UserID:     "",
		},
	})
	if err != nil {
		logger.Error("Failed to read user created auctions index: %v", err)
		return nil, ErrInternal
	}

	var auctionIDs []string
	if len(objects) > 0 {
		var index map[string]bool
		if err := json.Unmarshal([]byte(objects[0].Value), &index); err != nil {
			logger.Error("Failed to unmarshal user created auctions index: %v", err)
			return nil, ErrInternal
		}

		for auctionID := range index {
			auctionIDs = append(auctionIDs, auctionID)
		}
	}

	// Apply pagination
	if offset >= len(auctionIDs) {
		return &AuctionList{
			Auctions: []*Auction{},
			Cursor:   "",
		}, nil
	}

	end := offset + limit
	if end > len(auctionIDs) {
		end = len(auctionIDs)
	}

	paginatedIDs := auctionIDs[offset:end]

	// Read auction data
	var auctions []*Auction
	if len(paginatedIDs) > 0 {
		reads := make([]*runtime.StorageRead, len(paginatedIDs))
		for i, auctionID := range paginatedIDs {
			reads[i] = &runtime.StorageRead{
				Collection: AuctionCollectionKey,
				Key:        auctionID,
				UserID:     "",
			}
		}

		objects, err := nk.StorageRead(ctx, reads)
		if err != nil {
			logger.Error("Failed to read user created auctions: %v", err)
			return nil, ErrInternal
		}

		currentTime := time.Now().Unix()
		for _, obj := range objects {
			var auction Auction
			if err := json.Unmarshal([]byte(obj.Value), &auction); err != nil {
				logger.Error("Failed to unmarshal auction %s: %v", obj.Key, err)
				continue
			}

			// Update auction state based on current time
			a.updateAuctionState(&auction, currentTime, userID)
			auctions = append(auctions, &auction)
		}
	}

	// Determine next cursor
	var nextCursor string
	if end < len(auctionIDs) {
		nextCursor = strconv.Itoa(end)
	}

	return &AuctionList{
		Auctions: auctions,
		Cursor:   nextCursor,
	}, nil
}

// Follow ensures users receive real-time updates for auctions they have an interest in
func (a *AuctionsPamlogix) Follow(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, sessionID string, auctionIDs []string) (*AuctionList, error) {
	// Read auctions
	reads := make([]*runtime.StorageRead, len(auctionIDs))
	for i, auctionID := range auctionIDs {
		reads[i] = &runtime.StorageRead{
			Collection: AuctionCollectionKey,
			Key:        auctionID,
			UserID:     "",
		}
	}

	objects, err := nk.StorageRead(ctx, reads)
	if err != nil {
		logger.Error("Failed to read auctions for follow: %v", err)
		return nil, ErrInternal
	}

	var auctions []*Auction
	currentTime := time.Now().Unix()
	streamMode := uint8(1)
	subcontext := "auction_bid_updates"
	label := "auction_notifications"

	for _, obj := range objects {
		var auction Auction
		if err := json.Unmarshal([]byte(obj.Value), &auction); err != nil {
			logger.Error("Failed to unmarshal auction %s: %v", obj.Key, err)
			continue
		}

		a.updateAuctionState(&auction, currentTime, userID)
		auctions = append(auctions, &auction)

		// Join the user to the auction's notification stream
		subject := auction.Id
		joined, err := nk.StreamUserJoin(streamMode, subject, subcontext, label, userID, sessionID, false, true, "")
		if err != nil {
			logger.Error("Failed to join user %s to auction %s stream: %v", userID, auction.Id, err)
		} else if joined {
			logger.Info("User %s joined auction %s notification stream", userID, auction.Id)
		}
	}

	return &AuctionList{
		Auctions: auctions,
		Cursor:   "",
	}, nil
}

// SetOnClaimBid sets a custom reward function which will run after an auction's reward is claimed by the winning bidder
func (a *AuctionsPamlogix) SetOnClaimBid(fn OnAuctionReward[*AuctionReward]) {
	a.onClaimBid = fn
}

// SetOnClaimCreated sets a custom reward function which will run after an auction's winning bid is claimed by the auction creator
func (a *AuctionsPamlogix) SetOnClaimCreated(fn OnAuctionReward[*AuctionBidAmount]) {
	a.onClaimCreated = fn
}

// SetOnClaimCreatedFailed sets a custom reward function which will run after a failed auction is claimed by the auction creator
func (a *AuctionsPamlogix) SetOnClaimCreatedFailed(fn OnAuctionReward[*AuctionReward]) {
	a.onClaimCreatedFailed = fn
}

// SetOnCancel sets a custom reward function which will run after an auction is cancelled by the auction creator
func (a *AuctionsPamlogix) SetOnCancel(fn OnAuctionReward[*AuctionReward]) {
	a.onCancel = fn
}

// Helper methods

// checkUserFunds verifies that a user has sufficient currency to place a bid
func (a *AuctionsPamlogix) checkUserFunds(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, bid *AuctionBidAmount) error {
	if a.pamlogix == nil {
		logger.Warn("Cannot check user funds: no Pamlogix instance available")
		return ErrInternal
	}

	economySystem := a.pamlogix.GetEconomySystem()
	if economySystem == nil {
		logger.Warn("Cannot check user funds: no EconomySystem available")
		return ErrInternal
	}

	// Get user's account to check wallet
	account, err := nk.AccountGetId(ctx, userID)
	if err != nil {
		logger.Error("Failed to get user account: %v", err)
		return ErrInternal
	}

	// Get wallet from account
	wallet, err := economySystem.UnmarshalWallet(account)
	if err != nil {
		logger.Error("Failed to unmarshal wallet: %v", err)
		return ErrInternal
	}

	// Check if user has enough of each required currency
	for currencyID, amount := range bid.Currencies {
		if wallet[currencyID] < amount {
			logger.Debug("User %s does not have enough of currency %s (has %d, needs %d)", userID, currencyID, wallet[currencyID], amount)
			return ErrCurrencyInsufficient
		}
	}

	return nil
}

// returnBidToUser returns currency to a user when they are outbid
func (a *AuctionsPamlogix) returnBidToUser(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, bid *AuctionBidAmount) error {
	if a.pamlogix == nil {
		logger.Warn("Cannot return bid: no Pamlogix instance available")
		return ErrInternal
	}

	economySystem := a.pamlogix.GetEconomySystem()
	if economySystem == nil {
		logger.Warn("Cannot return bid: no EconomySystem available")
		return ErrInternal
	}

	// Grant the bid amount back to the user
	metadata := map[string]interface{}{
		"source": "auction_bid_return",
		"reason": "outbid",
	}

	_, _, _, err := economySystem.Grant(ctx, logger, nk, userID, bid.Currencies, nil, nil, metadata)
	if err != nil {
		logger.Error("Failed to return bid currencies to user %s: %v", userID, err)
		return err
	}

	logger.Info("Returned bid to user %s: %v", userID, bid.Currencies)
	return nil
}

// deductBidFromUser deducts currency from a user when they place a bid
func (a *AuctionsPamlogix) deductBidFromUser(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, bid *AuctionBidAmount) error {
	if a.pamlogix == nil {
		logger.Warn("Cannot deduct bid: no Pamlogix instance available")
		return ErrInternal
	}

	economySystem := a.pamlogix.GetEconomySystem()
	if economySystem == nil {
		logger.Warn("Cannot deduct bid: no EconomySystem available")
		return ErrInternal
	}

	// Convert to negative values for deduction
	deductCurrencies := make(map[string]int64)
	for currencyID, amount := range bid.Currencies {
		deductCurrencies[currencyID] = -amount
	}

	// Deduct the bid amount from the user
	metadata := map[string]interface{}{
		"source": "auction_bid",
		"reason": "bid_placed",
	}

	_, _, _, err := economySystem.Grant(ctx, logger, nk, userID, deductCurrencies, nil, nil, metadata)
	if err != nil {
		logger.Error("Failed to deduct bid currencies from user %s: %v", userID, err)
		return err
	}

	logger.Info("Deducted bid from user %s: %v", userID, bid.Currencies)
	return nil
}

func (a *AuctionsPamlogix) chargeListingCost(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, listingCost *AuctionsConfigAuctionConditionCost) error {
	if a.pamlogix == nil {
		logger.Warn("Cannot charge listing cost: no Pamlogix instance available")
		return ErrInternal
	}

	// Charge currencies if any
	if len(listingCost.Currencies) > 0 {
		economySystem := a.pamlogix.GetEconomySystem()
		if economySystem == nil {
			logger.Warn("Cannot charge listing cost currencies: no EconomySystem available")
			return ErrInternal
		}

		// Convert to negative values for deduction
		deductCurrencies := make(map[string]int64)
		for currencyID, amount := range listingCost.Currencies {
			deductCurrencies[currencyID] = -amount
		}

		// Charge the currency listing cost
		metadata := map[string]interface{}{
			"source": "auction_listing",
			"reason": "listing_cost_currencies",
		}

		_, _, _, err := economySystem.Grant(ctx, logger, nk, userID, deductCurrencies, nil, nil, metadata)
		if err != nil {
			logger.Error("Failed to charge listing cost currencies from user %s: %v", userID, err)
			return err
		}
	}

	// Charge energies if any
	if len(listingCost.Energies) > 0 {
		energySystem := a.pamlogix.GetEnergySystem()
		if energySystem == nil {
			logger.Warn("Cannot charge listing cost energies: no EnergySystem available")
			return ErrInternal
		}

		// Convert to int32 and negative values for deduction
		deductEnergies := make(map[string]int32)
		for energyID, amount := range listingCost.Energies {
			deductEnergies[energyID] = -int32(amount)
		}

		// Charge the energy listing cost
		_, _, err := energySystem.Spend(ctx, logger, nk, userID, deductEnergies)
		if err != nil {
			logger.Error("Failed to charge listing cost energies from user %s: %v", userID, err)
			return err
		}
	}

	// Charge items if any
	if len(listingCost.Items) > 0 {
		inventorySystem := a.pamlogix.GetInventorySystem()
		if inventorySystem == nil {
			logger.Warn("Cannot charge listing cost items: no InventorySystem available")
			return ErrInternal
		}

		// Convert to negative values for deduction
		deductItems := make(map[string]int64)
		for itemID, amount := range listingCost.Items {
			deductItems[itemID] = -amount
		}

		// Charge the item listing cost
		_, _, _, _, err := inventorySystem.GrantItems(ctx, logger, nk, userID, deductItems, false)
		if err != nil {
			logger.Error("Failed to charge listing cost items from user %s: %v", userID, err)
			return err
		}
	}

	logger.Info("Charged listing cost from user %s: currencies=%v, energies=%v, items=%v", userID, listingCost.Currencies, listingCost.Energies, listingCost.Items)
	return nil
}

func (a *AuctionsPamlogix) updateAuctionState(auction *Auction, currentTime int64, userID string) {
	auction.CurrentTimeSec = currentTime

	// Update start/end status
	auction.HasStarted = auction.StartTimeSec <= currentTime
	auction.HasEnded = auction.EndTimeSec <= currentTime || auction.CancelTimeSec > 0

	// Update capabilities
	auction.CanBid = auction.HasStarted && !auction.HasEnded && auction.UserId != userID
	if auction.Bid != nil && auction.Bid.UserId == userID {
		auction.CanBid = false // Can't bid if already highest bidder
	}

	auction.CanClaim = auction.HasEnded && ((auction.Bid != nil && auction.Bid.UserId == userID && auction.WinnerClaimSec == 0) ||
		(auction.UserId == userID && auction.OwnerClaimSec == 0))

	auction.CanCancel = !auction.HasEnded && auction.UserId == userID && auction.Bid == nil
}

func (a *AuctionsPamlogix) validateBid(auction *Auction, userID string, bid *AuctionBidAmount, currentTime int64) error {
	if auction.UserId == userID {
		return ErrAuctionOwnBid
	}

	if auction.Bid != nil && auction.Bid.UserId == userID {
		return ErrAuctionAlreadyBid
	}

	if !auction.HasStarted {
		return ErrAuctionNotStarted
	}

	if auction.HasEnded {
		return ErrAuctionEnded
	}

	// Validate bid amount
	if auction.BidNext != nil {
		for currency, amount := range auction.BidNext.Currencies {
			bidAmount, exists := bid.Currencies[currency]
			if !exists || bidAmount < amount {
				return ErrAuctionBidInsufficient
			}
		}
	}

	// Validate bid currencies are not negative
	for _, amount := range bid.Currencies {
		if amount <= 0 {
			return ErrAuctionBidInvalid
		}
	}

	return nil
}

func (a *AuctionsPamlogix) processBid(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, auction *Auction, userID string, bid *AuctionBidAmount, currentTime int64, bidIncrement *AuctionsConfigAuctionConditionBidIncrement) error {
	// Return previous bid to previous bidder if any
	if auction.Bid != nil {
		if err := a.returnBidToUser(ctx, logger, nk, auction.Bid.UserId, auction.Bid.Bid); err != nil {
			logger.Error("Failed to return bid to previous bidder %s: %v", auction.Bid.UserId, err)
			// Continue despite error as the new bid should still be processed
		}

		// Remove the auction from the previous bidder's bids index since they're no longer the highest bidder
		if err := a.removeFromUserBidsIndex(ctx, nk, auction.Bid.UserId, auction.Id); err != nil {
			logger.Error("Failed to remove auction from previous bidder's index: %v", err)
		}
	}

	// Deduct the bid amount from the new bidder
	if err := a.deductBidFromUser(ctx, logger, nk, userID, bid); err != nil {
		logger.Error("Failed to deduct bid from user %s: %v", userID, err)
		return err
	}

	// Set new bid
	newBid := &AuctionBid{
		UserId:        userID,
		Bid:           bid,
		CreateTimeSec: currentTime,
	}

	if auction.BidFirst == nil {
		auction.BidFirst = newBid
	}

	auction.Bid = newBid
	auction.UpdateTimeSec = currentTime

	// Add to bid history
	if auction.BidHistory == nil {
		auction.BidHistory = []*AuctionBid{}
	}
	auction.BidHistory = append([]*AuctionBid{newBid}, auction.BidHistory...)

	// Limit bid history
	maxHistory := 10 // Default
	if config, exists := a.config.Auctions[""]; exists && config.BidHistoryCount > 0 {
		maxHistory = config.BidHistoryCount
	}
	if len(auction.BidHistory) > maxHistory {
		auction.BidHistory = auction.BidHistory[:maxHistory]
	}

	// Calculate next bid amount
	auction.BidNext = a.calculateNextBid(bid, bidIncrement)

	// Check for extension
	if auction.ExtensionThresholdSec > 0 && auction.ExtensionSec > 0 {
		timeToEnd := auction.EndTimeSec - currentTime
		if timeToEnd <= auction.ExtensionThresholdSec {
			extension := auction.ExtensionSec
			if auction.ExtensionRemainingSec > 0 && extension > auction.ExtensionRemainingSec {
				extension = auction.ExtensionRemainingSec
			}

			if extension > 0 {
				auction.EndTimeSec += extension
				auction.ExtensionAddedSec += extension
				auction.ExtensionRemainingSec -= extension

				if auction.ExtensionRemainingSec < 0 {
					auction.ExtensionRemainingSec = 0
				}
			}
		}
	}

	// Update version
	auction.Version = a.generateVersion()

	return nil
}

func (a *AuctionsPamlogix) calculateNextBid(currentBid *AuctionBidAmount, bidIncrement *AuctionsConfigAuctionConditionBidIncrement) *AuctionBidAmount {
	nextBid := &AuctionBidAmount{
		Currencies: make(map[string]int64),
	}

	for currency, amount := range currentBid.Currencies {
		minIncrement := int64(0)

		// Calculate percentage-based increment if specified
		if bidIncrement != nil && bidIncrement.Percentage > 0 {
			percentageIncrement := int64(float64(amount) * bidIncrement.Percentage)
			minIncrement = percentageIncrement
		}

		// Calculate fixed increment if specified
		if bidIncrement != nil && bidIncrement.Fixed != nil {
			if fixedAmount, exists := bidIncrement.Fixed.Currencies[currency]; exists {
				// If both percentage and fixed are specified, use the maximum to satisfy both conditions
				if minIncrement > 0 {
					if fixedAmount > minIncrement {
						minIncrement = fixedAmount
					}
				} else {
					minIncrement = fixedAmount
				}
			}
		}

		// Default increment if no configuration provided
		if minIncrement == 0 {
			minIncrement = amount / 10
			if minIncrement < 1 {
				minIncrement = 1
			}
		}

		nextBid.Currencies[currency] = amount + minIncrement
	}

	return nextBid
}

func (a *AuctionsPamlogix) calculateFee(bidAmount *AuctionBidAmount, feeConfig *AuctionFee) *AuctionBidAmount {
	if feeConfig == nil {
		return &AuctionBidAmount{Currencies: make(map[string]int64)}
	}

	fee := &AuctionBidAmount{
		Currencies: make(map[string]int64),
	}

	for currency, amount := range bidAmount.Currencies {
		feeAmount := int64(0)

		// Calculate percentage fee
		if feeConfig.Percentage > 0 {
			feeAmount = int64(float64(amount) * feeConfig.Percentage)
		}

		// Add fixed fee
		if feeConfig.Fixed != nil {
			if fixedAmount, exists := feeConfig.Fixed.Currencies[currency]; exists {
				feeAmount += fixedAmount
			}
		}

		fee.Currencies[currency] = feeAmount
	}

	return fee
}

func (a *AuctionsPamlogix) validateItems(items []*InventoryItem, config *AuctionsConfigAuction) error {
	if len(items) == 0 {
		return ErrAuctionItemsInvalid
	}

	// If no restrictions are configured, allow any items
	if len(config.Items) == 0 && len(config.ItemSets) == 0 {
		return nil
	}

	// Check each item against allowed items and item sets
	for _, item := range items {
		if item == nil || item.Id == "" {
			return ErrAuctionItemsInvalid
		}

		itemAllowed := false

		// Check against allowed individual items
		for _, allowedItem := range config.Items {
			if item.Id == allowedItem {
				itemAllowed = true
				break
			}
		}

		// If not found in individual items, check against item sets
		if !itemAllowed {
			// Get inventory system to check item sets
			if a.pamlogix != nil {
				inventorySystem := a.pamlogix.GetInventorySystem()
				if inventorySystem != nil {
					// Get the inventory config to access the pre-computed item sets
					inventoryConfig, ok := inventorySystem.GetConfig().(*InventoryConfig)
					if ok && inventoryConfig != nil && inventoryConfig.ItemSets != nil {
						// Check if item belongs to any allowed item set
						for _, allowedSetID := range config.ItemSets {
							if itemsInSet, exists := inventoryConfig.ItemSets[allowedSetID]; exists {
								if itemsInSet[item.Id] {
									itemAllowed = true
									break
								}
							}
						}
					}
				}
			}
		}

		if !itemAllowed {
			return ErrAuctionItemsInvalid
		}
	}

	return nil
}

func (a *AuctionsPamlogix) saveAuction(ctx context.Context, nk runtime.NakamaModule, auction *Auction) error {
	data, err := json.Marshal(auction)
	if err != nil {
		return err
	}

	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{
		{
			Collection: AuctionCollectionKey,
			Key:        auction.Id,
			UserID:     "",
			Value:      string(data),
		},
	})

	return err
}

func (a *AuctionsPamlogix) addToIndex(ctx context.Context, nk runtime.NakamaModule, auctionID string) error {
	// Read current index
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: AuctionCollectionKey,
			Key:        AuctionIndexKey,
			UserID:     "",
		},
	})

	var index map[string]bool
	if len(objects) > 0 {
		if err := json.Unmarshal([]byte(objects[0].Value), &index); err != nil {
			return err
		}
	} else {
		index = make(map[string]bool)
	}

	// Add auction to index
	index[auctionID] = true

	// Save updated index
	data, err := json.Marshal(index)
	if err != nil {
		return err
	}

	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{
		{
			Collection: AuctionCollectionKey,
			Key:        AuctionIndexKey,
			UserID:     "",
			Value:      string(data),
		},
	})

	return err
}

func (a *AuctionsPamlogix) removeFromIndex(ctx context.Context, nk runtime.NakamaModule, auctionID string) error {
	// Read current index
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: AuctionCollectionKey,
			Key:        AuctionIndexKey,
			UserID:     "",
		},
	})

	if len(objects) == 0 {
		return nil // Index doesn't exist
	}

	var index map[string]bool
	if err := json.Unmarshal([]byte(objects[0].Value), &index); err != nil {
		return err
	}

	// Remove auction from index
	delete(index, auctionID)

	// Save updated index
	data, err := json.Marshal(index)
	if err != nil {
		return err
	}

	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{
		{
			Collection: AuctionCollectionKey,
			Key:        AuctionIndexKey,
			UserID:     "",
			Value:      string(data),
		},
	})

	return err
}

func (a *AuctionsPamlogix) generateVersion() string {
	return fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%d", time.Now().UnixNano()))))
}

func (a *AuctionsPamlogix) addToUserCreatedIndex(ctx context.Context, nk runtime.NakamaModule, userID, auctionID string) error {
	// Read current user's created auctions index
	userCreatedKey := fmt.Sprintf("%s_%s", AuctionUserCreatedKey, userID)
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: AuctionCollectionKey,
			Key:        userCreatedKey,
			UserID:     "",
		},
	})

	var index map[string]bool
	if len(objects) > 0 {
		if err := json.Unmarshal([]byte(objects[0].Value), &index); err != nil {
			return err
		}
	} else {
		index = make(map[string]bool)
	}

	// Add auction to user's index
	index[auctionID] = true

	// Save updated index
	data, err := json.Marshal(index)
	if err != nil {
		return err
	}

	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{
		{
			Collection: AuctionCollectionKey,
			Key:        userCreatedKey,
			UserID:     "",
			Value:      string(data),
		},
	})

	return err
}

func (a *AuctionsPamlogix) removeFromUserCreatedIndex(ctx context.Context, nk runtime.NakamaModule, userID, auctionID string) error {
	// Read current user's created auctions index
	userCreatedKey := fmt.Sprintf("%s_%s", AuctionUserCreatedKey, userID)
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: AuctionCollectionKey,
			Key:        userCreatedKey,
			UserID:     "",
		},
	})

	if len(objects) == 0 {
		return nil // Index doesn't exist
	}

	var index map[string]bool
	if err := json.Unmarshal([]byte(objects[0].Value), &index); err != nil {
		return err
	}

	// Remove auction from user's index
	delete(index, auctionID)

	// Save updated index
	data, err := json.Marshal(index)
	if err != nil {
		return err
	}

	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{
		{
			Collection: AuctionCollectionKey,
			Key:        userCreatedKey,
			UserID:     "",
			Value:      string(data),
		},
	})

	return err
}

func (a *AuctionsPamlogix) addToUserBidsIndex(ctx context.Context, nk runtime.NakamaModule, userID, auctionID string) error {
	// Read current user's bid auctions index
	userBidsKey := fmt.Sprintf("%s_%s", AuctionUserBidsKey, userID)
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: AuctionCollectionKey,
			Key:        userBidsKey,
			UserID:     "",
		},
	})

	var index map[string]bool
	if len(objects) > 0 {
		if err := json.Unmarshal([]byte(objects[0].Value), &index); err != nil {
			return err
		}
	} else {
		index = make(map[string]bool)
	}

	// Add auction to user's bids index
	index[auctionID] = true

	// Save updated index
	data, err := json.Marshal(index)
	if err != nil {
		return err
	}

	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{
		{
			Collection: AuctionCollectionKey,
			Key:        userBidsKey,
			UserID:     "",
			Value:      string(data),
		},
	})

	return err
}

func (a *AuctionsPamlogix) removeFromUserBidsIndex(ctx context.Context, nk runtime.NakamaModule, userID, auctionID string) error {
	// Read current user's bid auctions index
	userBidsKey := fmt.Sprintf("%s_%s", AuctionUserBidsKey, userID)
	objects, err := nk.StorageRead(ctx, []*runtime.StorageRead{
		{
			Collection: AuctionCollectionKey,
			Key:        userBidsKey,
			UserID:     "",
		},
	})

	if len(objects) == 0 {
		return nil // Index doesn't exist
	}

	var index map[string]bool
	if err := json.Unmarshal([]byte(objects[0].Value), &index); err != nil {
		return err
	}

	// Remove auction from user's bids index
	delete(index, auctionID)

	// Save updated index
	data, err := json.Marshal(index)
	if err != nil {
		return err
	}

	_, err = nk.StorageWrite(ctx, []*runtime.StorageWrite{
		{
			Collection: AuctionCollectionKey,
			Key:        userBidsKey,
			UserID:     "",
			Value:      string(data),
		},
	})

	return err
}

func (a *AuctionsPamlogix) sendBidNotification(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, auction *Auction, sessionID string) {
	// Create the bid notification payload
	bidNotification := &AuctionNotificationBid{
		Id:                    auction.Id,
		Version:               auction.Version,
		Bid:                   auction.Bid,
		BidNext:               auction.BidNext,
		ExtensionAddedSec:     auction.ExtensionAddedSec,
		ExtensionRemainingSec: auction.ExtensionRemainingSec,
		UpdateTimeSec:         auction.UpdateTimeSec,
		EndTimeSec:            auction.EndTimeSec,
		CurrentTimeSec:        auction.CurrentTimeSec,
	}

	// Marshal the notification to JSON for stream data
	notificationData, err := json.Marshal(bidNotification)
	if err != nil {
		logger.Error("Failed to marshal bid notification for auction %s: %v", auction.Id, err)
		return
	}

	// Send real-time notification via stream to auction followers
	// Stream mode 1 is typically used for custom application streams
	// Subject is the auction ID, subcontext can be "bid_updates"
	streamMode := uint8(1)
	subject := auction.Id
	subcontext := "auction_bid_updates"
	label := "auction_notifications"

	// Get list of users following this auction stream
	presences, err := nk.StreamUserList(streamMode, subject, subcontext, label, true, true)
	if err != nil {
		logger.Error("Failed to get auction followers for auction %s: %v", auction.Id, err)
		return
	}

	if len(presences) > 0 {
		// Send the notification to all followers via stream
		err = nk.StreamSend(streamMode, subject, subcontext, label, string(notificationData), presences, true)
		if err != nil {
			logger.Error("Failed to send stream notification for auction %s: %v", auction.Id, err)
		} else {
			logger.Info("Sent bid notification for auction %s to %d followers", auction.Id, len(presences))
		}
	}

	// Also send persistent notifications to interested users
	// Send to auction creator (unless they are the bidder)
	if auction.UserId != auction.Bid.UserId {
		content := map[string]interface{}{
			"auction_id": auction.Id,
			"bid_amount": auction.Bid.Bid.Currencies,
			"bidder_id":  auction.Bid.UserId,
			"type":       "auction_bid",
		}

		err = nk.NotificationSend(ctx, auction.UserId, "New bid on your auction", content, 1001, "", true)
		if err != nil {
			logger.Error("Failed to send notification to auction creator %s: %v", auction.UserId, err)
		}
	}

	// Send notification to previous high bidder (if any and different from current bidder)
	if len(auction.BidHistory) > 1 {
		previousBid := auction.BidHistory[1] // Second item is the previous bid
		if previousBid.UserId != auction.Bid.UserId {
			content := map[string]interface{}{
				"auction_id": auction.Id,
				"bid_amount": auction.Bid.Bid.Currencies,
				"type":       "auction_outbid",
			}

			err = nk.NotificationSend(ctx, previousBid.UserId, "You have been outbid", content, 1002, "", true)
			if err != nil {
				logger.Error("Failed to send outbid notification to user %s: %v", previousBid.UserId, err)
			}
		}
	}

	logger.Info("Completed bid notification processing for auction %s", auction.Id)
}

// followAuctionForUser adds a user to an auction's notification stream
func (a *AuctionsPamlogix) followAuctionForUser(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID, sessionID, auctionID string) {
	streamMode := uint8(1)
	subject := auctionID
	subcontext := "auction_bid_updates"
	label := "auction_notifications"

	joined, err := nk.StreamUserJoin(streamMode, subject, subcontext, label, userID, sessionID, false, true, "")
	if err != nil {
		logger.Error("Failed to join user %s to auction %s stream: %v", userID, auctionID, err)
	} else if joined {
		logger.Info("User %s automatically joined auction %s notification stream", userID, auctionID)
	}
}

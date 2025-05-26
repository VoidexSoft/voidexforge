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
	AuctionCollectionKey = "auctions"
	AuctionIndexKey      = "auction_index"
	AuctionBidsKey       = "auction_bids"
)

// AuctionsPamlogix implements the AuctionsSystem interface
type AuctionsPamlogix struct {
	config *AuctionsConfig

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

	// Check if user has sufficient funds
	// This would typically integrate with the economy system
	// For now, we'll assume the check passes

	// Process the bid
	if err := a.processBid(ctx, logger, nk, &auction, userID, bid, currentTime); err != nil {
		return nil, err
	}

	// Save updated auction
	if err := a.saveAuction(ctx, nk, &auction); err != nil {
		logger.Error("Failed to save auction after bid: %v", err)
		return nil, ErrInternal
	}

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
		// This would integrate with economy system to return funds
		// For now, we'll just mark it as cancelled
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
			Currencies: condition.BidStart.Currencies,
		}
	}

	// Set fee structure
	if condition.Fee != nil {
		auction.Fee = &AuctionFee{
			Percentage: condition.Fee.Percentage,
		}
		if condition.Fee.Fixed != nil {
			auction.Fee.Fixed = &AuctionBidAmount{
				Currencies: condition.Fee.Fixed.Currencies,
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

	return auction, nil
}

// ListBids returns auctions the user has successfully bid on
func (a *AuctionsPamlogix) ListBids(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, limit int, cursor string) (*AuctionList, error) {
	// This would typically query a user-specific index of bids
	// For now, we'll return an empty list
	return &AuctionList{
		Auctions: []*Auction{},
		Cursor:   "",
	}, nil
}

// ListCreated returns auctions the user has created
func (a *AuctionsPamlogix) ListCreated(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, userID string, limit int, cursor string) (*AuctionList, error) {
	// This would typically query a user-specific index of created auctions
	// For now, we'll return an empty list
	return &AuctionList{
		Auctions: []*Auction{},
		Cursor:   "",
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

	for _, obj := range objects {
		var auction Auction
		if err := json.Unmarshal([]byte(obj.Value), &auction); err != nil {
			logger.Error("Failed to unmarshal auction %s: %v", obj.Key, err)
			continue
		}

		a.updateAuctionState(&auction, currentTime, userID)
		auctions = append(auctions, &auction)
	}

	// TODO: Implement actual following mechanism with real-time updates

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

	return nil
}

func (a *AuctionsPamlogix) processBid(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, auction *Auction, userID string, bid *AuctionBidAmount, currentTime int64) error {
	// Return previous bid to previous bidder if any
	if auction.Bid != nil {
		// This would integrate with economy system
		// For now, we'll just log it
		logger.Info("Returning bid to previous bidder: %s", auction.Bid.UserId)
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
	auction.BidNext = a.calculateNextBid(bid)

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

func (a *AuctionsPamlogix) calculateNextBid(currentBid *AuctionBidAmount) *AuctionBidAmount {
	nextBid := &AuctionBidAmount{
		Currencies: make(map[string]int64),
	}

	// Simple increment - add 10% or minimum 1
	for currency, amount := range currentBid.Currencies {
		increment := amount / 10
		if increment < 1 {
			increment = 1
		}
		nextBid.Currencies[currency] = amount + increment
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
	// This would validate items against the allowed items/item sets
	// For now, we'll just check that items exist
	if len(items) == 0 {
		return ErrAuctionItemsInvalid
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

func (a *AuctionsPamlogix) sendBidNotification(ctx context.Context, logger runtime.Logger, nk runtime.NakamaModule, auction *Auction, sessionID string) {
	// This would send real-time notifications to auction followers
	// Implementation would depend on the specific real-time messaging system
	logger.Info("Sending bid notification for auction %s", auction.Id)
}

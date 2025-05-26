package pamlogix

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/heroiclabs/nakama-common/runtime"
	"google.golang.org/protobuf/encoding/protojson"
)

// rpcAuctionsGetTemplates handles the get templates RPC
func rpcAuctionsGetTemplates(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		auctionsSystem := p.GetAuctionsSystem()
		if auctionsSystem == nil {
			return "", runtime.NewError("auctions system not available", 12) // UNIMPLEMENTED
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", ErrNoSessionUser
		}

		templates, err := auctionsSystem.GetTemplates(ctx, logger, nk, userID)
		if err != nil {
			logger.Error("Error getting auction templates: %v", err)
			return "", err
		}

		responseData, err := json.Marshal(templates)
		if err != nil {
			logger.Error("Failed to marshal auction templates response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

// rpcAuctionsList handles the list auctions RPC
func rpcAuctionsList(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		auctionsSystem := p.GetAuctionsSystem()
		if auctionsSystem == nil {
			return "", runtime.NewError("auctions system not available", 12) // UNIMPLEMENTED
		}

		var request AuctionListRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal AuctionListRequest: %v", err)
			return "", ErrPayloadDecode
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", ErrNoSessionUser
		}

		// Set default limit if not provided
		limit := int(request.Limit)
		if limit <= 0 {
			limit = 20
		}

		auctionList, err := auctionsSystem.List(ctx, logger, nk, userID, request.Query, request.Sort, limit, request.Cursor)
		if err != nil {
			logger.Error("Error listing auctions: %v", err)
			return "", err
		}

		responseData, err := json.Marshal(auctionList)
		if err != nil {
			logger.Error("Failed to marshal auction list response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

// rpcAuctionsBid handles the bid on auction RPC
func rpcAuctionsBid(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		auctionsSystem := p.GetAuctionsSystem()
		if auctionsSystem == nil {
			return "", runtime.NewError("auctions system not available", 12) // UNIMPLEMENTED
		}

		var request AuctionBidRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal AuctionBidRequest: %v", err)
			return "", ErrPayloadDecode
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", ErrNoSessionUser
		}

		sessionID, ok := ctx.Value(runtime.RUNTIME_CTX_SESSION_ID).(string)
		if !ok || sessionID == "" {
			return "", ErrNoSessionID
		}

		marshaler := &protojson.MarshalOptions{}
		auction, err := auctionsSystem.Bid(ctx, logger, nk, userID, sessionID, request.Id, request.Version, request.Bid, marshaler)
		if err != nil {
			logger.Error("Error placing bid on auction: %v", err)
			return "", err
		}

		responseData, err := json.Marshal(auction)
		if err != nil {
			logger.Error("Failed to marshal auction bid response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

// rpcAuctionsClaimBid handles the claim bid RPC
func rpcAuctionsClaimBid(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		auctionsSystem := p.GetAuctionsSystem()
		if auctionsSystem == nil {
			return "", runtime.NewError("auctions system not available", 12) // UNIMPLEMENTED
		}

		var request AuctionClaimBidRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal AuctionClaimBidRequest: %v", err)
			return "", ErrPayloadDecode
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", ErrNoSessionUser
		}

		claimResult, err := auctionsSystem.ClaimBid(ctx, logger, nk, userID, request.Id)
		if err != nil {
			logger.Error("Error claiming auction bid: %v", err)
			return "", err
		}

		responseData, err := json.Marshal(claimResult)
		if err != nil {
			logger.Error("Failed to marshal auction claim bid response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

// rpcAuctionsClaimCreated handles the claim created auction RPC
func rpcAuctionsClaimCreated(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		auctionsSystem := p.GetAuctionsSystem()
		if auctionsSystem == nil {
			return "", runtime.NewError("auctions system not available", 12) // UNIMPLEMENTED
		}

		var request AuctionClaimCreatedRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal AuctionClaimCreatedRequest: %v", err)
			return "", ErrPayloadDecode
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", ErrNoSessionUser
		}

		claimResult, err := auctionsSystem.ClaimCreated(ctx, logger, nk, userID, request.Id)
		if err != nil {
			logger.Error("Error claiming created auction: %v", err)
			return "", err
		}

		responseData, err := json.Marshal(claimResult)
		if err != nil {
			logger.Error("Failed to marshal auction claim created response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

// rpcAuctionsCancel handles the cancel auction RPC
func rpcAuctionsCancel(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		auctionsSystem := p.GetAuctionsSystem()
		if auctionsSystem == nil {
			return "", runtime.NewError("auctions system not available", 12) // UNIMPLEMENTED
		}

		var request AuctionCancelRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal AuctionCancelRequest: %v", err)
			return "", ErrPayloadDecode
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", ErrNoSessionUser
		}

		cancelResult, err := auctionsSystem.Cancel(ctx, logger, nk, userID, request.Id)
		if err != nil {
			logger.Error("Error cancelling auction: %v", err)
			return "", err
		}

		responseData, err := json.Marshal(cancelResult)
		if err != nil {
			logger.Error("Failed to marshal auction cancel response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

// rpcAuctionsCreate handles the create auction RPC
func rpcAuctionsCreate(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		auctionsSystem := p.GetAuctionsSystem()
		if auctionsSystem == nil {
			return "", runtime.NewError("auctions system not available", 12) // UNIMPLEMENTED
		}

		var request AuctionCreateRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal AuctionCreateRequest: %v", err)
			return "", ErrPayloadDecode
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", ErrNoSessionUser
		}

		// Note: The AuctionCreateRequest uses instance_ids, not items directly
		// The items will be retrieved from inventory based on instance_ids
		auction, err := auctionsSystem.Create(ctx, logger, nk, userID, request.TemplateId, request.ConditionId, request.InstanceIds, request.StartTimeSec, nil, nil)
		if err != nil {
			logger.Error("Error creating auction: %v", err)
			return "", err
		}

		responseData, err := json.Marshal(auction)
		if err != nil {
			logger.Error("Failed to marshal auction create response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

// rpcAuctionsListBids handles the list user bids RPC
func rpcAuctionsListBids(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		auctionsSystem := p.GetAuctionsSystem()
		if auctionsSystem == nil {
			return "", runtime.NewError("auctions system not available", 12) // UNIMPLEMENTED
		}

		var request AuctionListBidsRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal AuctionListBidsRequest: %v", err)
			return "", ErrPayloadDecode
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", ErrNoSessionUser
		}

		// Set default limit if not provided
		limit := int(request.Limit)
		if limit <= 0 {
			limit = 20
		}

		auctionList, err := auctionsSystem.ListBids(ctx, logger, nk, userID, limit, request.Cursor)
		if err != nil {
			logger.Error("Error listing user bids: %v", err)
			return "", err
		}

		responseData, err := json.Marshal(auctionList)
		if err != nil {
			logger.Error("Failed to marshal auction list bids response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

// rpcAuctionsListCreated handles the list user created auctions RPC
func rpcAuctionsListCreated(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		auctionsSystem := p.GetAuctionsSystem()
		if auctionsSystem == nil {
			return "", runtime.NewError("auctions system not available", 12) // UNIMPLEMENTED
		}

		var request AuctionListCreatedRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal AuctionListCreatedRequest: %v", err)
			return "", ErrPayloadDecode
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", ErrNoSessionUser
		}

		// Set default limit if not provided
		limit := int(request.Limit)
		if limit <= 0 {
			limit = 20
		}

		auctionList, err := auctionsSystem.ListCreated(ctx, logger, nk, userID, limit, request.Cursor)
		if err != nil {
			logger.Error("Error listing user created auctions: %v", err)
			return "", err
		}

		responseData, err := json.Marshal(auctionList)
		if err != nil {
			logger.Error("Failed to marshal auction list created response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

// rpcAuctionsFollow handles the follow auctions RPC (for real-time updates)
func rpcAuctionsFollow(p *pamlogixImpl) func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		auctionsSystem := p.GetAuctionsSystem()
		if auctionsSystem == nil {
			return "", runtime.NewError("auctions system not available", 12) // UNIMPLEMENTED
		}

		var request AuctionsFollowRequest
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			logger.Error("Failed to unmarshal AuctionsFollowRequest: %v", err)
			return "", ErrPayloadDecode
		}

		userID, ok := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if !ok || userID == "" {
			return "", ErrNoSessionUser
		}

		sessionID, ok := ctx.Value(runtime.RUNTIME_CTX_SESSION_ID).(string)
		if !ok || sessionID == "" {
			return "", ErrNoSessionID
		}

		auctionList, err := auctionsSystem.Follow(ctx, logger, nk, userID, sessionID, request.Ids)
		if err != nil {
			logger.Error("Error following auctions: %v", err)
			return "", err
		}

		responseData, err := json.Marshal(auctionList)
		if err != nil {
			logger.Error("Failed to marshal auction follow response: %v", err)
			return "", ErrPayloadEncode
		}

		return string(responseData), nil
	}
}

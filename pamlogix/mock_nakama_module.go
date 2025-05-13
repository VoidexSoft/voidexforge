package pamlogix

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/heroiclabs/nakama-common/api"
	"github.com/heroiclabs/nakama-common/rtapi"
	"github.com/heroiclabs/nakama-common/runtime"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// Define the MockNakamaModule struct to match the test file and allow interface satisfaction
// Embed mock.Mock for testify compatibility

type MockNakamaModule struct {
	mock.Mock
	*sql.DB
	logger *zap.Logger
}

func (m *MockNakamaModule) Log(msg string, fields ...zap.Field) {
	if m.logger != nil {
		m.logger.Info(msg, fields...)
	}
}

// NewMockNakama returns a new instance of MockNakamaModule for use in tests
func NewMockNakama(t *testing.T) *MockNakamaModule {
	logger, _ := zap.NewDevelopment()
	return &MockNakamaModule{
		logger: logger,
	}
}

func (m *MockNakamaModule) StreamUserList(mode uint8, subject, subcontext, label string, includeHidden, includeNotHidden bool) ([]runtime.Presence, error) {
	args := m.Called(mode, subject, subcontext, label, includeHidden, includeNotHidden)
	return args.Get(0).([]runtime.Presence), args.Error(1)
}

func (m *MockNakamaModule) StreamUserGet(mode uint8, subject, subcontext, label, userID, sessionID string) (runtime.PresenceMeta, error) {
	args := m.Called(mode, subject, subcontext, label, userID, sessionID)
	return args.Get(0).(runtime.PresenceMeta), args.Error(1)
}

func (m *MockNakamaModule) StreamUserKick(mode uint8, subject, subcontext, label string, presence runtime.Presence) error {
	args := m.Called(mode, subject, subcontext, label, presence)
	return args.Error(0)
}

func (m *MockNakamaModule) StreamSend(mode uint8, subject, subcontext, label, data string, presences []runtime.Presence, reliable bool) error {
	args := m.Called(mode, subject, subcontext, label, data, presences, reliable)
	return args.Error(0)
}

func (m *MockNakamaModule) StreamSendRaw(mode uint8, subject, subcontext, label string, msg *rtapi.Envelope, presences []runtime.Presence, reliable bool) error {
	args := m.Called(mode, subject, subcontext, label, msg, presences, reliable)
	return args.Error(0)
}

func (m *MockNakamaModule) SessionDisconnect(ctx context.Context, sessionID string, reason ...runtime.PresenceReason) error {
	args := m.Called(ctx, sessionID, reason)
	return args.Error(0)
}

func (m *MockNakamaModule) NotificationsSend(ctx context.Context, notifications []*runtime.NotificationSend) error {
	args := m.Called(ctx, notifications)
	return args.Error(0)
}

func (m *MockNakamaModule) NotificationsUpdate(ctx context.Context, updates ...runtime.NotificationUpdate) error {
	args := m.Called(ctx, updates)
	return args.Error(0)
}

func (m *MockNakamaModule) NotificationsDelete(ctx context.Context, notifications []*runtime.NotificationDelete) error {
	args := m.Called(ctx, notifications)
	return args.Error(0)
}

func (m *MockNakamaModule) NotificationsGetId(ctx context.Context, userID string, ids []string) ([]*runtime.Notification, error) {
	args := m.Called(ctx, userID, ids)
	return args.Get(0).([]*runtime.Notification), args.Error(1)
}

func (m *MockNakamaModule) WalletUpdate(ctx context.Context, userID string, changeset map[string]int64, metadata map[string]interface{}, updateLedger bool) (updated map[string]int64, previous map[string]int64, err error) {
	args := m.Called(ctx, userID, changeset, metadata, updateLedger)
	return args.Get(0).(map[string]int64), args.Get(1).(map[string]int64), args.Error(2)
}

func (m *MockNakamaModule) WalletsUpdate(ctx context.Context, updates []*runtime.WalletUpdate, updateLedger bool) ([]*runtime.WalletUpdateResult, error) {
	args := m.Called(ctx, updates, updateLedger)
	return args.Get(0).([]*runtime.WalletUpdateResult), args.Error(1)
}

func (m *MockNakamaModule) WalletLedgerUpdate(ctx context.Context, itemID string, metadata map[string]interface{}) (runtime.WalletLedgerItem, error) {
	args := m.Called(ctx, itemID, metadata)
	return args.Get(0).(runtime.WalletLedgerItem), args.Error(1)
}

func (m *MockNakamaModule) WalletLedgerList(ctx context.Context, userID string, limit int, cursor string) ([]runtime.WalletLedgerItem, string, error) {
	args := m.Called(ctx, userID, limit, cursor)
	return args.Get(0).([]runtime.WalletLedgerItem), args.String(1), args.Error(2)
}

func (m *MockNakamaModule) StorageList(ctx context.Context, callerID, userID, collection string, limit int, cursor string) ([]*api.StorageObject, string, error) {
	args := m.Called(ctx, callerID, userID, collection, limit, cursor)
	return args.Get(0).([]*api.StorageObject), args.String(1), args.Error(2)
}

func (m *MockNakamaModule) StorageRead(ctx context.Context, objectIDs []*runtime.StorageRead) ([]*api.StorageObject, error) {
	args := m.Called(ctx, objectIDs)
	return args.Get(0).([]*api.StorageObject), args.Error(1)
}

func (m *MockNakamaModule) StorageWrite(ctx context.Context, writes []*runtime.StorageWrite) ([]*api.StorageObjectAck, error) {
	args := m.Called(ctx, writes)
	return args.Get(0).([]*api.StorageObjectAck), args.Error(1)
}

func (m *MockNakamaModule) StorageDelete(ctx context.Context, deletes []*runtime.StorageDelete) error {
	args := m.Called(ctx, deletes)
	return args.Error(0)
}

func (m *MockNakamaModule) MultiUpdate(ctx context.Context, accountUpdates []*runtime.AccountUpdate, storageWrites []*runtime.StorageWrite, storageDeletes []*runtime.StorageDelete, walletUpdates []*runtime.WalletUpdate, updateLedger bool) ([]*api.StorageObjectAck, []*runtime.WalletUpdateResult, error) {
	args := m.Called(ctx, accountUpdates, storageWrites, storageDeletes, walletUpdates, updateLedger)
	return args.Get(0).([]*api.StorageObjectAck), args.Get(1).([]*runtime.WalletUpdateResult), args.Error(2)
}

func (m *MockNakamaModule) TournamentRecordsHaystack(ctx context.Context, id, ownerID string, limit int, cursor string, expiry int64) (*api.TournamentRecordList, error) {
	args := m.Called(ctx, id, ownerID, limit, cursor, expiry)
	return args.Get(0).(*api.TournamentRecordList), args.Error(1)
}

func (m *MockNakamaModule) GroupsGetId(ctx context.Context, groupIDs []string) ([]*api.Group, error) {
	args := m.Called(ctx, groupIDs)
	return args.Get(0).([]*api.Group), args.Error(1)
}

func (m *MockNakamaModule) GroupCreate(ctx context.Context, userID, name, creatorID, langTag, description, avatarUrl string, open bool, metadata map[string]interface{}, maxCount int) (*api.Group, error) {
	args := m.Called(ctx, userID, name, creatorID, langTag, description, avatarUrl, open, metadata, maxCount)
	return args.Get(0).(*api.Group), args.Error(1)
}

func (m *MockNakamaModule) GroupUpdate(ctx context.Context, id, userID, name, creatorID, langTag, description, avatarUrl string, open bool, metadata map[string]interface{}, maxCount int) error {
	args := m.Called(ctx, id, userID, name, creatorID, langTag, description, avatarUrl, open, metadata, maxCount)
	return args.Error(0)
}

func (m *MockNakamaModule) GroupUserJoin(ctx context.Context, groupID, userID, username string) error {
	args := m.Called(ctx, groupID, userID, username)
	return args.Error(0)
}

func (m *MockNakamaModule) GroupUserLeave(ctx context.Context, groupID, userID, username string) error {
	args := m.Called(ctx, groupID, userID, username)
	return args.Error(0)
}

func (m *MockNakamaModule) GroupUsersAdd(ctx context.Context, callerID, groupID string, userIDs []string) error {
	args := m.Called(ctx, callerID, groupID, userIDs)
	return args.Error(0)
}

func (m *MockNakamaModule) GroupUsersBan(ctx context.Context, callerID, groupID string, userIDs []string) error {
	args := m.Called(ctx, callerID, groupID, userIDs)
	return args.Error(0)
}

func (m *MockNakamaModule) GroupUsersKick(ctx context.Context, callerID, groupID string, userIDs []string) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) GroupUsersPromote(ctx context.Context, callerID, groupID string, userIDs []string) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) GroupUsersDemote(ctx context.Context, callerID, groupID string, userIDs []string) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) GroupUsersList(ctx context.Context, id string, limit int, state *int, cursor string) ([]*api.GroupUserList_GroupUser, string, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) GroupsList(ctx context.Context, name, langTag string, members *int, open *bool, limit int, cursor string) ([]*api.Group, string, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) GroupsGetRandom(ctx context.Context, count int) ([]*api.Group, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) UserGroupsList(ctx context.Context, userID string, limit int, state *int, cursor string) ([]*api.UserGroupList_UserGroup, string, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) FriendMetadataUpdate(ctx context.Context, userID string, friendUserId string, metadata map[string]any) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) FriendsList(ctx context.Context, userID string, limit int, state *int, cursor string) ([]*api.Friend, string, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) FriendsOfFriendsList(ctx context.Context, userID string, limit int, cursor string) ([]*api.FriendsOfFriendsList_FriendOfFriend, string, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) FriendsAdd(ctx context.Context, userID string, username string, ids []string, usernames []string, metadata map[string]any) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) FriendsDelete(ctx context.Context, userID string, username string, ids []string, usernames []string) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) FriendsBlock(ctx context.Context, userID string, username string, ids []string, usernames []string) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) Event(ctx context.Context, evt *api.Event) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) MetricsCounterAdd(name string, tags map[string]string, delta int64) {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) MetricsGaugeSet(name string, tags map[string]string, value float64) {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) MetricsTimerRecord(name string, tags map[string]string, value time.Duration) {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) ChannelMessageSend(ctx context.Context, channelID string, content map[string]interface{}, senderId, senderUsername string, persist bool) (*rtapi.ChannelMessageAck, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) ChannelMessageUpdate(ctx context.Context, channelID, messageID string, content map[string]interface{}, senderId, senderUsername string, persist bool) (*rtapi.ChannelMessageAck, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) ChannelMessageRemove(ctx context.Context, channelId, messageId string, senderId, senderUsername string, persist bool) (*rtapi.ChannelMessageAck, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) ChannelMessagesList(ctx context.Context, channelId string, limit int, forward bool, cursor string) (messages []*api.ChannelMessage, nextCursor string, prevCursor string, err error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) StatusFollow(sessionID string, userIDs []string) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) StatusUnfollow(sessionID string, userIDs []string) error {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) GetSatori() runtime.Satori {
	//TODO implement me
	panic("implement me")
}

func (m *MockNakamaModule) GetFleetManager() runtime.FleetManager {
	//TODO implement me
	panic("implement me")
}

// Add any missing types here for compilation
type Presence interface{}
type PresenceMeta interface{}
type PresenceReason int

// type ChannelType int
type WalletUpdate struct{}
type WalletUpdateResult struct{}
type WalletLedgerItem struct{}
type StorageRead = runtime.StorageRead
type StorageWrite = runtime.StorageWrite
type StorageDelete struct{}
type AccountUpdate struct{}
type NotificationSend struct{}
type NotificationUpdate struct{}
type NotificationDelete struct{}
type Notification struct{}

// Add a placeholder GroupJoinRequest type for testing if not available in Nakama API
// Remove this if the real type exists in the Nakama API package
// This is only for test compilation

type GroupJoinRequest struct{}

// Satisfy the NakamaModule interface with stubs
func (m *MockNakamaModule) AuthenticateApple(ctx context.Context, token, username string, create bool) (string, string, bool, error) {
	return "", "", false, nil
}
func (m *MockNakamaModule) AuthenticateCustom(ctx context.Context, id, username string, create bool) (string, string, bool, error) {
	return "", "", false, nil
}
func (m *MockNakamaModule) AuthenticateDevice(ctx context.Context, id, username string, create bool) (string, string, bool, error) {
	return "", "", false, nil
}
func (m *MockNakamaModule) AuthenticateEmail(ctx context.Context, email, password, username string, create bool) (string, string, bool, error) {
	return "", "", false, nil
}
func (m *MockNakamaModule) AuthenticateFacebook(ctx context.Context, token string, importFriends bool, username string, create bool) (string, string, bool, error) {
	return "", "", false, nil
}
func (m *MockNakamaModule) AuthenticateFacebookInstantGame(ctx context.Context, signedPlayerInfo string, username string, create bool) (string, string, bool, error) {
	return "", "", false, nil
}
func (m *MockNakamaModule) AuthenticateGameCenter(ctx context.Context, playerID, bundleID string, timestamp int64, salt, signature, publicKeyUrl, username string, create bool) (string, string, bool, error) {
	return "", "", false, nil
}
func (m *MockNakamaModule) AuthenticateGoogle(ctx context.Context, token, username string, create bool) (string, string, bool, error) {
	return "", "", false, nil
}
func (m *MockNakamaModule) AuthenticateSteam(ctx context.Context, token, username string, create bool) (string, string, bool, error) {
	return "", "", false, nil
}
func (m *MockNakamaModule) AuthenticateTokenGenerate(userID, username string, exp int64, vars map[string]string) (string, int64, error) {
	return "", 0, nil
}
func (m *MockNakamaModule) AccountGetId(ctx context.Context, userID string) (*api.Account, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(*api.Account), args.Error(1)
}
func (m *MockNakamaModule) AccountsGetId(ctx context.Context, userIDs []string) ([]*api.Account, error) {
	args := m.Called(ctx, userIDs)
	return args.Get(0).([]*api.Account), args.Error(1)
}
func (m *MockNakamaModule) AccountUpdateId(ctx context.Context, userID, username string, metadata map[string]interface{}, displayName, timezone, location, langTag, avatarUrl string) error {
	args := m.Called(ctx, userID, username, metadata, displayName, timezone, location, langTag, avatarUrl)
	return args.Error(0)
}
func (m *MockNakamaModule) AccountDeleteId(ctx context.Context, userID string, recorded bool) error {
	args := m.Called(ctx, userID, recorded)
	return args.Error(0)
}
func (m *MockNakamaModule) AccountExportId(ctx context.Context, userID string) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}
func (m *MockNakamaModule) UsersGetId(ctx context.Context, userIDs []string, facebookIDs []string) ([]*api.User, error) {
	args := m.Called(ctx, userIDs, facebookIDs)
	return args.Get(0).([]*api.User), args.Error(1)
}
func (m *MockNakamaModule) UsersGetUsername(ctx context.Context, usernames []string) ([]*api.User, error) {
	args := m.Called(ctx, usernames)
	return args.Get(0).([]*api.User), args.Error(1)
}
func (m *MockNakamaModule) UsersGetFriendStatus(ctx context.Context, userID string, userIDs []string) ([]*api.Friend, error) {
	args := m.Called(ctx, userID, userIDs)
	return args.Get(0).([]*api.Friend), args.Error(1)
}
func (m *MockNakamaModule) UsersGetRandom(ctx context.Context, count int) ([]*api.User, error) {
	args := m.Called(ctx, count)
	return args.Get(0).([]*api.User), args.Error(1)
}
func (m *MockNakamaModule) UsersBanId(ctx context.Context, userIDs []string) error {
	args := m.Called(ctx, userIDs)
	return args.Error(0)
}
func (m *MockNakamaModule) UsersUnbanId(ctx context.Context, userIDs []string) error {
	args := m.Called(ctx, userIDs)
	return args.Error(0)
}
func (m *MockNakamaModule) LinkApple(ctx context.Context, userID, token string) error {
	args := m.Called(ctx, userID, token)
	return args.Error(0)
}
func (m *MockNakamaModule) LinkCustom(ctx context.Context, userID, customID string) error {
	args := m.Called(ctx, userID, customID)
	return args.Error(0)
}
func (m *MockNakamaModule) LinkDevice(ctx context.Context, userID, deviceID string) error {
	args := m.Called(ctx, userID, deviceID)
	return args.Error(0)
}
func (m *MockNakamaModule) LinkEmail(ctx context.Context, userID, email, password string) error {
	args := m.Called(ctx, userID, email, password)
	return args.Error(0)
}
func (m *MockNakamaModule) LinkFacebook(ctx context.Context, userID, username, token string, importFriends bool) error {
	args := m.Called(ctx, userID, username, token, importFriends)
	return args.Error(0)
}
func (m *MockNakamaModule) LinkFacebookInstantGame(ctx context.Context, userID, signedPlayerInfo string) error {
	args := m.Called(ctx, userID, signedPlayerInfo)
	return args.Error(0)
}
func (m *MockNakamaModule) LinkGameCenter(ctx context.Context, userID, playerID, bundleID string, timestamp int64, salt, signature, publicKeyUrl string) error {
	args := m.Called(ctx, userID, playerID, bundleID, timestamp, salt, signature, publicKeyUrl)
	return args.Error(0)
}
func (m *MockNakamaModule) LinkGoogle(ctx context.Context, userID, token string) error {
	args := m.Called(ctx, userID, token)
	return args.Error(0)
}
func (m *MockNakamaModule) LinkSteam(ctx context.Context, userID, username, token string, importFriends bool) error {
	args := m.Called(ctx, userID, username, token, importFriends)
	return args.Error(0)
}
func (m *MockNakamaModule) CronPrev(expression string, timestamp int64) (int64, error) {
	args := m.Called(expression, timestamp)
	return args.Get(0).(int64), args.Error(1)
}
func (m *MockNakamaModule) CronNext(expression string, timestamp int64) (int64, error) {
	args := m.Called(expression, timestamp)
	return args.Get(0).(int64), args.Error(1)
}
func (m *MockNakamaModule) ReadFile(path string) (*os.File, error) {
	args := m.Called(path)
	return args.Get(0).(*os.File), args.Error(1)
}
func (m *MockNakamaModule) UnlinkApple(ctx context.Context, userID, token string) error {
	args := m.Called(ctx, userID, token)
	return args.Error(0)
}
func (m *MockNakamaModule) UnlinkCustom(ctx context.Context, userID, customID string) error {
	args := m.Called(ctx, userID, customID)
	return args.Error(0)
}
func (m *MockNakamaModule) UnlinkDevice(ctx context.Context, userID, deviceID string) error {
	args := m.Called(ctx, userID, deviceID)
	return args.Error(0)
}
func (m *MockNakamaModule) UnlinkEmail(ctx context.Context, userID, email string) error {
	args := m.Called(ctx, userID, email)
	return args.Error(0)
}
func (m *MockNakamaModule) UnlinkFacebook(ctx context.Context, userID, token string) error {
	args := m.Called(ctx, userID, token)
	return args.Error(0)
}
func (m *MockNakamaModule) UnlinkFacebookInstantGame(ctx context.Context, userID, signedPlayerInfo string) error {
	args := m.Called(ctx, userID, signedPlayerInfo)
	return args.Error(0)
}
func (m *MockNakamaModule) UnlinkGameCenter(ctx context.Context, userID, playerID, bundleID string, timestamp int64, salt, signature, publicKeyUrl string) error {
	args := m.Called(ctx, userID, playerID, bundleID, timestamp, salt, signature, publicKeyUrl)
	return args.Error(0)
}
func (m *MockNakamaModule) UnlinkGoogle(ctx context.Context, userID, token string) error {
	args := m.Called(ctx, userID, token)
	return args.Error(0)
}
func (m *MockNakamaModule) UnlinkSteam(ctx context.Context, userID, token string) error {
	args := m.Called(ctx, userID, token)
	return args.Error(0)
}
func (m *MockNakamaModule) StreamUserJoin(mode uint8, subject, subcontext, label, userID, sessionID string, hidden, persistence bool, status string) (bool, error) {
	args := m.Called(mode, subject, subcontext, label, userID, sessionID, hidden, persistence, status)
	return args.Bool(0), args.Error(1)
}
func (m *MockNakamaModule) StreamUserUpdate(mode uint8, subject, subcontext, label, userID, sessionID string, hidden, persistence bool, status string) error {
	args := m.Called(mode, subject, subcontext, label, userID, sessionID, hidden, persistence, status)
	return args.Error(0)
}
func (m *MockNakamaModule) StreamUserLeave(mode uint8, subject, subcontext, label, userID, sessionID string) error {
	args := m.Called(mode, subject, subcontext, label, userID, sessionID)
	return args.Error(0)
}
func (m *MockNakamaModule) StreamCount(mode uint8, subject, subcontext, label string) (int, error) {
	args := m.Called(mode, subject, subcontext, label)
	return args.Int(0), args.Error(1)
}
func (m *MockNakamaModule) StreamClose(mode uint8, subject, subcontext, label string) error {
	args := m.Called(mode, subject, subcontext, label)
	return args.Error(0)
}
func (m *MockNakamaModule) SessionLogout(userID, token, refreshToken string) error {
	args := m.Called(userID, token, refreshToken)
	return args.Error(0)
}
func (m *MockNakamaModule) MatchCreate(ctx context.Context, module string, params map[string]interface{}) (string, error) {
	args := m.Called(ctx, module, params)
	return args.String(0), args.Error(1)
}
func (m *MockNakamaModule) MatchGet(ctx context.Context, id string) (*api.Match, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*api.Match), args.Error(1)
}
func (m *MockNakamaModule) MatchList(ctx context.Context, limit int, authoritative bool, label string, minSize, maxSize *int, query string) ([]*api.Match, error) {
	args := m.Called(ctx, limit, authoritative, label, minSize, maxSize, query)
	return args.Get(0).([]*api.Match), args.Error(1)
}
func (m *MockNakamaModule) MatchSignal(ctx context.Context, id string, data string) (string, error) {
	args := m.Called(ctx, id, data)
	return args.String(0), args.Error(1)
}
func (m *MockNakamaModule) NotificationSend(ctx context.Context, userID, subject string, content map[string]interface{}, code int, sender string, persistent bool) error {
	args := m.Called(ctx, userID, subject, content, code, sender, persistent)
	return args.Error(0)
}
func (m *MockNakamaModule) NotificationsList(ctx context.Context, userID string, limit int, cursor string) ([]*api.Notification, string, error) {
	args := m.Called(ctx, userID, limit, cursor)
	return args.Get(0).([]*api.Notification), args.String(1), args.Error(2)
}
func (m *MockNakamaModule) NotificationSendAll(ctx context.Context, subject string, content map[string]interface{}, code int, persistent bool) error {
	args := m.Called(ctx, subject, content, code, persistent)
	return args.Error(0)
}
func (m *MockNakamaModule) NotificationsDeleteId(ctx context.Context, userID string, ids []string) error {
	args := m.Called(ctx, userID, ids)
	return args.Error(0)
}
func (m *MockNakamaModule) StorageIndexList(ctx context.Context, callerID, indexName, query string, limit int, order []string, cursor string) (*api.StorageObjects, string, error) {
	args := m.Called(ctx, callerID, indexName, query, limit, order, cursor)
	return args.Get(0).(*api.StorageObjects), args.String(1), args.Error(2)
}
func (m *MockNakamaModule) LeaderboardCreate(ctx context.Context, id string, authoritative bool, sortOrder, operator, resetSchedule string, metadata map[string]interface{}, enableRanks bool) error {
	args := m.Called(ctx, id, authoritative, sortOrder, operator, resetSchedule, metadata, enableRanks)
	return args.Error(0)
}
func (m *MockNakamaModule) LeaderboardDelete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockNakamaModule) LeaderboardList(limit int, cursor string) (*api.LeaderboardList, error) {
	args := m.Called(limit, cursor)
	return args.Get(0).(*api.LeaderboardList), args.Error(1)
}
func (m *MockNakamaModule) LeaderboardRanksDisable(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockNakamaModule) LeaderboardRecordsList(ctx context.Context, id string, ownerIDs []string, limit int, cursor string, expiry int64) (records []*api.LeaderboardRecord, ownerRecords []*api.LeaderboardRecord, nextCursor string, prevCursor string, err error) {
	args := m.Called(ctx, id, ownerIDs, limit, cursor, expiry)
	return args.Get(0).([]*api.LeaderboardRecord), args.Get(1).([]*api.LeaderboardRecord), args.String(2), args.String(3), args.Error(4)
}
func (m *MockNakamaModule) LeaderboardRecordsListCursorFromRank(id string, rank, overrideExpiry int64) (string, error) {
	args := m.Called(id, rank, overrideExpiry)
	return args.String(0), args.Error(1)
}
func (m *MockNakamaModule) LeaderboardRecordWrite(ctx context.Context, id, ownerID, username string, score, subscore int64, metadata map[string]interface{}, overrideOperator *int) (*api.LeaderboardRecord, error) {
	args := m.Called(ctx, id, ownerID, username, score, subscore, metadata, overrideOperator)
	return args.Get(0).(*api.LeaderboardRecord), args.Error(1)
}
func (m *MockNakamaModule) LeaderboardRecordDelete(ctx context.Context, id, ownerID string) error {
	args := m.Called(ctx, id, ownerID)
	return args.Error(0)
}
func (m *MockNakamaModule) LeaderboardsGetId(ctx context.Context, ids []string) ([]*api.Leaderboard, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]*api.Leaderboard), args.Error(1)
}
func (m *MockNakamaModule) LeaderboardRecordsHaystack(ctx context.Context, id, ownerID string, limit int, cursor string, expiry int64) (*api.LeaderboardRecordList, error) {
	args := m.Called(ctx, id, ownerID, limit, cursor, expiry)
	return args.Get(0).(*api.LeaderboardRecordList), args.Error(1)
}
func (m *MockNakamaModule) PurchaseValidateApple(ctx context.Context, userID, receipt string, persist bool, passwordOverride ...string) (*api.ValidatePurchaseResponse, error) {
	args := m.Called(ctx, userID, receipt, persist, passwordOverride)
	return args.Get(0).(*api.ValidatePurchaseResponse), args.Error(1)
}
func (m *MockNakamaModule) PurchaseValidateGoogle(ctx context.Context, userID, receipt string, persist bool, overrides ...struct {
	ClientEmail string
	PrivateKey  string
}) (*api.ValidatePurchaseResponse, error) {
	args := m.Called(ctx, userID, receipt, persist, overrides)
	return args.Get(0).(*api.ValidatePurchaseResponse), args.Error(1)
}
func (m *MockNakamaModule) PurchaseValidateHuawei(ctx context.Context, userID, signature, inAppPurchaseData string, persist bool) (*api.ValidatePurchaseResponse, error) {
	args := m.Called(ctx, userID, signature, inAppPurchaseData, persist)
	return args.Get(0).(*api.ValidatePurchaseResponse), args.Error(1)
}
func (m *MockNakamaModule) PurchaseValidateFacebookInstant(ctx context.Context, userID, signedRequest string, persist bool) (*api.ValidatePurchaseResponse, error) {
	args := m.Called(ctx, userID, signedRequest, persist)
	return args.Get(0).(*api.ValidatePurchaseResponse), args.Error(1)
}
func (m *MockNakamaModule) PurchasesList(ctx context.Context, userID string, limit int, cursor string) (*api.PurchaseList, error) {
	args := m.Called(ctx, userID, limit, cursor)
	return args.Get(0).(*api.PurchaseList), args.Error(1)
}
func (m *MockNakamaModule) PurchaseGetByTransactionId(ctx context.Context, transactionID string) (*api.ValidatedPurchase, error) {
	args := m.Called(ctx, transactionID)
	return args.Get(0).(*api.ValidatedPurchase), args.Error(1)
}
func (m *MockNakamaModule) SubscriptionValidateApple(ctx context.Context, userID, receipt string, persist bool, passwordOverride ...string) (*api.ValidateSubscriptionResponse, error) {
	args := m.Called(ctx, userID, receipt, persist, passwordOverride)
	return args.Get(0).(*api.ValidateSubscriptionResponse), args.Error(1)
}
func (m *MockNakamaModule) SubscriptionValidateGoogle(ctx context.Context, userID, receipt string, persist bool, overrides ...struct {
	ClientEmail string
	PrivateKey  string
}) (*api.ValidateSubscriptionResponse, error) {
	args := m.Called(ctx, userID, receipt, persist, overrides)
	return args.Get(0).(*api.ValidateSubscriptionResponse), args.Error(1)
}
func (m *MockNakamaModule) SubscriptionsList(ctx context.Context, userID string, limit int, cursor string) (*api.SubscriptionList, error) {
	args := m.Called(ctx, userID, limit, cursor)
	return args.Get(0).(*api.SubscriptionList), args.Error(1)
}
func (m *MockNakamaModule) SubscriptionGetByProductId(ctx context.Context, userID, productID string) (*api.ValidatedSubscription, error) {
	args := m.Called(ctx, userID, productID)
	return args.Get(0).(*api.ValidatedSubscription), args.Error(1)
}
func (m *MockNakamaModule) TournamentCreate(ctx context.Context, id string, authoritative bool, sortOrder, operator, resetSchedule string, metadata map[string]interface{}, title, description string, category, startTime, endTime, duration, maxSize, maxNumScore int, joinRequired, enableRanks bool) error {
	args := m.Called(ctx, id, authoritative, sortOrder, operator, resetSchedule, metadata, title, description, category, startTime, endTime, duration, maxSize, maxNumScore, joinRequired, enableRanks)
	return args.Error(0)
}
func (m *MockNakamaModule) TournamentDelete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockNakamaModule) TournamentAddAttempt(ctx context.Context, id, ownerID string, count int) error {
	args := m.Called(ctx, id, ownerID, count)
	return args.Error(0)
}
func (m *MockNakamaModule) TournamentJoin(ctx context.Context, id, ownerID, username string) error {
	args := m.Called(ctx, id, ownerID, username)
	return args.Error(0)
}
func (m *MockNakamaModule) TournamentsGetId(ctx context.Context, tournamentIDs []string) ([]*api.Tournament, error) {
	args := m.Called(ctx, tournamentIDs)
	return args.Get(0).([]*api.Tournament), args.Error(1)
}
func (m *MockNakamaModule) TournamentList(ctx context.Context, categoryStart, categoryEnd, startTime, endTime, limit int, cursor string) (*api.TournamentList, error) {
	args := m.Called(ctx, categoryStart, categoryEnd, startTime, endTime, limit, cursor)
	return args.Get(0).(*api.TournamentList), args.Error(1)
}
func (m *MockNakamaModule) TournamentRanksDisable(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *MockNakamaModule) TournamentRecordsList(ctx context.Context, tournamentId string, ownerIDs []string, limit int, cursor string, overrideExpiry int64) (records []*api.LeaderboardRecord, ownerRecords []*api.LeaderboardRecord, prevCursor string, nextCursor string, err error) {
	args := m.Called(ctx, tournamentId, ownerIDs, limit, cursor, overrideExpiry)
	return args.Get(0).([]*api.LeaderboardRecord), args.Get(1).([]*api.LeaderboardRecord), args.String(2), args.String(3), args.Error(4)
}
func (m *MockNakamaModule) TournamentRecordWrite(ctx context.Context, id, ownerID, username string, score, subscore int64, metadata map[string]interface{}, operatorOverride *int) (*api.LeaderboardRecord, error) {
	args := m.Called(ctx, id, ownerID, username, score, subscore, metadata, operatorOverride)
	return args.Get(0).(*api.LeaderboardRecord), args.Error(1)
}
func (m *MockNakamaModule) TournamentRecordDelete(ctx context.Context, id, ownerID string) error {
	args := m.Called(ctx, id, ownerID)
	return args.Error(0)
}
func (m *MockNakamaModule) TournamentsGetCategory(categoryStart, categoryEnd, startTime, endTime, limit int, cursor string) (*api.TournamentList, error) {
	args := m.Called(categoryStart, categoryEnd, startTime, endTime, limit, cursor)
	return args.Get(0).(*api.TournamentList), args.Error(1)
}
func (m *MockNakamaModule) TournamentRecordListCursorFromRank(id string, rank, overrideExpiry int64) (string, error) {
	args := m.Called(id, rank, overrideExpiry)
	return args.String(0), args.Error(1)
}
func (m *MockNakamaModule) TournamentRecordWriteOverride(ctx context.Context, id, ownerID, username string, score, subscore int64, metadata map[string]interface{}, operatorOverride *int) (*api.LeaderboardRecord, error) {
	args := m.Called(ctx, id, ownerID, username, score, subscore, metadata, operatorOverride)
	return args.Get(0).(*api.LeaderboardRecord), args.Error(1)
}
func (m *MockNakamaModule) ChannelJoin(ctx context.Context, target, typeStr, persistence, hidden string) (string, error) {
	args := m.Called(ctx, target, typeStr, persistence, hidden)
	return args.String(0), args.Error(1)
}
func (m *MockNakamaModule) ChannelLeave(ctx context.Context, channelID string) error {
	args := m.Called(ctx, channelID)
	return args.Error(0)
}
func (m *MockNakamaModule) ChannelMessageList(ctx context.Context, channelID string, limit int, forward bool, cursor string) ([]*api.ChannelMessage, string, error) {
	return nil, "", nil
}
func (m *MockNakamaModule) GroupDelete(ctx context.Context, groupID string) error {
	return nil
}
func (m *MockNakamaModule) GroupUserList(ctx context.Context, groupID string, limit int, state int, cursor string) ([]*api.GroupUserList_GroupUser, string, error) {
	return nil, "", nil
}
func (m *MockNakamaModule) GroupUsersUnban(ctx context.Context, groupID string, userIDs []string) error {
	return nil
}
func (m *MockNakamaModule) GroupJoinRequestsList(ctx context.Context, groupID string, limit int, cursor string) ([]*GroupJoinRequest, string, error) {
	return nil, "", nil
}
func (m *MockNakamaModule) GroupJoinRequestAccept(ctx context.Context, groupID, userID string) error {
	return nil
}
func (m *MockNakamaModule) GroupJoinRequestReject(ctx context.Context, groupID, userID string) error {
	return nil
}
func (m *MockNakamaModule) GroupList(ctx context.Context, name string, limit int, cursor string) ([]*api.Group, string, error) {
	return nil, "", nil
}
func (m *MockNakamaModule) GroupGetId(ctx context.Context, groupIDs []string) ([]*api.Group, error) {
	return nil, nil
}
func (m *MockNakamaModule) GroupGetUserId(ctx context.Context, userIDs []string) ([]*api.Group, error) {
	return nil, nil
}
func (m *MockNakamaModule) GroupLeave(ctx context.Context, groupID, userID string) error {
	return nil
}
func (m *MockNakamaModule) GroupAddEdge(ctx context.Context, groupID, userID string, state int) error {
	return nil
}
func (m *MockNakamaModule) GroupDeleteEdge(ctx context.Context, groupID, userID string) error {
	return nil
}
func (m *MockNakamaModule) GroupUpdateEdge(ctx context.Context, groupID, userID string, state int) error {
	return nil
}

// ChannelIdBuild is required by the NakamaModule interface
func (m *MockNakamaModule) ChannelIdBuild(ctx context.Context, sender string, target string, chanType runtime.ChannelType) (string, error) {
	return "mock_channel_id", nil
}

package mock

import (
	"bytes"
	"time"

	"github.com/chadsec1/decoyim/config"
	"github.com/chadsec1/decoyim/coylog"
	"github.com/chadsec1/decoyim/otrclient"
	"github.com/chadsec1/decoyim/roster"
	"github.com/chadsec1/decoyim/session/access"
	sdata "github.com/chadsec1/decoyim/session/data"
	"github.com/chadsec1/decoyim/session/muc"
	mdata "github.com/chadsec1/decoyim/session/muc/data"
	"github.com/chadsec1/decoyim/tls"
	"github.com/chadsec1/decoyim/xmpp/data"
	xi "github.com/chadsec1/decoyim/xmpp/interfaces"
	"github.com/chadsec1/decoyim/xmpp/jid"

	"github.com/coyim/otr3"
)

// SessionMock is a mock of the Session interface
type SessionMock struct{}

// ApprovePresenceSubscription is the implementation for Session interface
func (*SessionMock) ApprovePresenceSubscription(jid.WithoutResource, string) error {
	return nil
}

// AwaitVersionReply is the implementation for Session interface
func (*SessionMock) AwaitVersionReply(<-chan data.Stanza, string) {}

// Close is the implementation for Session interface
func (*SessionMock) Close() {}

// AutoApprove is the implementation for Session interface
func (*SessionMock) AutoApprove(string) {}

// CommandManager is the implementation for Session interface
func (*SessionMock) CommandManager() otrclient.CommandManager {
	return nil
}

// Config is the implementation for Session interface
func (*SessionMock) Config() *config.ApplicationConfig {
	return nil
}

// Conn is the implementation for Session interface
func (*SessionMock) Conn() xi.Conn {
	return nil
}

// Connect is the implementation for Session interface
func (*SessionMock) Connect(string, tls.Verifier) error {
	return nil
}

// ConversationManager is the implementation for Session interface
func (*SessionMock) ConversationManager() otrclient.ConversationManager {
	return nil
}

// DenyPresenceSubscription is the implementation for Session interface
func (*SessionMock) DenyPresenceSubscription(jid.WithoutResource, string) error {
	return nil
}

// DisplayName is the implementation for Session interface
func (*SessionMock) DisplayName() string {
	return ""
}

// EncryptAndSendTo is the implementation for Session interface
func (*SessionMock) EncryptAndSendTo(jid.Any, string) (int, bool, error) {
	return 0, false, nil
}

// GetConfig is the implementation for Session interface
func (*SessionMock) GetConfig() *config.Account {
	return nil
}

// GetInMemoryLog is the implementation for Session interface
func (*SessionMock) GetInMemoryLog() *bytes.Buffer {
	return nil
}

// GroupDelimiter is the implementation for Session interface
func (*SessionMock) GroupDelimiter() string {
	return ""
}

// HandleConfirmOrDeny is the implementation for Session interface
func (*SessionMock) HandleConfirmOrDeny(jid.WithoutResource, bool) {}

// IsConnected is the implementation for Session interface
func (*SessionMock) IsConnected() bool {
	return false
}

// IsDisconnected is the implementation for Session interface
func (*SessionMock) IsDisconnected() bool {
	return false
}

// ManuallyEndEncryptedChat is the implementation for Session interface
func (*SessionMock) ManuallyEndEncryptedChat(jid.Any) error {
	return nil
}

// PrivateKeys is the implementation for Session interface
func (*SessionMock) PrivateKeys() []otr3.PrivateKey {
	return nil
}

// R is the implementation for Session interface
func (*SessionMock) R() *roster.List {
	return nil
}

// ReloadKeys is the implementation for Session interface
func (*SessionMock) ReloadKeys() {}

// RemoveContact is the implementation for Session interface
func (*SessionMock) RemoveContact(string) {}

// RequestPresenceSubscription is the implementation for Session interface
func (*SessionMock) RequestPresenceSubscription(jid.WithoutResource, string) error {
	return nil
}

// Send is the implementation for Session interface
func (*SessionMock) Send(jid.Any, string, bool) error {
	return nil
}

// SendMUCMessage is the implementation for Session interface
func (*SessionMock) SendMUCMessage(to, from, body string) error {
	return nil
}

// UpdateRoomSubject is the implementation for Session interface
func (*SessionMock) UpdateRoomSubject(roomID jid.Bare, from, subject string) error {
	return nil
}

// UpdateOccupantAffiliations is the implementation for Session interface
func (*SessionMock) UpdateOccupantAffiliations(jid.Bare, []*muc.RoomOccupantItem) (<-chan bool, <-chan error) {
	return nil, nil
}

// GetRoomOccupantsByAffiliation is the implementation for Session interface
func (*SessionMock) GetRoomOccupantsByAffiliation(roomID jid.Bare, a mdata.Affiliation) (<-chan []*muc.RoomOccupantItem, <-chan error) {
	return nil, nil
}

// SendPing is the implementation for Session interface
func (*SessionMock) SendPing() {}

// SetCommandManager is the implementation for Session interface
func (*SessionMock) SetCommandManager(otrclient.CommandManager) {}

// SetConnector is the implementation for Session interface
func (*SessionMock) SetConnector(access.Connector) {}

// SetLastActionTime is the implementation for Session interface
func (*SessionMock) SetLastActionTime(time.Time) {}

// SetWantToBeOnline is the implementation for Session interface
func (*SessionMock) SetWantToBeOnline(bool) {}

// Subscribe is the implementation for Session interface
func (*SessionMock) Subscribe(chan<- interface{}) {}

// Timeout is the implementation for Session interface
func (*SessionMock) Timeout(data.Cookie, time.Time) {}

// StartSMP is the implementation for Session interface
func (*SessionMock) StartSMP(jid.WithResource, string, string) {}

// FinishSMP is the implementation for Session interface
func (*SessionMock) FinishSMP(jid.WithResource, string) {}

// AbortSMP is the implementation for Session interface
func (*SessionMock) AbortSMP(jid.WithResource) {}

// PublishEvent is the implementation for Session interface
func (*SessionMock) PublishEvent(interface{}) {}

// SendIQError is the implementation for Session interface
func (*SessionMock) SendIQError(*data.ClientIQ, interface{}) {}

// SendIQResult is the implementation for Session interface
func (*SessionMock) SendIQResult(*data.ClientIQ, interface{}) {}

// SendFileTo is the implementation for Session interface
func (*SessionMock) SendFileTo(jid.Any, string, func() bool, func(bool)) *sdata.FileTransferControl {
	return nil
}

// SendDirTo is the implementation for Session interface
func (*SessionMock) SendDirTo(jid.Any, string, func() bool, func(bool)) *sdata.FileTransferControl {
	return nil
}

// CreateSymmetricKeyFor is the implementation for Session interface
func (*SessionMock) CreateSymmetricKeyFor(jid.Any) []byte {
	return nil
}

// GetAndWipeSymmetricKeyFor is the implementation for Session interface
func (*SessionMock) GetAndWipeSymmetricKeyFor(jid.Any) []byte {
	return nil
}

// HasRoom is the implementation for Session interface
func (s *SessionMock) HasRoom(jid.Bare, chan<- *muc.RoomListing) (<-chan bool, <-chan error) {
	return nil, nil
}

// GetRoomListing is the implementation for Session interface
func (s *SessionMock) GetRoomListing(jid.Bare, chan<- *muc.RoomListing) {}

// RefreshRoomProperties is the implementation for Session interface
func (s *SessionMock) RefreshRoomProperties(jid.Bare) {}

// GetRooms is the implementation for Session interface
func (*SessionMock) GetRooms(jid.Domain, string) (<-chan *muc.RoomListing, <-chan *muc.ServiceListing, <-chan error) {
	return nil, nil, nil
}

// JoinRoom is the implementation for Session interface
func (s *SessionMock) JoinRoom(jid.Bare, string, string) error {
	return nil
}

// CreateInstantRoom is the implementation for session interface
func (*SessionMock) CreateInstantRoom(jid.Bare) (<-chan bool, <-chan error) {
	return nil, nil
}

// CreateReservedRoom is the implementation for session interface
func (*SessionMock) CreateReservedRoom(jid.Bare) (<-chan *muc.RoomConfigForm, <-chan error) {
	return nil, nil
}

// GetRoomConfigurationForm is the implementation for session interface
func (*SessionMock) GetRoomConfigurationForm(jid.Bare) (<-chan *muc.RoomConfigForm, <-chan error) {
	return nil, nil
}

// SubmitRoomConfigurationForm is the implementation for session interface
func (*SessionMock) SubmitRoomConfigurationForm(jid.Bare, *muc.RoomConfigForm) (<-chan bool, <-chan *muc.SubmitFormError) {
	return nil, nil
}

// CancelRoomConfiguration is the implementation for session interface
func (*SessionMock) CancelRoomConfiguration(jid.Bare) <-chan error {
	return nil
}

// GetChatServices is the implementation for session interface
func (*SessionMock) GetChatServices(jid.Domain) (<-chan jid.Domain, <-chan error, func()) {
	return nil, nil, nil
}

// DestroyRoom is the implementation for session interface
func (*SessionMock) DestroyRoom(jid.Bare, string, jid.Bare, string) (<-chan bool, <-chan error) {
	return nil, nil
}

// UpdateOccupantAffiliation is the implementation for session interface
func (*SessionMock) UpdateOccupantAffiliation(roomID jid.Bare, occupantNickname string, occupantRealJID jid.Full, affiliation mdata.Affiliation, reason string) (<-chan bool, <-chan error) {
	return nil, nil
}

// UpdateOccupantRole is the implementation for session interface
func (*SessionMock) UpdateOccupantRole(jid.Bare, string, mdata.Role, string) (<-chan bool, <-chan error) {
	return nil, nil
}

// Log is the implementation for session interface
func (*SessionMock) Log() coylog.Logger {
	return nil
}

// LeaveRoom is the implementation for session interface
func (*SessionMock) LeaveRoom(room jid.Bare, nickname string) (<-chan bool, <-chan error) {
	return nil, nil
}

// GetRoom is the implementation for session interface
func (*SessionMock) GetRoom(jid.Bare) (*muc.Room, bool) {
	return nil, false
}

// NewRoom is the implementation for session interface
func (*SessionMock) NewRoom(jid.Bare) *muc.Room {
	return nil
}

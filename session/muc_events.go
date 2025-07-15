package session

import (
	"time"

	"github.com/chadsec1/decoyim/session/events"
	"github.com/chadsec1/decoyim/session/muc"
	"github.com/chadsec1/decoyim/session/muc/data"
	"github.com/chadsec1/decoyim/xmpp/jid"
)

func (m *mucManager) publishRoomEvent(roomID jid.Bare, ev events.MUC) {
	room, exists := m.roomManager.GetRoom(roomID)
	if !exists {
		m.log.WithField("room", roomID).Error("Trying to publish an event in a room that does not exist")
		return
	}
	room.Publish(ev)
}

func (m *mucManager) roomCreated(roomID jid.Bare) {
	ev := events.MUCRoomCreated{}
	ev.Room = roomID

	m.publishEvent(ev)
}

func (m *mucManager) roomRenamed(roomID jid.Bare) {
	m.publishRoomEvent(roomID, events.MUCRoomRenamed{})
}

func (m *mucManager) occupantLeft(roomID jid.Bare, op *muc.OccupantPresenceInfo) {
	ev := events.MUCOccupantLeft{}
	ev.Nickname = op.Nickname
	ev.RealJid = op.RealJid
	ev.Affiliation = op.AffiliationRole.Affiliation
	ev.Role = op.AffiliationRole.Role

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) occupantJoined(roomID jid.Bare, op *muc.OccupantPresenceInfo) {
	ev := events.MUCOccupantJoined{}
	ev.Nickname = op.Nickname
	ev.RealJid = op.RealJid
	ev.Affiliation = op.AffiliationRole.Affiliation
	ev.Role = op.AffiliationRole.Role

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) occupantUpdate(roomID jid.Bare, op *muc.OccupantPresenceInfo) {
	ev := events.MUCOccupantUpdated{}
	ev.Nickname = op.Nickname
	ev.RealJid = op.RealJid
	ev.Affiliation = op.AffiliationRole.Affiliation
	ev.Role = op.AffiliationRole.Role
	ev.Status = op.Status
	ev.StatusMessage = op.StatusMessage

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) loggingEnabled(roomID jid.Bare) {
	m.publishRoomEvent(roomID, events.MUCLoggingEnabled{})
}

func (m *mucManager) loggingDisabled(roomID jid.Bare) {
	m.publishRoomEvent(roomID, events.MUCLoggingDisabled{})
}

func (m *mucManager) selfOccupantJoined(roomID jid.Bare, op *muc.OccupantPresenceInfo) {
	ev := events.MUCSelfOccupantJoined{}
	ev.Nickname = op.Nickname
	ev.RealJid = op.RealJid
	ev.Affiliation = op.AffiliationRole.Affiliation
	ev.Role = op.AffiliationRole.Role

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) liveMessageReceived(roomID jid.Bare, nickname, message string, timestamp time.Time) {
	m.appendHistoryMessage(roomID, nickname, message, timestamp)

	ev := events.MUCLiveMessageReceived{}
	ev.Nickname = nickname
	ev.Message = message
	ev.Timestamp = timestamp

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) delayedMessageReceived(roomID jid.Bare, nickname, message string, timestamp time.Time) {
	ev := events.MUCDelayedMessageReceived{}
	ev.Nickname = nickname
	ev.Message = message
	ev.Timestamp = timestamp

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) discussionHistoryReceived(roomID jid.Bare, history *data.DiscussionHistory) {
	ev := events.MUCDiscussionHistoryReceived{}
	ev.History = history

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) subjectReceived(roomID jid.Bare, subject string) {
	ev := events.MUCSubjectReceived{}
	ev.Subject = subject

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) joinRoomFinished(roomID jid.Bare) {
	ev := events.MUCJoinRoomFinished{}
	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) subjectUpdated(roomID jid.Bare, nickname, subject string) {
	ev := events.MUCSubjectUpdated{}
	ev.Nickname = nickname
	ev.Subject = subject

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) nonAnonymousRoom(roomID jid.Bare) {
	m.roomAnonymityChanged(roomID, "no")
}

func (m *mucManager) semiAnonymousRoom(roomID jid.Bare) {
	m.roomAnonymityChanged(roomID, "semi")
}

func (m *mucManager) roomAnonymityChanged(roomID jid.Bare, value string) {
	ev := events.MUCRoomAnonymityChanged{}
	ev.AnonymityLevel = value

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) roomDiscoInfoReceived(roomID jid.Bare, di data.RoomDiscoInfo) {
	ev := events.MUCRoomDiscoInfoReceived{}
	ev.DiscoInfo = di

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) roomDiscoInfoRequestTimeout(roomID jid.Bare) {
	m.publishRoomEvent(roomID, events.MUCRoomConfigTimeout{})
}

func (m *mucManager) roomConfigChanged(roomID jid.Bare, changes []data.RoomConfigType, discoInfo data.RoomDiscoInfo) {
	ev := events.MUCRoomConfigChanged{}
	ev.Changes = changes
	ev.DiscoInfo = discoInfo

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) occupantRemoved(roomID jid.Bare, nickname string) {
	ev := events.MUCOccupantRemoved{}
	ev.Nickname = nickname

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) removeSelfOccupant(roomID jid.Bare) {
	ev := events.MUCSelfOccupantRemoved{}

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) roomDestroyed(roomID jid.Bare, reason string, alternativeRoomID jid.Bare, password string) {
	ev := events.MUCRoomDestroyed{}
	ev.Reason = reason
	ev.AlternativeRoom = alternativeRoomID
	ev.Password = password

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) occupantAffiliationRoleUpdated(roomID jid.Bare, affiliationRoleUpdate data.AffiliationRoleUpdate) {
	ev := events.MUCOccupantAffiliationRoleUpdated{}
	ev.AffiliationRoleUpdate = affiliationRoleUpdate

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) selfOccupantAffiliationRoleUpdated(roomID jid.Bare, selfAffiliationRoleUpdate data.SelfAffiliationRoleUpdate) {
	ev := events.MUCSelfOccupantAffiliationRoleUpdated{}
	ev.AffiliationRoleUpdate = selfAffiliationRoleUpdate

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) occupantAffiliationUpdated(roomID jid.Bare, affiliationUpdate data.AffiliationUpdate) {
	ev := events.MUCOccupantAffiliationUpdated{}
	ev.AffiliationUpdate = affiliationUpdate

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) selfOccupantAffiliationUpdated(roomID jid.Bare, affiliationUpdate data.SelfAffiliationUpdate) {
	ev := events.MUCSelfOccupantAffiliationUpdated{}
	ev.AffiliationUpdate = affiliationUpdate

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) occupantRoleUpdated(roomID jid.Bare, roleUpdate data.RoleUpdate) {
	ev := events.MUCOccupantRoleUpdated{}
	ev.RoleUpdate = roleUpdate

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) selfOccupantRoleUpdated(roomID jid.Bare, roleUpdate data.SelfRoleUpdate) {
	ev := events.MUCSelfOccupantRoleUpdated{}
	ev.RoleUpdate = roleUpdate

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) occupantKicked(roomID jid.Bare, roleUpdate data.RoleUpdate) {
	ev := events.MUCOccupantKicked{}
	ev.RoleUpdate = roleUpdate

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) selfOccupantKicked(roomID jid.Bare, roleUpdate data.SelfRoleUpdate) {
	ev := events.MUCSelfOccupantKicked{}
	ev.RoleUpdate = roleUpdate

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) accountAffiliationUpdated(roomID jid.Bare, accountAddress jid.Any, affiliation data.Affiliation) {
	m.publishRoomEvent(roomID, events.MUCAccountAffiliationUpdated{
		AccountAddress: accountAddress,
		Affiliation:    affiliation,
	})
}

func (m *mucManager) occupantRemovedOnAffiliationChange(roomID jid.Bare, nickname string) {
	ev := events.MUCOccupantRemovedOnAffiliationChange{}
	ev.Nickname = nickname

	m.publishRoomEvent(roomID, ev)
}

func (m *mucManager) selfOccupantRemovedOnAffiliationChange(roomID jid.Bare) {
	m.publishRoomEvent(roomID, events.MUCSelfOccupantRemovedOnAffiliationChange{})
}

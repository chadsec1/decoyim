package session

import (
	"github.com/chadsec1/decoyim/session/muc"
	"github.com/chadsec1/decoyim/session/muc/data"
	xmppData "github.com/chadsec1/decoyim/xmpp/data"
	"github.com/chadsec1/decoyim/xmpp/jid"
	log "github.com/sirupsen/logrus"
)

func newMUCRoomOccupant(nickname string, affiliation data.Affiliation, role data.Role, realJid jid.Full) *muc.Occupant {
	return &muc.Occupant{
		Nickname:    nickname,
		Affiliation: affiliation,
		Role:        role,
		RealJid:     realJid,
	}
}

func (m *mucManager) handleOccupantUpdate(roomID jid.Bare, op *muc.OccupantPresenceInfo) {
	l := m.log.WithFields(log.Fields{
		"room":     roomID,
		"occupant": op.Nickname,
		"method":   "handleOccupantUpdate",
	})

	room, ok := m.roomManager.GetRoom(roomID)
	if !ok {
		l.Error("Trying to get a room that is not in the room manager")
		return
	}

	occupantUpdateInfo := m.newOccupantPresenceUpdateData(room, op)

	updated := room.Roster().UpdateOrAddOccupant(op)
	// This is a temporary validation while 'state machine' pattern is implemented.
	if room.IsSelfOccupantInTheRoom() {
		if updated {
			m.handleOccupantAffiliationRoleUpdate(occupantUpdateInfo)
			m.occupantUpdate(roomID, op)
		} else {
			m.occupantJoined(roomID, op)
		}
	}
}

type occupantPresenceUpdateData struct {
	room                *muc.Room
	currentOccupantInfo *muc.OccupantPresenceInfo
	newOccupantInfo     *muc.OccupantPresenceInfo
	actorOccupant       *data.Actor
}

func (m *mucManager) occupantPresenceCurrentInfo(room *muc.Room, nickname string) (*muc.OccupantPresenceInfo, bool) {
	occupant, exists := room.Roster().GetOccupant(nickname)
	if !exists {
		return nil, false
	}

	op := &muc.OccupantPresenceInfo{
		Nickname: occupant.Nickname,
		RealJid:  occupant.RealJid,
		AffiliationRole: &muc.OccupantAffiliationRole{
			Affiliation: occupant.Affiliation,
			Role:        occupant.Role,
		},
	}

	return op, true
}

func (m *mucManager) newOccupantPresenceUpdateData(room *muc.Room, newOccupantInfo *muc.OccupantPresenceInfo) *occupantPresenceUpdateData {
	currentOccupantInfo, _ := m.occupantPresenceCurrentInfo(room, newOccupantInfo.Nickname)

	return &occupantPresenceUpdateData{
		room,
		currentOccupantInfo,
		newOccupantInfo,
		newOccupantInfo.GetActorInformation(room),
	}
}

func (od *occupantPresenceUpdateData) previousAffiliation() data.Affiliation {
	return od.currentOccupantInfo.AffiliationRole.Affiliation
}

func (od *occupantPresenceUpdateData) newAffiliation() data.Affiliation {
	return od.newOccupantInfo.AffiliationRole.Affiliation
}

func (od *occupantPresenceUpdateData) previousRole() data.Role {
	return od.currentOccupantInfo.AffiliationRole.Role
}

func (od *occupantPresenceUpdateData) newRole() data.Role {
	return od.newOccupantInfo.AffiliationRole.Role
}

func (od *occupantPresenceUpdateData) isOwnPresence() bool {
	return od.nickname() == od.room.SelfOccupantNickname()
}

func (od *occupantPresenceUpdateData) nickname() string {
	return od.newOccupantInfo.Nickname
}

func (od *occupantPresenceUpdateData) reason() string {
	return od.newOccupantInfo.AffiliationRole.Reason
}

func (m *mucManager) handleOccupantAffiliationRoleUpdate(occupantUpdateInfo *occupantPresenceUpdateData) {
	prevAffiliation := occupantUpdateInfo.previousAffiliation()
	prevRole := occupantUpdateInfo.previousRole()

	newAffiliation := occupantUpdateInfo.newAffiliation()
	newRole := occupantUpdateInfo.newRole()

	switch {
	case prevAffiliation.IsDifferentFrom(newAffiliation) && prevRole.IsDifferentFrom(newRole):
		m.handleOccupantAffiliationRoleUpdated(occupantUpdateInfo)

	case prevAffiliation.IsDifferentFrom(newAffiliation):
		m.handleOccupantAffiliationUpdated(occupantUpdateInfo)

	case prevRole.IsDifferentFrom(newRole):
		m.handleOccupantRoleUpdated(occupantUpdateInfo)
	}
}

func (m *mucManager) handleOccupantAffiliationRoleUpdated(occupantUpdateInfo *occupantPresenceUpdateData) {
	affiliationRoleUpate := data.AffiliationRoleUpdate{
		Nickname:            occupantUpdateInfo.nickname(),
		Reason:              occupantUpdateInfo.reason(),
		NewAffiliation:      occupantUpdateInfo.newAffiliation(),
		PreviousAffiliation: occupantUpdateInfo.previousAffiliation(),
		NewRole:             occupantUpdateInfo.newRole(),
		PreviousRole:        occupantUpdateInfo.previousRole(),
		Actor:               occupantUpdateInfo.actorOccupant,
	}

	if occupantUpdateInfo.isOwnPresence() {
		selfAffiliationRoleUpdate := data.SelfAffiliationRoleUpdate{}
		selfAffiliationRoleUpdate.AffiliationRoleUpdate = affiliationRoleUpate
		m.selfOccupantAffiliationRoleUpdated(occupantUpdateInfo.room.ID, selfAffiliationRoleUpdate)
		return
	}

	m.occupantAffiliationRoleUpdated(occupantUpdateInfo.room.ID, affiliationRoleUpate)
}

func (m *mucManager) handleOccupantAffiliationUpdated(occupantUpdateInfo *occupantPresenceUpdateData) {
	affiliationUpate := data.AffiliationUpdate{
		Nickname: occupantUpdateInfo.nickname(),
		Reason:   occupantUpdateInfo.reason(),
		New:      occupantUpdateInfo.newAffiliation(),
		Previous: occupantUpdateInfo.previousAffiliation(),
		Actor:    occupantUpdateInfo.actorOccupant,
	}

	if occupantUpdateInfo.isOwnPresence() {
		selfAffiliationUpdate := data.SelfAffiliationUpdate{}
		selfAffiliationUpdate.AffiliationUpdate = affiliationUpate
		m.selfOccupantAffiliationUpdated(occupantUpdateInfo.room.ID, selfAffiliationUpdate)
		return
	}

	m.occupantAffiliationUpdated(occupantUpdateInfo.room.ID, affiliationUpate)
}

func (m *mucManager) handleOccupantRoleUpdated(occupantUpdateInfo *occupantPresenceUpdateData) {
	roleUpdate := data.RoleUpdate{
		Nickname: occupantUpdateInfo.nickname(),
		Reason:   occupantUpdateInfo.reason(),
		New:      occupantUpdateInfo.newRole(),
		Previous: occupantUpdateInfo.previousRole(),
		Actor:    occupantUpdateInfo.actorOccupant,
	}

	if occupantUpdateInfo.isOwnPresence() {
		selfRoleUpdate := data.SelfRoleUpdate{}
		selfRoleUpdate.RoleUpdate = roleUpdate
		m.selfOccupantRoleUpdated(occupantUpdateInfo.room.ID, selfRoleUpdate)
		return
	}

	m.occupantRoleUpdated(occupantUpdateInfo.room.ID, roleUpdate)
}

func (m *mucManager) handleOccupantLeft(roomID jid.Bare, op *muc.OccupantPresenceInfo) {
	l := m.log.WithFields(log.Fields{
		"room":     roomID,
		"occupant": op.Nickname,
		"method":   "handleOccupantLeft",
	})

	r, ok := m.roomManager.GetRoom(roomID)
	if !ok {
		l.Error("Trying to get a room that is not in the room manager")
		return
	}

	err := r.Roster().RemoveOccupant(op.Nickname)
	if err != nil {
		l.WithError(err).Error("An error occurred trying to remove the occupant from the roster")
		return
	}

	m.occupantLeft(roomID, op)
}

func (m *mucManager) handleOccupantKick(roomID jid.Bare, op *muc.OccupantPresenceInfo) {
	l := m.log.WithFields(log.Fields{
		"room":     roomID,
		"occupant": op.Nickname,
		"method":   "handleOccupantLeft",
	})

	r, ok := m.roomManager.GetRoom(roomID)
	if !ok {
		l.Debug("Trying to get a room that is not in the room manager")
		return
	}

	occupantKicked := m.newOccupantPresenceUpdateData(r, op)

	roleUpdate := data.RoleUpdate{
		Nickname: occupantKicked.nickname(),
		Reason:   occupantKicked.reason(),
		Actor:    occupantKicked.actorOccupant,
		New:      occupantKicked.newRole(),
		Previous: occupantKicked.previousRole(),
	}

	err := r.Roster().RemoveOccupant(op.Nickname)
	if err != nil {
		l.WithError(err).Error("An error occurred trying to remove the occupant from the roster")
		return
	}

	if occupantKicked.isOwnPresence() {
		selfRoleUpdate := data.SelfRoleUpdate{}
		selfRoleUpdate.RoleUpdate = roleUpdate
		m.selfOccupantKicked(roomID, selfRoleUpdate)
		m.deleteRoomFromManager(roomID)
		return
	}

	m.occupantKicked(roomID, roleUpdate)
}

func (m *mucManager) handleOccupantBanned(roomID jid.Bare, op *muc.OccupantPresenceInfo) {
	l := m.log.WithFields(log.Fields{
		"room":     roomID,
		"occupant": op.Nickname,
		"method":   "handleOccupantBanned",
	})

	r, ok := m.roomManager.GetRoom(roomID)
	if !ok {
		l.Debug("Trying to get a room that is not in the room manager")
		return
	}

	occupantBanned := m.newOccupantPresenceUpdateData(r, op)
	err := r.Roster().RemoveOccupant(op.Nickname)
	if err != nil {
		l.WithError(err).Error("An error occurred trying to remove the occupant from the roster")
		return
	}

	m.handleOccupantAffiliationUpdated(occupantBanned)
	if occupantBanned.isOwnPresence() {
		m.deleteRoomFromManager(roomID)
	}
}

func (m *mucManager) handleOccupantUnavailable(roomID jid.Bare, op *muc.OccupantPresenceInfo, u *xmppData.MUCUser) {
	if u == nil || u.Destroy == nil {
		return
	}

	m.handleRoomDestroyed(roomID, u.Destroy)
}

func (m *mucManager) handleRoomDestroyed(roomID jid.Bare, d *xmppData.MUCRoomDestroy) {
	j, ok := jid.TryParseBare(d.Jid)
	if d.Jid != "" && !ok {
		m.log.WithFields(log.Fields{
			"room":            roomID,
			"alternativeRoom": d.Jid,
			"method":          "handleRoomDestroyed",
		}).Warn("Invalid alternative room ID")
	}

	m.roomDestroyed(roomID, d.Reason, j, d.Password)
}

func (m *mucManager) handleNonMembersRemoved(roomID jid.Bare, op *muc.OccupantPresenceInfo) {
	l := m.log.WithFields(log.Fields{
		"room":     roomID,
		"occupant": op.Nickname,
		"method":   "handleNonMembersRemoved",
	})

	r, ok := m.roomManager.GetRoom(roomID)
	if !ok {
		l.Error("Trying to get a room that is not in the room manager")
		return
	}

	err := r.Roster().RemoveOccupant(op.Nickname)
	if err != nil {
		l.WithError(err).Error("An error occurred trying to remove the occupant from the roster")
	}

	if r.SelfOccupant().Nickname == op.Nickname {
		m.removeSelfOccupant(roomID)
		m.deleteRoomFromManager(roomID)
		return
	}
	m.occupantRemoved(roomID, op.Nickname)
}

func (m *mucManager) handleOccupantRemovedOnAffiliationChange(roomID jid.Bare, op *muc.OccupantPresenceInfo) {
	l := m.log.WithFields(log.Fields{
		"room":     roomID,
		"occupant": op.Nickname,
		"method":   "handleOccupantRemovedOnAffiliationChange",
	})

	r, ok := m.roomManager.GetRoom(roomID)
	if !ok {
		l.Error("Trying to get a room that is not in the room manager")
		return
	}

	err := r.Roster().RemoveOccupant(op.Nickname)
	if err != nil {
		l.WithError(err).Error("An error occurred trying to remove the occupant from the roster")
	}

	if r.SelfOccupant().Nickname == op.Nickname {
		m.selfOccupantRemovedOnAffiliationChange(roomID)
		m.deleteRoomFromManager(roomID)
		return
	}
	m.occupantRemovedOnAffiliationChange(roomID, op.Nickname)
}

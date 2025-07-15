package session

import (
	"time"

	"github.com/chadsec1/decoyim/session/muc/data"
	xmppData "github.com/chadsec1/decoyim/xmpp/data"
	"github.com/chadsec1/decoyim/xmpp/jid"
	log "github.com/sirupsen/logrus"
)

func (m *mucManager) receiveClientMessage(stanza *xmppData.ClientMessage) {
	m.log.WithField("stanza", stanza).Debug("handleMUCReceivedClientMessage()")

	// See https://xmpp.org/extensions/xep-0045.html#order
	switch {
	case isRoomSubject(stanza):
		m.handleSubjectReceived(stanza)
	case isDelayedMessage(stanza):
		m.handleMessageReceived(stanza, m.appendHistoryMessage)
	case isLiveMessage(stanza):
		m.handleMessageReceived(stanza, m.liveMessageReceived)
	case isRoomConfigUpdate(stanza):
		m.handleRoomConfigUpdate(stanza)
	}
}

func (m *mucManager) appendHistoryMessage(roomID jid.Bare, nickname, message string, timestamp time.Time) {
	room, ok := m.roomManager.GetRoom(roomID)
	if ok {
		room.AddHistoryMessage(nickname, message, timestamp)
	}
}

// The discussion history MUST happen only one time in the events flow of XMPP's MUC
// This should be done in a proper way, maybe in the pending "state machine" pattern
// that we want to implement later, when that happens, this method should be fine
func (m *mucManager) handleDiscussionHistory(stanza *xmppData.ClientMessage) {
	roomID := m.retrieveRoomID(stanza.From, "handleDiscussionHistory")
	room, ok := m.roomManager.GetRoom(roomID)
	if ok {
		m.discussionHistoryReceived(roomID, room.GetDiscussionHistory())
	}
}

func (m *mucManager) handleSubjectReceived(stanza *xmppData.ClientMessage) {
	l := m.log.WithFields(log.Fields{
		"from": stanza.From,
		"who":  "handleSubjectReceived",
	})

	roomID := m.retrieveRoomID(stanza.From, "handleSubjectReceived")
	room, ok := m.roomManager.GetRoom(roomID)
	if !ok {
		l.WithField("room", roomID).Error("Error trying to read the subject of room")
		return
	}

	s := getSubjectFromStanza(stanza)
	updated := room.UpdateSubject(s)
	if updated {
		m.subjectUpdated(roomID, getNicknameFromStanza(stanza), s)
		return
	}

	m.handleDiscussionHistory(stanza)
	m.subjectReceived(roomID, s)
	m.joinRoomFinished(roomID)
}

func (m *mucManager) handleMessageReceived(stanza *xmppData.ClientMessage, h func(jid.Bare, string, string, time.Time)) {
	roomID, nickname := m.retrieveRoomIDAndNickname(stanza.From)
	h(roomID, nickname, stanza.Body, retrieveMessageTime(stanza))
}

func (m *mucManager) handleMUCUserMessage(stanza *xmppData.ClientMessage) {
	roomID := m.retrieveRoomID(stanza.From, "handleMUCUserMessage")
	m.accountAffiliationUpdated(roomID, jid.Parse(stanza.MUCUser.Item.Jid), affiliationFromMUCUserItem(stanza.MUCUser.Item))
}

func affiliationFromMUCUserItem(item *xmppData.MUCUserItem) data.Affiliation {
	affiliation := data.AffiliationNone
	if item != nil && item.Affiliation != "" {
		affiliation = item.Affiliation
	}
	return affiliationFromString(affiliation)
}

func bodyHasContent(stanza *xmppData.ClientMessage) bool {
	return stanza.Body != ""
}

func isDelayedMessage(stanza *xmppData.ClientMessage) bool {
	return stanza.Delay != nil
}

func isLiveMessage(stanza *xmppData.ClientMessage) bool {
	return bodyHasContent(stanza) && !isDelayedMessage(stanza)
}

func isRoomSubject(stanza *xmppData.ClientMessage) bool {
	return stanza.Subject != nil && stanza.Body == ""
}

func hasMUCUserExtension(stanza *xmppData.ClientMessage) bool {
	return stanza.MUCUser != nil
}

func getNicknameFromStanza(stanza *xmppData.ClientMessage) string {
	from, ok := jid.TryParseFull(stanza.From)
	if ok {
		return from.Resource().String()
	}

	return ""
}

func getSubjectFromStanza(stanza *xmppData.ClientMessage) string {
	if isRoomSubject(stanza) {
		return stanza.Subject.Text
	}

	return ""
}

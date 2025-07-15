package session

import (
	"github.com/chadsec1/decoyim/session/events"
	"github.com/chadsec1/decoyim/xmpp/data"
	xmppData "github.com/chadsec1/decoyim/xmpp/data"
	"github.com/chadsec1/decoyim/xmpp/jid"
)

func (m *mucManager) publishMUCError(roomID jid.Bare, nickname string, e *data.StanzaError) {
	ev := events.MUCError{}
	ev.ErrorType = getEventErrorTypeBasedOnStanzaError(e)
	ev.Room = roomID
	ev.Nickname = nickname

	m.publishEvent(ev)
}

func (m *mucManager) publishMUCMessageError(roomID jid.Bare, e *data.StanzaError) {
	ev := events.MUCError{}
	ev.ErrorType = getEventErrorTypeBasedOnMessageError(e)
	ev.Room = roomID

	m.publishEvent(ev)
}

func isMUCErrorPresence(e *xmppData.StanzaError) bool {
	return e != nil && (e.MUCNotAuthorized != nil ||
		e.MUCForbidden != nil ||
		e.MUCItemNotFound != nil ||
		e.MUCNotAllowed != nil ||
		e.MUCNotAcceptable != nil ||
		e.MUCRegistrationRequired != nil ||
		e.MUCConflict != nil ||
		e.MUCServiceUnavailable != nil)
}

func getEventErrorTypeBasedOnMessageError(e *data.StanzaError) events.MUCErrorType {
	t := events.MUCNoError
	switch {
	case e.MUCForbidden != nil:
		t = events.MUCMessageForbidden
	case e.MUCNotAcceptable != nil:
		t = events.MUCMessageNotAcceptable
	}
	return t
}

func getEventErrorTypeBasedOnStanzaError(e *data.StanzaError) events.MUCErrorType {
	t := events.MUCNoError
	switch {
	case e.MUCNotAuthorized != nil:
		t = events.MUCNotAuthorized
	case e.MUCForbidden != nil:
		t = events.MUCForbidden
	case e.MUCItemNotFound != nil:
		t = events.MUCItemNotFound
	case e.MUCNotAllowed != nil:
		t = events.MUCNotAllowed
	case e.MUCNotAcceptable != nil:
		t = events.MUCNotAcceptable
	case e.MUCRegistrationRequired != nil:
		t = events.MUCRegistrationRequired
	case e.MUCConflict != nil:
		t = events.MUCConflict
	case e.MUCServiceUnavailable != nil:
		t = events.MUCServiceUnavailable
	}
	return t
}

func isMUCError(e *data.StanzaError) bool {
	return getEventErrorTypeBasedOnStanzaError(e) != events.MUCNoError
}

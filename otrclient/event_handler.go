package otrclient

import (
	log "github.com/sirupsen/logrus"

	"github.com/chadsec1/decoyim/coylog"
	"github.com/chadsec1/decoyim/i18n"
	"github.com/chadsec1/decoyim/xmpp/jid"
	"github.com/coyim/otr3"
)

var (
	// ErrorPrefix can be used to make an OTR error by appending an error message
	// to it.
	ErrorPrefix = "?OTR Error:"
)

// EventHandler is used to contain information pertaining to the events of a specific OTR interaction
type EventHandler struct {
	SmpQuestion        string
	securityChange     SecurityChange
	WaitingForSecret   bool
	account            string
	peer               jid.Any
	notifications      chan<- string
	delayedMessageSent chan<- int
	delays             map[int]bool
	pendingDelays      int
	Log                coylog.Logger
}

// ConsumeDelayedState returns whether the given trace has been delayed or not, blanking out that status as a side effect
func (e *EventHandler) ConsumeDelayedState(trace int) bool {
	val, ok := e.delays[trace]
	delete(e.delays, trace)
	return ok && val
}

func (e *EventHandler) notify(s string) {
	e.notifications <- s
}

// HandleErrorMessage is called when asked to handle a specific error message
func (e *EventHandler) HandleErrorMessage(error otr3.ErrorCode) []byte {
	e.Log.WithField("error", error).Debug("HandleErrorMessage()")

	switch error {
	case otr3.ErrorCodeEncryptionError:
		return []byte("Error occurred encrypting message.")
	case otr3.ErrorCodeMessageUnreadable:
		return []byte("You transmitted an unreadable encrypted message.")
	case otr3.ErrorCodeMessageMalformed:
		return []byte("You transmitted a malformed data message.")
	case otr3.ErrorCodeMessageNotInPrivate:
		return []byte("You sent encrypted data to a peer, who wasn't expecting it.")
	}

	return nil
}

// HandleSecurityEvent is called to handle a specific security event
func (e *EventHandler) HandleSecurityEvent(event otr3.SecurityEvent) {
	e.Log.WithFields(log.Fields{
		"event": event,
	}).Debug("HandleSecurityEvent()")
	switch event {
	case otr3.GoneSecure:
		e.pendingDelays = 0
		e.securityChange = NewKeys
	case otr3.StillSecure:
		e.securityChange = RenewedKeys
	case otr3.GoneInsecure:
		e.securityChange = ConversationEnded
	}
}

// HandleSMPEvent is called to handle a specific SMP event
func (e *EventHandler) HandleSMPEvent(event otr3.SMPEvent, progressPercent int, question string) {
	e.Log.WithFields(log.Fields{
		"event":    event,
		"progress": progressPercent,
		"question": question,
	}).Debug("HandleSMPEvent()")

	switch event {
	case otr3.SMPEventAskForSecret, otr3.SMPEventAskForAnswer:
		e.securityChange = SMPSecretNeeded
		e.SmpQuestion = question
		e.WaitingForSecret = true
	case otr3.SMPEventSuccess:
		if progressPercent == 100 {
			e.securityChange = SMPComplete
		}
	case otr3.SMPEventAbort, otr3.SMPEventFailure, otr3.SMPEventCheated:
		e.securityChange = SMPFailed
	}
}

// HandleMessageEvent is called to handle a specific message event
func (e *EventHandler) HandleMessageEvent(event otr3.MessageEvent, message []byte, err error, trace ...interface{}) {
	switch event {
	case otr3.MessageEventLogHeartbeatReceived:
		e.Log.WithFields(log.Fields{
			"from": e.peer,
		}).Debug("Heartbeat received")
	case otr3.MessageEventLogHeartbeatSent:
		e.Log.WithFields(log.Fields{
			"to": e.peer,
		}).Debug("Heartbeat sent")
	case otr3.MessageEventReceivedMessageUnrecognized:
		e.Log.WithFields(log.Fields{
			"from": e.peer,
		}).Debug("Unrecognized OTR message received")
	case otr3.MessageEventEncryptionRequired:
		e.delays[trace[0].(int)] = true
		e.pendingDelays++
		if e.pendingDelays == 1 {
			e.notify(i18n.Local("Attempting to start a private conversation..."))
		}
	case otr3.MessageEventEncryptionError:
		// This happens when something goes wrong putting together a new data packet in OTR
		e.notify(i18n.Local("An error occurred when encrypting your message. The message was not sent."))
	case otr3.MessageEventConnectionEnded:
		// This happens when we have finished a conversation and tries to send a message afterwards
		e.notify(i18n.Local("Your message was not sent, since the other person has already closed their private connection to you."))
	case otr3.MessageEventMessageReflected:
		e.notify(i18n.Local("We are receiving our own OTR messages. You are either trying to talk to yourself, or someone is reflecting your messages back at you."))
	case otr3.MessageEventSetupError:
		e.notify(i18n.Local("Error setting up private conversation."))
		if err != nil {
			e.Log.WithError(err).WithField("with", e.peer).
				Warn("Error setting up private conversation")
		}
	case otr3.MessageEventMessageSent:
		if len(trace) > 0 {
			e.delayedMessageSent <- trace[0].(int)
		}
	case otr3.MessageEventMessageResent:
		e.notify(i18n.Local("The last message to the other person was resent, since we couldn't deliver the message previously."))
	case otr3.MessageEventReceivedMessageUnreadable:
		// This happens when the authenticator is wrong, message counters are out of whack
		// or several other things that indicate tampering or attack
		e.notify(i18n.Local("We received an unreadable encrypted message. It has probably been tampered with, or was sent from an older client."))
	case otr3.MessageEventReceivedMessageMalformed:
		// This happens when the OTR header is malformed, or different deserialization issues
		e.notify(i18n.Local("We received a malformed data message."))
	case otr3.MessageEventReceivedMessageGeneralError:
		// This happens when we receive an error from the other party
		e.notify(i18n.Localf("We received this error from the other person: %s.", string(message)))
	case otr3.MessageEventReceivedMessageNotInPrivate:
		// This happens when we receive what looks like a data message, but we're not in encrypted state
		// TODO: this should open conversation window
		e.notify(i18n.Local("We received an encrypted message which can't be read, since private communication is not currently turned on. You should ask your peer to repeat what they said."))
	case otr3.MessageEventReceivedMessageUnencrypted:
		// This happens when we receive a non-OTR message, even though we have require encryption turned on
		e.notify(i18n.Local("We received a message that was transferred without encryption"))
	case otr3.MessageEventReceivedMessageForOtherInstance:
		// We ignore this message on purpose, for now it would be too noisy to notify about it
	default:
		e.Log.WithError(err).WithFields(log.Fields{
			"from":    e.peer,
			"event":   event,
			"message": message,
		}).Warn("Unhandled OTR3 Message Event")
	}
}

// ConsumeSecurityChange is called to get the current security change and forget the old one
func (e *EventHandler) ConsumeSecurityChange() SecurityChange {
	ret := e.securityChange
	e.securityChange = NoChange
	return ret
}

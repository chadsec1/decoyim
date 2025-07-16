package gui

import (
	"sync"
	"time"

	"github.com/chadsec1/decoyim/decoylog"
	"github.com/chadsec1/decoyim/session/events"
	"github.com/chadsec1/decoyim/session/muc/data"
	"github.com/chadsec1/decoyim/xmpp/jid"
)

func (v *roomView) handleRoomEvent(ev events.MUC) {
	switch t := ev.(type) {
	case events.MUCSelfOccupantJoined:
		v.publishSelfOccupantJoinedEvent(selfOccupantJoinedEvent{t.Nickname, t.Role})
	case events.MUCOccupantUpdated:
		v.publishEvent(occupantUpdatedEvent{
			nickname: t.Nickname,
			role:     t.Role,
		})
	case events.MUCOccupantJoined:
		v.handleOccupantJoinedEvent(t.Nickname)
	case events.MUCOccupantLeft:
		v.publishEvent(occupantLeftEvent{t.Nickname})
	case events.MUCLiveMessageReceived:
		v.publishMessageEvent("live", t.Nickname, t.Message, t.Timestamp)
	case events.MUCDelayedMessageReceived:
		v.publishMessageEvent("delayed", t.Nickname, t.Message, t.Timestamp)
	case events.MUCDiscussionHistoryReceived:
		v.publishEvent(discussionHistoryEvent{history: t.History})
	case events.MUCSubjectUpdated:
		v.publishSubjectUpdatedEvent(t.Nickname, t.Subject)
	case events.MUCSubjectReceived:
		v.publishSubjectReceivedEvent(t.Subject)
	case events.MUCJoinRoomFinished:
		v.publishEvent(joinRoomFinishedEvent{
			isReconnecting: v.isReconnecting,
		})
	case events.MUCLoggingEnabled:
		v.publishEvent(loggingEnabledEvent{})
	case events.MUCLoggingDisabled:
		v.publishEvent(loggingDisabledEvent{})
	case events.MUCRoomAnonymityChanged:
		v.publishEvent(roomAnonymityEvent{
			anonymityLevel: t.AnonymityLevel,
		})
	case events.MUCRoomDiscoInfoReceived:
		v.handleDiscoInfoReceived(t.DiscoInfo)
	case events.MUCRoomConfigTimeout:
		v.handleDiscoInfoTimeout()
	case events.MUCRoomConfigChanged:
		v.publishEvent(roomConfigChangedEvent{
			changes:   roomConfigChangedTypes(t.Changes),
			discoInfo: t.DiscoInfo,
		})
	case events.MUCOccupantRemoved:
		v.publishEvent(occupantRemovedEvent{
			nickname: t.Nickname,
		})
	case events.MUCSelfOccupantRemoved:
		v.publishEvent(selfOccupantRemovedEvent{})
	case events.MUCRoomDestroyed:
		v.publishEvent(roomDestroyedEvent{
			reason:      t.Reason,
			alternative: t.AlternativeRoom,
			password:    t.Password,
		})
	case events.MUCSelfOccupantAffiliationRoleUpdated:
		v.publishSelfOccupantAffiliationRoleUpdatedEvent(t.AffiliationRoleUpdate)
	case events.MUCOccupantAffiliationRoleUpdated:
		v.publishOccupantAffiliationRoleUpdatedEvent(t.AffiliationRoleUpdate)
	case events.MUCOccupantAffiliationUpdated:
		v.publishOccupantAffiliationUpdatedEvent(t.AffiliationUpdate)
	case events.MUCSelfOccupantAffiliationUpdated:
		v.publishSelfOccupantAffiliationUpdatedEvent(t.AffiliationUpdate)
	case events.MUCOccupantRoleUpdated:
		v.publishOccupantRoleUpdatedEvent(t.RoleUpdate)
	case events.MUCSelfOccupantRoleUpdated:
		v.publishSelfOccupantRoleUpdatedEvent(t.RoleUpdate)
	case events.MUCOccupantKicked:
		v.publishOccupantRoleUpdatedEvent(t.RoleUpdate)
	case events.MUCSelfOccupantKicked:
		v.publishSelfOccupantRoleUpdatedEvent(t.RoleUpdate)
	case events.MUCSelfOccupantConnected:
		v.publishSelfOccupantConnectedEvent()
	case events.MUCSelfOccupantDisconnected:
		v.publishSelfOccupantDisconnectedEvent()
	case events.MUCSelfOccupantConnecting:
		v.publishSelfOccupantConnectingEvent()
	case events.MUCAccountAffiliationUpdated:
		v.publishAccountAffiliationUpdated(t.AccountAddress, t.Affiliation)
	case events.MUCOccupantRemovedOnAffiliationChange:
		v.publishOccupantRemovedOnAffiliationChangeEvent(t.Nickname)
	case events.MUCSelfOccupantRemovedOnAffiliationChange:
		v.publishSelfOccupantRemovedOnAffiliationChangeEvent()
	default:
		v.log.WithField("event", t).Warn("Unsupported room event received")
	}
}

type roomViewObserver struct {
	id       string
	onNotify func(roomViewEvent)
}

type roomViewSubscribers struct {
	observers     []*roomViewObserver
	observersLock sync.Mutex
	log           decoylog.Logger
}

func (v *roomView) newRoomViewSubscribers() *roomViewSubscribers {
	return &roomViewSubscribers{
		log: v.log.WithField("where", "roomViewSubscribers"),
	}
}

func (s *roomViewSubscribers) subscribe(id string, onNotify func(roomViewEvent)) {
	s.observersLock.Lock()
	defer s.observersLock.Unlock()

	s.observers = append(s.observers, &roomViewObserver{
		id:       id,
		onNotify: onNotify,
	})
}

func removeFromObservers(obs []*roomViewObserver, id string) []*roomViewObserver {
	for i, o := range obs {
		if o.id == id {
			return append(obs[:i], obs[i+1:]...)
		}
	}
	return obs
}

func (s *roomViewSubscribers) unsubscribe(id string) {
	s.observersLock.Lock()
	defer s.observersLock.Unlock()

	s.observers = removeFromObservers(s.observers, id)
}

func (s *roomViewSubscribers) publish(ev roomViewEvent) {
	s.observersLock.Lock()
	observers := s.observers
	s.observersLock.Unlock()

	for _, o := range observers {
		o.onNotify(ev)
	}
}

func (v *roomView) subscribe(id string, onNotify func(roomViewEvent)) {
	v.subscribers.subscribe(id, onNotify)
}

func (v *roomView) unsubscribe(id string) {
	v.subscribers.unsubscribe(id)
}

// publishEvent MUST NOT be called from the UI thread
func (v *roomView) publishEvent(ev roomViewEvent) {
	if v.opened {
		// We should analyze if at some point we need
		// to be able to publish an event in a closed view
		v.subscribers.publish(ev)
	}
}

// publishSelfOccupantJoinedEvent MUST NOT be called from the UI thread
func (v *roomView) publishSelfOccupantJoinedEvent(ev selfOccupantJoinedEvent) {
	isReconnecting := v.isReconnecting
	enteredAtLeastOnce := v.enteredAtLeastOnce

	if isReconnecting && enteredAtLeastOnce {
		v.publishEvent(selfOccupantReconnectedEvent{ev})
	} else {
		v.publishEvent(ev)
	}
}

// publishMessageEvent MUST NOT be called from the UI thread
func (v *roomView) publishMessageEvent(tp, nickname, message string, timestamp time.Time) {
	v.publishEvent(messageEvent{
		tp:        tp,
		nickname:  nickname,
		message:   message,
		timestamp: timestamp,
	})
}

// publishSubjectUpdatedEvent MUST NOT be called from the UI thread
func (v *roomView) publishSubjectUpdatedEvent(nickname, subject string) {
	v.publishEvent(subjectUpdatedEvent{
		nickname: nickname,
		subject:  subject,
	})
}

// publishSubjectReceivedEvent MUST NOT be called from the UI thread
func (v *roomView) publishSubjectReceivedEvent(subject string) {
	v.publishEvent(subjectReceivedEvent{
		subject: subject,
	})
}

// publishSelfOccupantConnectedEvent MUST NOT be called from the UI thread
func (v *roomView) publishSelfOccupantConnectedEvent() {
	v.publishEvent(selfOccupantConnectedEvent{})
}

// publishSelfOccupantRoleUpdatedEvent MUST NOT be called from the UI thread
func (v *roomView) publishSelfOccupantDisconnectedEvent() {
	v.publishEvent(selfOccupantDisconnectedEvent{})
}

// publishSelfOccupantConnectingEvent MUST NOT be called from the UI thread
func (v *roomView) publishSelfOccupantConnectingEvent() {
	v.publishEvent(selfOccupantConnectingEvent{})
}

// publishAccountAffiliationUpdated MUST NOT be called from the UI thread
func (v *roomView) publishAccountAffiliationUpdated(accountAddress jid.Any, affiliation data.Affiliation) {
	v.publishEvent(accountAffiliationUpdated{accountAddress, affiliation})
}

// publishAccountAffiliationUpdated MUST NOT be called from the UI thread
func (v *roomView) publishOccupantRemovedOnAffiliationChangeEvent(nickname string) {
	v.publishEvent(occupantRemovedOnAffiliationChangeEvent{nickname})
}

// publishAccountAffiliationUpdated MUST NOT be called from the UI thread
func (v *roomView) publishSelfOccupantRemovedOnAffiliationChangeEvent() {
	v.publishEvent(selfOccupantRemovedOnAffiliationChangeEvent{})
}

func (v *roomView) handleOccupantJoinedEvent(nickname string) {
	reconnecting := v.isReconnecting
	if !reconnecting {
		v.publishEvent(occupantJoinedEvent{
			nickname: nickname,
		})
	}
}

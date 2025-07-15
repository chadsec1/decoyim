package gui

import (
	"fmt"
	"sync"

	"github.com/chadsec1/decoyim/i18n"
	"github.com/chadsec1/decoyim/session"
	"github.com/chadsec1/decoyim/xmpp/jid"
	"github.com/coyim/gotk3adapter/gtki"
)

var supportedCreateMUCErrors map[error]string

// We should put here all MUC-related errors that we want to support
// and have a custom and useful user message for each one
func initMUCSupportedErrors() {
	supportedCreateMUCErrors = map[error]string{
		session.ErrInvalidInformationQueryRequest:      i18n.Local("Something went wrong trying to communicate with the server."),
		session.ErrUnexpectedResponse:                  i18n.Local("The connection to the server can't be established."),
		session.ErrInformationQueryResponse:            i18n.Local("You don't have permission to create a room."),
		session.ErrInformationQueryResponseWithGoneTag: i18n.Local("Could not create the room with the name provided. Please, change it and try again."),
	}
}

func newNicknameConflictError(n jid.Resource) error {
	return fmt.Errorf("the nickname \"%s\" is already being used", n)
}

func newRegistrationRequiredError(roomID jid.Bare) error {
	return fmt.Errorf("the room \"%s\" only allows registered members", roomID)
}

type callbacksSet struct {
	callbacks []func()
	sync.Mutex
}

func newCallbacksSet(callbacks ...func()) *callbacksSet {
	return &callbacksSet{
		callbacks: callbacks,
	}
}

func (s *callbacksSet) add(callbacks ...func()) {
	s.Lock()
	defer s.Unlock()
	s.callbacks = append(s.callbacks, callbacks...)
}

func (s *callbacksSet) invokeAll() {
	s.Lock()
	defer s.Unlock()

	for _, cb := range s.callbacks {
		cb()
	}
}

func doOnlyOnceAtATime(f func(func())) func() {
	isDoing := false
	return func() {
		if isDoing {
			return
		}
		isDoing = true
		// The "done" function should be called ONLY from the UI thread,
		// in other cases it's not "safe" executing it.
		f(func() {
			isDoing = false
		})
	}
}

func getEntryText(e gtki.Entry) string {
	t, err := e.GetText()
	if err != nil {
		panic(err)
	}

	return t
}

func setTextViewText(tv gtki.TextView, t string) {
	b, err := tv.GetBuffer()
	if err != nil {
		panic(err)
	}

	b.SetText(t)
}

func getTextViewText(tv gtki.TextView) string {
	b, err := tv.GetBuffer()
	if err != nil {
		panic(err)
	}

	return b.GetText(b.GetStartIter(), b.GetEndIter(), false)
}

func setFieldVisibility(w gtki.Widget, v bool) {
	w.SetVisible(v)
}

func setFieldSensitive(w gtki.Widget, v bool) {
	w.SetSensitive(v)
}

func setFieldLabel(w gtki.Widget, l string) {
	w.SetProperty("label", l)
}

func disableField(w gtki.Widget) {
	setFieldSensitive(w, false)
}

func enableField(w gtki.Widget) {
	setFieldSensitive(w, true)
}

func setEntryText(e gtki.Entry, t string) {
	e.SetText(t)
}

func setSwitchActive(s gtki.Switch, v bool) {
	s.SetActive(v)
}

func getSwitchActive(s gtki.Switch) bool {
	return s.GetActive()
}

func setLabelText(l gtki.Label, t string) {
	l.SetLabel(t)
}

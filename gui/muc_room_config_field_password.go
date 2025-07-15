package gui

import (
	"github.com/chadsec1/decoyim/i18n"
	"github.com/chadsec1/decoyim/session/muc"
	"github.com/coyim/gotk3adapter/gtki"
)

type roomConfigFormFieldPassword struct {
	*roomConfigFormField
	value *muc.RoomConfigFieldTextValue

	entry                   gtki.Entry  `gtk-widget:"password-entry"`
	confirmEntry            gtki.Entry  `gtk-widget:"password-confirmation-entry"`
	showPasswordButton      gtki.Button `gtk-widget:"password-show-button"`
	passwordMatchErrorLabel gtki.Label  `gtk-widget:"password-match-error"`
}

func newRoomConfigFormFieldPassword(ft muc.RoomConfigFieldType, fieldInfo roomConfigFieldTextInfo, value *muc.RoomConfigFieldTextValue, onShowValidationErrors func(), onHideValidationErrors func()) *roomConfigFormFieldPassword {
	field := &roomConfigFormFieldPassword{value: value}
	field.roomConfigFormField = newRoomConfigFormField(ft, fieldInfo, "MUCRoomConfigFormFieldPassword", onShowValidationErrors, onHideValidationErrors)

	field.initBuilder()
	field.initDefaults()

	return field
}

func (f *roomConfigFormFieldPassword) initBuilder() {
	panicOnDevError(f.builder.bindObjects(f))
	f.builder.ConnectSignals(map[string]interface{}{
		"on_show_password_clicked":   f.onShowPasswordClicked,
		"on_password_change":         f.onPasswordChange,
		"on_confirm_password_change": f.changeConfirmPasswordEntryStyle,
	})
}

func (f *roomConfigFormFieldPassword) initDefaults() {
	f.confirmEntry.SetSensitive(!f.entry.GetVisibility())
	mucStyles.setErrorLabelStyle(f.passwordMatchErrorLabel)
	mucStyles.setEntryErrorStyle(f.confirmEntry)

	if f.value.Text() != "" {
		setEntryText(f.entry, f.value.Text())
		setEntryText(f.confirmEntry, f.value.Text())
	}
}

func (f *roomConfigFormFieldPassword) passwordsMatch() bool {
	return getEntryText(f.entry) == getEntryText(f.confirmEntry)
}

func (f *roomConfigFormFieldPassword) currentPassword() string {
	return getEntryText(f.entry)
}

// updateFieldValue MUST be called from the UI thread
func (f *roomConfigFormFieldPassword) updateFieldValue() {
	f.value.SetText(f.currentPassword())
}

// isValid implements the hasRoomConfigFormField interface
func (f *roomConfigFormFieldPassword) isValid() bool {
	return f.passwordsMatch()
}

// showValidationErrors implements the hasRoomConfigFormField interface
func (f *roomConfigFormFieldPassword) showValidationErrors() {
	f.changeConfirmPasswordEntryStyle()
	f.onShowConfirmPasswordBasedOnMatchError()
	f.focusConfirm()
}

func (f *roomConfigFormFieldPassword) focusConfirm() {
	f.confirmEntry.GrabFocus()
}

// onShowConfirmPasswordBasedOnMatchError MUST be called from the UI thread
func (f *roomConfigFormFieldPassword) onShowConfirmPasswordBasedOnMatchError() {
	f.passwordMatchErrorLabel.SetVisible(!f.passwordsMatch())
}

// onPasswordChange MUST be called from the UI thread
func (f *roomConfigFormFieldPassword) onPasswordChange() {
	f.passwordMatchErrorLabel.SetVisible(false)
	if f.entry.GetVisibility() {
		f.confirmEntry.SetText(getEntryText(f.entry))
	}

	f.changeConfirmPasswordEntryStyle()
}

// changeConfirmPasswordEntryStyle MUST be called from the UI thread
func (f *roomConfigFormFieldPassword) changeConfirmPasswordEntryStyle() {
	f.passwordMatchErrorLabel.SetVisible(false)
	sc, _ := f.confirmEntry.GetStyleContext()
	if !f.passwordsMatch() {
		f.onShowValidationErrors()
		sc.AddClass("entry-error")
		return
	}

	sc.RemoveClass("entry-error")
	f.hideValidationErrors()
}

// onShowPasswordClicked MUST be called from the UI thread
func (f *roomConfigFormFieldPassword) onShowPasswordClicked() {
	visible := f.entry.GetVisibility()
	if !visible {
		f.confirmEntry.SetText(getEntryText(f.entry))
	}
	f.confirmEntry.SetVisibility(!visible)
	f.confirmEntry.SetSensitive(visible)
	f.entry.SetVisibility(!visible)
	f.updateShowPasswordLabel(!visible)
}

// updateShowPasswordLabel MUST be called from the UI thread
func (f *roomConfigFormFieldPassword) updateShowPasswordLabel(v bool) {
	if v {
		f.showPasswordButton.SetLabel(i18n.Local("Hide"))
		return
	}
	f.showPasswordButton.SetLabel(i18n.Local("Show"))
}

// focusWidget implements the hasRoomConfigFormField interface
func (f *roomConfigFormFieldPassword) focusWidget() focusable {
	return f.confirmEntry
}

package gui

import (
	"errors"

	"github.com/chadsec1/decoyim/i18n"
	"github.com/chadsec1/decoyim/xmpp/jid"
	"github.com/coyim/gotk3adapter/gtki"
)

var (
	errEmptyMemberIdentifier   = errors.New("empty member identifier (jid)")
	errInvalidMemberIdentifier = errors.New("invalid member identifier (jid)")
)

type mucRoomConfigListAddComponent struct {
	u *gtkUI

	dialog          gtki.Dialog `gtk-widget:"room-config-list-add-dialog"`
	titleLabel      gtki.Label  `gtk-widget:"room-config-list-add-title"`
	contentBox      gtki.Box    `gtk-widget:"room-config-list-add-body"`
	removeAllButton gtki.Button `gtk-widget:"room-config-list-remove-all-button"`
	applyButton     gtki.Button `gtk-widget:"room-config-list-add-apply"`
	notificationBox gtki.Box    `gtk-widget:"notification-box"`

	notifications *notificationsComponent
	dialogTitle   string
	formTitle     string
	form          *roomConfigListForm
	formItems     []*mucRoomConfigListFormItem
	onApply       func(jidList []string)
}

func newMUCRoomConfigListAddComponent(dialogTitle, formTitle string, onApply func(jidList []string), parent gtki.Window) *mucRoomConfigListAddComponent {
	la := &mucRoomConfigListAddComponent{
		dialogTitle: dialogTitle,
		formTitle:   formTitle,
		onApply:     onApply,
	}

	la.initBuilder()
	la.initNotifications()
	la.initAddOccupantForm()
	la.initDefaults(parent)

	return la
}

func (la *mucRoomConfigListAddComponent) initBuilder() {
	builder := newBuilder("MUCRoomConfigListAddDialog")
	panicOnDevError(builder.bindObjects(la))

	builder.ConnectSignals(map[string]interface{}{
		"on_cancel":     la.close,
		"on_remove_all": la.onRemoveAllClicked,
		"on_apply":      la.onApplyClicked,
	})
}

func (la *mucRoomConfigListAddComponent) initNotifications() {
	la.notifications = la.u.newNotificationsComponent()
	la.notificationBox.Add(la.notifications.contentBox())
}

func (la *mucRoomConfigListAddComponent) initDefaults(parent gtki.Window) {
	la.removeAllButton.SetSensitive(false)

	la.dialog.SetTitle(la.dialogTitle)
	la.dialog.SetTransientFor(parent)
	la.titleLabel.SetLabel(la.formTitle)
}

func (la *mucRoomConfigListAddComponent) initAddOccupantForm() {
	la.form = la.newAddOccupantForm()
	defaultFormItem := newMUCRoomConfigListFormItem(la.form, la.appendNewFormItem, nil)
	la.contentBox.PackStart(defaultFormItem.contentBox(), false, true, 0)
}

// newAddOccupantForm MUST be called from the UI thread
func (la *mucRoomConfigListAddComponent) newAddOccupantForm() *roomConfigListForm {
	return newRoomConfigListForm(
		la.refresh,
		la.onApplyClicked,
	)
}

// appendNewFormItem MUST be called from the UI thread
func (la *mucRoomConfigListAddComponent) appendNewFormItem(jid string) {
	if !la.jidCanBeAdded(jid) {
		return
	}

	onRemove := func(index int) {
		la.removeItemAndUpdateIndexes(index)
		la.form.focusJidEntry()
		la.refresh()
	}

	form := la.newAddOccupantForm()
	form.setJid(jid)

	item := newMUCRoomConfigListFormItem(form, nil, onRemove)
	item.updateIndex(len(la.formItems))
	la.prependItem(item)
	la.contentBox.PackEnd(item.contentBox(), false, true, 0)

	la.refresh()
}

// prependItem MUST NOT be called from the UI thread
func (la *mucRoomConfigListAddComponent) prependItem(item *mucRoomConfigListFormItem) {
	la.formItems = append([]*mucRoomConfigListFormItem{item}, la.formItems...)
	la.reindex()
}

// reindex MUST NOT be called from the UI thread
func (la *mucRoomConfigListAddComponent) reindex() {
	for index, itm := range la.formItems {
		itm.updateIndex(index)
	}
}

func (la *mucRoomConfigListAddComponent) jidCanBeAdded(jid string) bool {
	for _, l := range la.formItems {
		if l.form.jid() == jid {
			return false
		}
	}

	return true
}

// removeItemByIndex MUST be called from the UI thread
func (la *mucRoomConfigListAddComponent) removeItemAndUpdateIndexes(index int) {
	la.removeItemByIndex(index)
	la.updateItemIndexes()
}

func (la *mucRoomConfigListAddComponent) removeItemByIndex(index int) {
	if itm, ok := la.getItemToRemove(index); ok {
		la.contentBox.Remove(itm.contentBox())
		sl := append(la.formItems[:index], la.formItems[index+1:]...)
		la.formItems = sl
	}
}

func (la *mucRoomConfigListAddComponent) getItemToRemove(index int) (*mucRoomConfigListFormItem, bool) {
	for idx, itm := range la.formItems {
		if idx == index {
			return itm, true
		}
	}
	return nil, false
}

func (la *mucRoomConfigListAddComponent) updateItemIndexes() {
	for i, itm := range la.formItems {
		if i != itm.index {
			itm.updateIndex(itm.index - 1)
		}
	}
}

// forEachForm MUST be called from the UI thread
func (la *mucRoomConfigListAddComponent) forEachForm(fn func(*roomConfigListForm) bool) {
	for _, itm := range la.formItems {
		if ok := fn(itm.form); !ok {
			return
		}
	}
}

// areAllFormsFilled MUST be called from the UI thread
func (la *mucRoomConfigListAddComponent) areAllFormsFilled() bool {
	formsAreFilled := la.form.isFilled()

	if la.hasItems() {
		la.forEachForm(func(form *roomConfigListForm) bool {
			formsAreFilled = form.isFilled()
			return formsAreFilled
		})
	}

	return formsAreFilled
}

// refresh MUST be called from the UI thread
func (la *mucRoomConfigListAddComponent) refresh() {
	la.removeAllButton.SetSensitive(la.hasItems())
	la.enableApplyIfConditionsAreMet()
	la.refreshApplyButtonLabel()
}

// refreshApplyButtonLabel MUST be called from the UI thread
func (la *mucRoomConfigListAddComponent) refreshApplyButtonLabel() {
	applyButtonLabel := i18n.Local("Add")
	if la.hasMoreThanOneListItem() {
		applyButtonLabel = i18n.Local("Add all")
	}
	la.applyButton.SetLabel(applyButtonLabel)
}

func (la *mucRoomConfigListAddComponent) hasMoreThanOneListItem() bool {
	totalItems := len(la.formItems)
	hasItemTyped := la.form.isFilled()

	return totalItems > 1 || (totalItems == 1 && hasItemTyped)
}

// enableApplyIfConditionsAreMet MUST be called from the UI thread
func (la *mucRoomConfigListAddComponent) enableApplyIfConditionsAreMet() {
	la.applyButton.SetSensitive(la.areAllFormsFilled())
}

// onRemoveAllClicked MUST be called from the UI thread
func (la *mucRoomConfigListAddComponent) onRemoveAllClicked() {
	itms := la.formItems
	la.formItems = nil

	for _, itm := range itms {
		la.contentBox.Remove(itm.contentBox())
	}

	la.form.resetAndFocusJidEntry()
	la.refresh()
}

// onApplyClicked MUST be called from the UI thread
func (la *mucRoomConfigListAddComponent) onApplyClicked() {
	if la.isValid() {
		jidList := []string{}

		if la.form.isFilled() {
			jidList = append(jidList, la.form.jid())
		}

		la.forEachForm(func(form *roomConfigListForm) bool {
			if form.isFilled() {
				jidList = append(jidList, form.jid())
			}
			return true
		})

		la.onApply(jidList)
		la.close()
	}
}

// isValid MUST be called from the UI thread
func (la *mucRoomConfigListAddComponent) isValid() bool {
	var ok bool
	var err error

	if la.form.isFilled() {
		ok, err = la.isFormValid(la.form)
		if err != nil {
			la.notifyError(la.friendlyErrorMessage(la.form, err))
			return false
		}
	}

	la.forEachForm(func(form *roomConfigListForm) bool {
		if ok, err = la.isFormValid(form); err != nil {
			la.notifyError(la.friendlyErrorMessage(form, err))
			return false
		}
		return true
	})

	return ok
}

// isFormValid MUST be called from the UI thread
func (la *mucRoomConfigListAddComponent) isFormValid(form *roomConfigListForm) (bool, error) {
	if form.jid() == "" {
		return false, errEmptyMemberIdentifier
	}

	if !jid.ValidJID(form.jid()) {
		return false, errInvalidMemberIdentifier
	}

	if !form.isFilled() {
		return false, errRoomConfigListFormNotFilled
	}

	return true, nil
}

// notifyError MUST be called from the UI thread
func (la *mucRoomConfigListAddComponent) notifyError(err string) {
	la.notifications.notifyOnError(err)
}

// close MUST be called from the UI thread
func (la *mucRoomConfigListAddComponent) close() {
	la.dialog.Destroy()
}

// show MUST be called from the UI thread
func (la *mucRoomConfigListAddComponent) show() {
	la.dialog.Show()
}

func (la *mucRoomConfigListAddComponent) hasItems() bool {
	return len(la.formItems) > 0
}

func (la *mucRoomConfigListAddComponent) friendlyErrorMessage(form *roomConfigListForm, err error) string {
	switch err {
	case errEmptyMemberIdentifier:
		return i18n.Local("You must enter the account address.")
	case errInvalidMemberIdentifier:
		return i18n.Local("You must provide a valid account address.")
	default:
		return form.friendlyErrorMessage(err)
	}
}

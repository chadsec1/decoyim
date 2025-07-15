package gui

import (
	"github.com/chadsec1/decoyim/i18n"

	"github.com/chadsec1/decoyim/coylog"
	"github.com/chadsec1/decoyim/session/muc"
	"github.com/coyim/gotk3adapter/gdki"
	"github.com/coyim/gotk3adapter/gtki"
	log "github.com/sirupsen/logrus"
)

var roomConfigPagesFields = map[mucRoomConfigPageID][]muc.RoomConfigFieldType{
	roomConfigInformationPageIndex: {
		muc.RoomConfigFieldName,
		muc.RoomConfigFieldDescription,
		muc.RoomConfigFieldLanguage,
		muc.RoomConfigFieldIsPublic,
		muc.RoomConfigFieldIsPersistent,
	},
	roomConfigAccessPageIndex: {
		muc.RoomConfigFieldPassword,
		muc.RoomConfigFieldIsMembersOnly,
		muc.RoomConfigFieldAllowInvites,
	},
	roomConfigPermissionsPageIndex: {
		muc.RoomConfigFieldWhoIs,
		muc.RoomConfigFieldIsModerated,
		muc.RoomConfigFieldCanChangeSubject,
		muc.RoomConfigFieldAllowPrivateMessages,
		muc.RoomConfigFieldPresenceBroadcast,
	},
	roomConfigPositionsPageIndex: {
		muc.RoomConfigFieldOwners,
		muc.RoomConfigFieldAdmins,
		muc.RoomConfigFieldMembers,
	},
	roomConfigOthersPageIndex: {
		muc.RoomConfigFieldMaxOccupantsNumber,
		muc.RoomConfigFieldMaxHistoryFetch,
		muc.RoomConfigFieldEnableLogging,
		muc.RoomConfigFieldAllowVisitorNickchange,
		muc.RoomConfigFieldAllowVoiceRequest,
		muc.RoomConfigFieldAllowSubscription,
		muc.RoomConfigFieldMembersByDefault,
		muc.RoomConfigFieldAllowVisitorStatus,
		muc.RoomConfigAllowPrivateMessagesFromVisitors,
		muc.RoomConfigPublicList,
	},
}

var roomConfigAdvancedFields = []muc.RoomConfigFieldType{
	muc.RoomConfigFieldAllowQueryUsers,
	muc.RoomConfigFieldPubsub,
	muc.RoomConfigFieldVoiceRequestMinInteval,
}

type roomConfigPage struct {
	u      *gtkUI
	form   *muc.RoomConfigForm
	fields []hasRoomConfigFormField

	title               string
	pageID              mucRoomConfigPageID
	focusableFields     []focusable
	roomConfigComponent *mucRoomConfigComponent

	sw                  gtki.ScrolledWindow `gtk-widget:"room-config-page-scrolled-window"`
	page                gtki.Overlay        `gtk-widget:"room-config-page-overlay"`
	header              gtki.Label          `gtk-widget:"room-config-page-header-label"`
	content             gtki.Box            `gtk-widget:"room-config-page-content"`
	notificationsArea   gtki.Box            `gtk-widget:"notifications-box"`
	autojoinContent     gtki.Box            `gtk-widget:"room-config-autojoin-content"`
	autojoinCheckButton gtki.CheckButton    `gtk-widget:"room-config-autojoin"`

	notifications  *notificationsComponent
	loadingOverlay *loadingOverlayComponent
	doAfterRefresh *callbacksSet

	onShowValidationErrors func()
	onHideValidationErrors func()

	log coylog.Logger
}

func (c *mucRoomConfigComponent) newConfigPage(pageID mucRoomConfigPageID, parent gtki.Window) *roomConfigPage {
	p := &roomConfigPage{
		u:                      c.u,
		roomConfigComponent:    c,
		title:                  configPageDisplayTitle(pageID),
		pageID:                 pageID,
		doAfterRefresh:         newCallbacksSet(),
		onShowValidationErrors: c.onValidationErrors.invokeAll,
		onHideValidationErrors: c.onNoValidationErrors.invokeAll,
		form:                   c.data.configForm,
		log: c.log.WithFields(log.Fields{
			"page": pageID,
		}),
	}

	p.initBuilder()
	p.initDefaults(parent)
	mucStyles.setRoomConfigPageStyle(p.content)

	p.content.SetFocusVAdjustment(p.sw.GetVAdjustment())
	return p
}

func (p *roomConfigPage) initBuilder() {
	builder := newBuilder("MUCRoomConfigPage")
	panicOnDevError(builder.bindObjects(p))
	builder.ConnectSignals(map[string]interface{}{
		"on_autojoin_toggled": func() {
			p.roomConfigComponent.updateAutoJoin(p.autojoinCheckButton.GetActive())
		},
		"on_key_press": p.onKeyPress,
	})

	p.notifications = p.u.newNotificationsComponent()
	p.notificationsArea.Add(p.notifications.contentBox())

	p.loadingOverlay = p.u.newLoadingOverlayComponent()
	p.page.AddOverlay(p.loadingOverlay.overlay)
}

func (p *roomConfigPage) onKeyPress(_ gtki.Widget, ev gdki.Event) bool {
	if keepFocus(ev) {
		return false
	}

	return p.tryChangeFocus(ev)
}

func keepFocus(ev gdki.Event) bool {
	return !isTab(ev) && !isLeftTab(ev)
}

func (p *roomConfigPage) tryChangeFocus(ev gdki.Event) bool {
	i, ok := p.currentFocusableIndex()
	if !ok {
		return false
	}

	f, ok := p.focusableToJumpTo(determineDirection(ev), i)
	if !ok {
		return false
	}

	f.GrabFocus()
	return true
}

func (p *roomConfigPage) currentFocusableIndex() (int, bool) {
	for i, f := range p.focusableFields {
		if f.HasFocus() {
			return i, true
		}
	}

	return 0, false
}

func (p *roomConfigPage) focusableToJumpTo(d int, currentFocusableIndex int) (focusable, bool) {
	targetFocusableIndex := currentFocusableIndex + d

	if p.isValidFocusableIndex(targetFocusableIndex) {
		return p.focusableFields[targetFocusableIndex], true
	}

	return nil, false
}

func determineDirection(ev gdki.Event) int {
	if isTab(ev) {
		return 1
	}

	return -1
}

func (p *roomConfigPage) isValidFocusableIndex(ix int) bool {
	return ix >= 0 && ix < len(p.focusableFields)
}

func (p *roomConfigPage) appendFocusableFields(w ...focusable) {
	p.focusableFields = append(p.focusableFields, w...)
}

func (p *roomConfigPage) appendFocusableFieldsFrom(fields []hasRoomConfigFormField) {
	for _, f := range fields {
		p.appendFocusableFields(f.focusWidget())
	}
}

func (p *roomConfigPage) initDefaults(parent gtki.Window) {
	p.initIntroPage()
	switch p.pageID {
	case roomConfigSummaryPageIndex:
		p.initSummary()
		return
	case roomConfigPositionsPageIndex:
		p.initOccupants(parent)
		return
	case roomConfigOthersPageIndex:
		p.initKnownFields()
		p.initUnknownFields()
		p.initAdvancedOptionsFields()
		return
	}
	p.initKnownFields()
}

func (p *roomConfigPage) initIntroPage() {
	intro := configPageDisplayIntro(p.pageID)
	if intro == "" {
		p.header.SetVisible(false)
		return
	}
	p.header.SetText(intro)
}

func (p *roomConfigPage) initKnownFields() {
	if knownFields, ok := roomConfigPagesFields[p.pageID]; ok {
		booleanFields := []hasRoomConfigFormField{}
		for _, kf := range knownFields {
			if knownField, ok := p.form.GetKnownField(kf); ok {
				field, err := roomConfigFormFieldByType(kf, roomConfigFieldsTexts[kf], knownField.ValueType(), p.onShowValidationErrors, p.onHideValidationErrors)
				if err != nil {
					p.log.WithError(err).Error("Room configuration form field not supported")
					continue
				}
				if f, ok := field.(*roomConfigFormFieldBoolean); ok {
					booleanFields = append(booleanFields, f)
					continue
				}
				p.appendFocusableFields(field.focusWidget())
				p.addField(field)
			}
		}
		if len(booleanFields) > 0 {
			p.appendFields(booleanFields...)
			p.appendFocusableFieldsFrom(booleanFields)
			p.addField(newRoomConfigFormFieldBooleanContainer(booleanFields))
		}
	}
}

func (p *roomConfigPage) initUnknownFields() {
	booleanFields := []hasRoomConfigFormField{}
	for _, ff := range p.form.GetUnknownFields() {
		field, err := roomConfigFormUnknownFieldByType(newRoomConfigFieldTextInfo(ff.Label, ff.Description), ff.ValueType(), p.onShowValidationErrors, p.onHideValidationErrors)
		p.appendFocusableFields(field.focusWidget())
		if err != nil {
			p.log.WithError(err).Error("Room configuration form field not supported")
			continue
		}
		if f, ok := field.(*roomConfigFormFieldBoolean); ok {
			booleanFields = append(booleanFields, f)
			continue
		}
		p.addField(field)
	}
	if len(booleanFields) > 0 {
		p.appendFields(booleanFields...)
		p.addField(newRoomConfigFormFieldBooleanContainer(booleanFields))
	}
}

func (p *roomConfigPage) appendFields(fields ...hasRoomConfigFormField) {
	p.fields = append(p.fields, fields...)
}

func (p *roomConfigPage) initAdvancedOptionsFields() {
	booleanFields := []hasRoomConfigFormField{}
	advancedFields := []hasRoomConfigFormField{}
	for _, aff := range roomConfigAdvancedFields {
		if knownField, ok := p.form.GetKnownField(aff); ok {
			field, err := roomConfigFormFieldByType(aff, roomConfigFieldsTexts[aff], knownField.ValueType(), p.onShowValidationErrors, p.onHideValidationErrors)
			if err != nil {
				p.log.WithError(err).Error("Room configuration form field not supported")
				continue
			}
			if f, ok := field.(*roomConfigFormFieldBoolean); ok {
				booleanFields = append(booleanFields, f)
				continue
			}
			advancedFields = append(advancedFields, field)
		}
	}
	advancedFocusables := append(advancedFields, booleanFields...)

	if len(booleanFields) > 0 {
		advancedFields = append(advancedFields, newRoomConfigFormFieldBooleanContainer(booleanFields))
	}

	if len(advancedFields) > 0 {
		p.appendFields(advancedFields...)
		afc := newRoomConfigFormFieldAdvancedOptionsContainer(advancedFields)
		p.addField(afc)
		p.appendFocusableFields(afc.focusWidget())
		p.appendFocusableFieldsFrom(advancedFocusables)
	}
}

func (p *roomConfigPage) initSummary() {
	p.initSummaryFields(roomConfigInformationPageIndex)
	p.initSummaryFields(roomConfigAccessPageIndex)
	p.initSummaryFields(roomConfigPermissionsPageIndex)
	p.initSummaryFields(roomConfigPositionsPageIndex)
	p.initSummaryFields(roomConfigOthersPageIndex)
	if p.roomConfigComponent.data.roomConfigScenario == roomConfigScenarioCreate {
		p.autojoinCheckButton.SetActive(p.roomConfigComponent.data.autoJoinRoomAfterSaved)
		p.appendFocusableFields(p.autojoinCheckButton)
		p.autojoinContent.Show()
	}
}

func (p *roomConfigPage) initSummaryFields(pageID mucRoomConfigPageID) {
	lb := newRoomConfigFormFieldLinkButton(pageID, p.roomConfigComponent.setCurrentPage)
	if pageID == roomConfigInformationPageIndex {
		p.appendFocusableFields(lb.focusWidget())
	}
	p.addField(lb)

	if pageID == roomConfigPositionsPageIndex {
		p.initOccupantsSummaryFields()
		return
	}

	fields := []hasRoomConfigFormField{}
	for _, kf := range roomConfigPagesFields[pageID] {
		if knownField, ok := p.form.GetKnownField(kf); ok {
			fields = append(fields, newRoomConfigSummaryField(kf, roomConfigFieldsTexts[kf], knownField.ValueType()))
		}
	}

	if pageID == roomConfigOthersPageIndex {
		fields = append(fields, p.otherPageSummaryFields()...)
	}

	p.addField(newRoomConfigSummaryFieldContainer(fields))
}

func (p *roomConfigPage) otherPageSummaryFields() []hasRoomConfigFormField {
	fields := []hasRoomConfigFormField{}

	for _, ff := range p.form.GetUnknownFields() {
		fields = append(fields, newRoomConfigSummaryField(muc.RoomConfigFieldUnexpected, newRoomConfigFieldTextInfo(ff.Label, ff.Description), ff.ValueType()))
	}

	advancedFields := []hasRoomConfigFormField{}
	for _, aff := range roomConfigAdvancedFields {
		if knownField, ok := p.form.GetKnownField(aff); ok {
			advancedFields = append(advancedFields, newRoomConfigSummaryField(aff, roomConfigFieldsTexts[aff], knownField.ValueType()))
		}
	}

	if len(advancedFields) > 0 {
		fields = append(fields, newAdvancedOptionSummaryField(advancedFields))
	}

	return fields
}

func (p *roomConfigPage) initOccupantsSummaryFields() {
	fields := []hasRoomConfigFormField{
		newRoomConfigSummaryOccupantField(i18n.Local("Owners"), p.form.OwnersList),
		newRoomConfigSummaryOccupantField(i18n.Local("Administrators"), p.form.AdminsList),
		newRoomConfigSummaryOccupantField(i18n.Local("Banned"), p.form.BanList),
	}

	p.addField(newRoomConfigSummaryFieldContainer(fields))
}

func (p *roomConfigPage) initOccupants(parent gtki.Window) {
	p.addField(newRoomConfigPositionsField(roomConfigPositionsOptions{
		affiliation:            ownerAffiliation,
		occupantList:           p.form.OwnersList(),
		setOccupantList:        p.form.SetOwnerList,
		setRemovedOccupantList: p.form.UpdateRemovedOccupantList,
		displayErrors:          func() { p.notifyError(i18n.Local("The room must have at least one owner")) },
		parentWindow:           parent,
	}))
	p.content.Add(createSeparator(gtki.HorizontalOrientation))

	p.addField(newRoomConfigPositionsField(roomConfigPositionsOptions{
		affiliation:            adminAffiliation,
		occupantList:           p.form.AdminsList(),
		setOccupantList:        p.form.SetAdminList,
		setRemovedOccupantList: p.form.UpdateRemovedOccupantList,
		parentWindow:           parent,
	}))
	p.content.Add(createSeparator(gtki.HorizontalOrientation))

	p.addField(newRoomConfigPositionsField(roomConfigPositionsOptions{
		affiliation:            outcastAffiliation,
		occupantList:           p.form.BanList(),
		setOccupantList:        p.form.SetBanList,
		setRemovedOccupantList: p.form.UpdateRemovedOccupantList,
		parentWindow:           parent,
	}))
}

func (p *roomConfigPage) addField(field hasRoomConfigFormField) {
	p.appendFields(field)
	p.content.Add(field.fieldWidget())
	p.doAfterRefresh.add(field.refreshContent)
}

// isValid MUST be called from the UI thread
func (p *roomConfigPage) isValid() bool {
	isValid := true
	for _, f := range p.fields {
		if !f.isValid() {
			f.showValidationErrors()
			isValid = false
		}
	}
	return isValid
}

func (p *roomConfigPage) updateFieldValues() {
	for _, f := range p.fields {
		f.updateFieldValue()
	}
}

// refresh MUST be called from the UI thread
func (p *roomConfigPage) refresh() {
	p.page.ShowAll()
	p.hideLoadingOverlay()
	p.clearErrors()
	p.doAfterRefresh.invokeAll()
}

// clearErrors MUST be called from the ui thread
func (p *roomConfigPage) clearErrors() {
	p.notifications.clearErrors()
}

// notifyError MUST be called from the ui thread
func (p *roomConfigPage) notifyError(m string) {
	p.notifications.notifyOnError(m)
}

// onConfigurationApply MUST be called from the ui thread
func (p *roomConfigPage) onConfigurationApply() {
	p.showLoadingOverlay(i18n.Local("Saving room configuration"))
}

// onConfigurationApplyError MUST be called from the ui thread
func (p *roomConfigPage) onConfigurationApplyError() {
	p.hideLoadingOverlay()
}

// showLoadingOverlay MUST be called from the ui thread
func (p *roomConfigPage) showLoadingOverlay(m string) {
	p.loadingOverlay.setSolid()
	p.loadingOverlay.showWithMessage(m)
}

// hideLoadingOverlay MUST be called from the ui thread
func (p *roomConfigPage) hideLoadingOverlay() {
	p.loadingOverlay.hide()
}

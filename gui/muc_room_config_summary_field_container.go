package gui

import (
	"github.com/chadsec1/decoyim/session/muc"
	"github.com/coyim/gotk3adapter/gtki"
)

type roomConfigSummaryFieldContainer struct {
	fields []hasRoomConfigFormField

	widget  gtki.Box     `gtk-widget:"room-config-field-box"`
	content gtki.ListBox `gtk-widget:"room-config-fields-content"`
}

func newRoomConfigSummaryFieldContainer(f []hasRoomConfigFormField) hasRoomConfigFormField {
	field := &roomConfigSummaryFieldContainer{
		fields: f,
	}

	field.initBuilder()
	field.initDefaults()

	return field
}

func (fc *roomConfigSummaryFieldContainer) initBuilder() {
	builder := newBuilder("MUCRoomConfigSummaryFieldContainer")
	panicOnDevError(builder.bindObjects(fc))
}

func (fc *roomConfigSummaryFieldContainer) initDefaults() {
	fields := addSeparatorsBetweenWidgets(fc.fieldWidgets())
	for _, f := range fields {
		fc.content.Add(f)
	}

	fc.disableFocusOnSummaryFields()
}

// disableFocusOnSummaryFields MUST be called from the UI thread
func (fc *roomConfigSummaryFieldContainer) disableFocusOnSummaryFields() {
	for _, f := range fc.content.GetChildren() {
		f.SetCanFocus(false)
		f.SetProperty("activatable", false)
		f.SetProperty("selectable", false)
	}
}

func (fc *roomConfigSummaryFieldContainer) fieldWidgets() (widgets []gtki.Widget) {
	for _, f := range fc.fields {
		widgets = append(widgets, f.fieldWidget())
	}
	return
}

func addSeparatorsBetweenWidgets(fields []gtki.Widget) (widgets []gtki.Widget) {
	widgets = append(widgets, fields[0])
	for _, f := range fields[1:] {
		widgets = append(widgets, createSeparator(gtki.HorizontalOrientation))
		widgets = append(widgets, f)
	}
	return
}

func (fc *roomConfigSummaryFieldContainer) fieldWidget() gtki.Widget {
	return fc.widget
}

// refreshContent MUST NOT be called from the UI thread
func (fc *roomConfigSummaryFieldContainer) refreshContent() {
	for _, f := range fc.fields {
		f.refreshContent()
	}
}

// updateFieldValue MUST be called from the UI thread
func (fc *roomConfigSummaryFieldContainer) updateFieldValue() {
	for _, f := range fc.fields {
		f.updateFieldValue()
	}
}

// isValid implements the hasRoomConfigFormField interface
func (fc *roomConfigSummaryFieldContainer) isValid() bool {
	return true
}

// showValidationErrors implements the hasRoomConfigFormField interface
func (fc *roomConfigSummaryFieldContainer) showValidationErrors() {}

// fieldKey implements the hasRoomConfigFormField interface
func (fc *roomConfigSummaryFieldContainer) fieldKey() muc.RoomConfigFieldType {
	return muc.RoomConfigFieldUnexpected
}

// focusWidget implements the hasRoomConfigFormField interface
func (fc *roomConfigSummaryFieldContainer) focusWidget() focusable {
	return fc.content
}

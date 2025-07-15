package gui

import (
	"github.com/chadsec1/decoyim/text"
	"github.com/coyim/gotk3adapter/gtki"
	"github.com/coyim/gotk3adapter/pangoi"
)

type infoBarHighlightType int

const (
	infoBarHighlightNickname infoBarHighlightType = iota
	infoBarHighlightAffiliation
	infoBarHighlightRole
)

type infoBarHighlightAttributes struct {
	labelNickname    gtki.Label `gtk-widget:"labelNickname"`
	labelAffiliation gtki.Label `gtk-widget:"labelAffiliation"`
	labelRole        gtki.Label `gtk-widget:"labelRole"`
}

func newInfoBarHighlightAttributes(tp infoBarHighlightType) pangoi.AttrList {
	ibh := &infoBarHighlightAttributes{}

	builder := newBuilder("InfoBarHighlightAttributes")
	panicOnDevError(builder.bindObjects(ibh))

	var highlightLabel gtki.Label
	switch tp {
	case infoBarHighlightNickname:
		highlightLabel = ibh.labelNickname
	case infoBarHighlightAffiliation:
		highlightLabel = ibh.labelAffiliation
	case infoBarHighlightRole:
		highlightLabel = ibh.labelRole
	}

	if highlightLabel != nil {
		if vv, err := highlightLabel.GetAttributes(); err == nil {
			return vv
		}
	}

	return nil
}

type infobarHighlightFormatter struct {
	text string
}

func newInfobarHighlightFormatter(text string) *infobarHighlightFormatter {
	return &infobarHighlightFormatter{text}
}

const (
	highlightFormatNickname    = "nickname"
	highlightFormatAffiliation = "affiliation"
	highlightFormatRole        = "role"
)

var highlightFormats = []string{
	highlightFormatNickname,
	highlightFormatAffiliation,
	highlightFormatRole,
}

var infoBarHighlightFormats = map[string]infoBarHighlightType{
	highlightFormatNickname:    infoBarHighlightNickname,
	highlightFormatAffiliation: infoBarHighlightAffiliation,
	highlightFormatRole:        infoBarHighlightRole,
}

func (f *infobarHighlightFormatter) formatLabel(label gtki.Label) {
	formatted, _ := text.ParseWithFormat(f.text)

	text, formats := formatted.Join()
	label.SetText(text)

	pangoAttrList := g.pango.AttrListNew()

	for _, format := range formats {
		if highlightType, ok := infoBarHighlightFormats[format.Format]; ok {
			copy := newInfoBarHighlightAttributes(highlightType)
			copyAttributesTo(pangoAttrList, copy, format.Start, format.Start+format.Length)
		}
	}

	label.SetAttributes(pangoAttrList)
}

func copyAttributesTo(toAttrList, fromAttrList pangoi.AttrList, startIndex, endIndex int) {
	for _, attr := range fromAttrList.GetAttributes() {
		attr.SetStartIndex(startIndex)
		attr.SetEndIndex(endIndex)

		toAttrList.Insert(attr)
	}
}

package gui

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/coyim/gotk3adapter/gtki"
	"github.com/coyim/gotk3adapter/pangoi"
)

const (
	ulIndexID     = 0
	ulIndexWeight = 5
)

var ulAllIndexValues = []int{0, 1, 2, 3, 4, 5, 6, 7, 8}

type unifiedLayout struct {
	ui                       *gtkUI
	cl                       *conversationList
	leftPane                 gtki.Box
	rightPane                gtki.Box      `gtk-widget:"right"`
	notebook                 gtki.Notebook `gtk-widget:"notebook"`
	header                   gtki.Label    `gtk-widget:"header_label"`
	headerBox                gtki.Box      `gtk-widget:"header_box"`
	close                    gtki.Button   `gtk-widget:"close_button"`
	convsVisible             bool
	inPageSet                bool
	isFullscreen             bool
	originalPosition         *windowPosition
	originalExpandedPosition *windowPosition
	itemMap                  map[int]*conversationStackItem
}

type windowPosition struct {
	posX int
	posY int
}

type conversationList struct {
	layout *unifiedLayout
	view   gtki.TreeView  `gtk-widget:"treeview"`
	model  gtki.ListStore `gtk-widget:"liststore"`
}

type conversationStackItem struct {
	*conversationPane
	pageIndex      int
	needsAttention bool
	iter           gtki.TreeIter
	layout         *unifiedLayout
}

func newUnifiedLayout(ui *gtkUI, left, parent gtki.Box) *unifiedLayout {
	ul := &unifiedLayout{
		ui:                       ui,
		cl:                       &conversationList{},
		originalPosition:         &windowPosition{0, 0},
		originalExpandedPosition: &windowPosition{0, 0},
		leftPane:                 left,
		itemMap:                  make(map[int]*conversationStackItem),
		isFullscreen:             false,
	}
	ul.cl.layout = ul

	builder := newBuilder("UnifiedLayout")
	panicOnDevError(builder.bindObjects(ul.cl))
	panicOnDevError(builder.bindObjects(ul))

	//ul.cl.model needs to be kept beyond the lifespan of the builder.
	ul.cl.model.Ref()
	runtime.SetFinalizer(ul.cl, func(cl interface{}) {
		cl.(*conversationList).model.Unref()
		cl.(*conversationList).model = nil
	})

	builder.ConnectSignals(map[string]interface{}{
		"on_activate":    ul.cl.onActivate,
		"on_clicked":     ul.onCloseClicked,
		"on_switch_page": ul.onSwitchPage,
	})

	connectShortcut("<Primary>Page_Down", ul.ui.mainUI.window, ul.nextTab)
	connectShortcut("<Primary>Page_Up", ul.ui.mainUI.window, ul.previousTab)
	connectShortcut("F11", ul.ui.mainUI.window, ul.toggleFullscreen)

	parent.PackStart(ul.rightPane, false, true, 0)
	parent.SetChildPacking(ul.leftPane, false, true, 0, gtki.PACK_START)

	ul.rightPane.Hide()

	left.SetHAlign(gtki.ALIGN_FILL)
	left.SetHExpand(true)
	return ul
}

func (ul *unifiedLayout) onConversationChanged(csi *conversationStackItem) {
	if !csi.isCurrent() {
		csi.needsAttention = true
		csi.applyTextWeight()
	}
}

func (cl *conversationList) add(csi *conversationStackItem) {
	if csi.iter == nil {
		csi.iter = cl.model.Append()
		cl.updateItem(csi)
	}
}

func (cl *conversationList) remove(csi *conversationStackItem) {
	if csi.iter == nil {
		return
	}
	cl.model.Remove(csi.iter)
	csi.iter = nil
}

func (cl *conversationList) updateItem(csi *conversationStackItem) {
	cs := cl.layout.ui.currentColorSet()
	peer, ok := csi.currentPeer()
	if !ok {
		csi.Log().WithField("peer", csi.target.NoResource()).Warn("No peer found for")
		return
	}
	_ = cl.model.Set2(csi.iter, ulAllIndexValues, []interface{}{
		csi.pageIndex,
		csi.shortName(),
		peer.Jid.String(),
		decideColorFor(cs, peer),
		cs.rosterPeerBackground,
		csi.getTextWeight(),
		createTooltipFor(peer),
		statusIcons[decideStatusFor(peer)].GetPixbuf(),
		csi.getUnderline(),
	},
	)
}

func (u *gtkUI) getPositionFromCurrent() *windowPosition {
	posx, posy := u.mainUI.window.GetPosition()
	return &windowPosition{
		posX: posx,
		posY: posy,
	}
}

func (pos *windowPosition) equals(other *windowPosition) bool {
	return pos.posX == other.posX && pos.posY == other.posY
}

func (ul *unifiedLayout) showConversations() {
	if ul.convsVisible {
		return
	}

	ul.originalPosition = ul.ui.getPositionFromCurrent()

	ul.leftPane.SetHExpand(false)
	ul.rightPane.SetHExpand(true)

	ul.ui.mainUI.window.Resize(934, 600)
	ul.rightPane.Show()

	ul.convsVisible = true
	ul.update()
	go func() {
		time.Sleep(time.Duration(1) * time.Second)
		doInUIThread(func() {
			ul.originalExpandedPosition = ul.ui.getPositionFromCurrent()
		})
	}()
}

func moveTo(ul *unifiedLayout) {
	time.Sleep(time.Duration(20) * time.Millisecond)
	ul.ui.mainUI.window.Move(ul.originalPosition.posX, ul.originalPosition.posY)
}

func (ul *unifiedLayout) hideConversations() {
	if !ul.convsVisible {
		return
	}

	currentPos := ul.ui.getPositionFromCurrent()

	width := ul.leftPane.GetAllocatedWidth()
	height := ul.ui.mainUI.window.GetAllocatedHeight()
	ul.rightPane.SetHExpand(false)
	ul.rightPane.Hide()
	ul.leftPane.SetHExpand(true)
	ul.ui.mainUI.window.Resize(width, height)
	ul.convsVisible = false

	if currentPos.equals(ul.originalExpandedPosition) {
		go moveTo(ul)
	}
}

func (csi *conversationStackItem) isVisible() bool {
	return csi.isCurrent() && csi.layout.ui.mainUI.window.HasToplevelFocus()
}

func (csi *conversationStackItem) setEnabled(enabled bool) {
	csi.Log().WithField("enabled", enabled).Debug("csi.SetEnabled()")
}

func (csi *conversationStackItem) shortName() string {
	// TODO: this might be unsafe - it should use the JID methods
	ss := strings.Split(csi.target.NoResource().String(), "@")
	uiName := ss[0]

	peer, ok := csi.currentPeer()
	// TODO: this logic is definitely a bit iffy, and should be fixed.
	if ok && peer.NameForPresentation() != peer.Jid.String() {
		uiName = peer.NameForPresentation()
	}

	return uiName
}

func (csi *conversationStackItem) isCurrent() bool {
	if csi == nil {
		return false
	}
	return csi.layout.notebook.GetCurrentPage() == csi.pageIndex
}

func (csi *conversationStackItem) getUnderline() int {
	if csi.isCurrent() {
		return pangoi.UNDERLINE_SINGLE
	}
	return pangoi.UNDERLINE_NONE
}

func (csi *conversationStackItem) getTextWeight() int {
	if csi.needsAttention {
		return 700
	}
	return 500
}

func (csi *conversationStackItem) applyTextWeight() {
	if csi.iter == nil {
		return
	}
	weight := csi.getTextWeight()
	_ = csi.layout.cl.model.SetValue(csi.iter, ulIndexWeight, weight)
}

func (csi *conversationStackItem) show(userInitiated bool) {
	csi.layout.showConversations()
	csi.updateSecurityWarning()
	csi.layout.cl.add(csi)
	csi.widget.Show()

	if userInitiated {
		csi.bringToFront()
		return
	}
	if !csi.isCurrent() {
		csi.needsAttention = true
		csi.applyTextWeight()
	}
}

func (csi *conversationStackItem) potentialTarget() string {
	p := csi.target.PotentialResource().String()
	if p != "" {
		return fmt.Sprintf(" (%s)", p)
	}
	return ""
}

func (csi *conversationStackItem) bringToFront() {
	csi.layout.showConversations()
	csi.needsAttention = false
	csi.applyTextWeight()
	csi.layout.setCurrentPage(csi)
	title := windowConversationTitle(csi.layout.ui, csi.currentPeerForSending(), csi.account, csi.potentialTarget())
	csi.layout.ui.mainUI.window.SetTitle(title)
	csi.entry.GrabFocus()
	csi.layout.update()
}

func (csi *conversationStackItem) remove() {
	csi.layout.cl.remove(csi)
	csi.widget.Hide()
}

func (csi *conversationStackItem) destroy() {
	csi.layout.closeNthConversation(csi.pageIndex)
	csi.widget.Destroy()
}

func (cl *conversationList) getItemForIter(iter gtki.TreeIter) *conversationStackItem {
	val, err := cl.model.GetValue(iter, ulIndexID)
	if err != nil {
		log.WithError(err).Warn("Error getting ulIndexID value")
		return nil
	}
	gv, err := val.GoValue()
	if err != nil {
		log.WithError(err).Warn("Error getting GoValue for ulIndexID")
		return nil
	}
	return cl.layout.itemMap[gv.(int)]
}

func (cl *conversationList) onActivate(v gtki.TreeView, path gtki.TreePath) {
	iter, err := cl.model.GetIter(path)
	if err != nil {
		log.WithError(err).Warn("Error converting path to iter")
		return
	}
	csi := cl.getItemForIter(iter)
	if csi != nil {
		csi.bringToFront()
		cl.removeSelection()
	}
}

func (cl *conversationList) removeSelection() {
	ts, _ := cl.view.GetSelection()
	if _, iter, ok := ts.GetSelected(); ok {
		path, _ := cl.model.GetPath(iter)
		ts.UnselectPath(path)
	}
}

func (ul *unifiedLayout) setCurrentPage(csi *conversationStackItem) {
	ul.inPageSet = true
	ul.notebook.SetCurrentPage(csi.pageIndex)
	ul.update()
	ul.inPageSet = false
}

func (ul *unifiedLayout) closeNthConversation(n int) {
	if n >= 0 {
		item := ul.itemMap[n]
		if item != nil {
			item.remove()
		}
	}

	if !ul.displayFirstConvo() {
		ul.header.SetText("")
		ul.hideConversations()
	}
}

func (ul *unifiedLayout) onCloseClicked() {
	page := ul.notebook.GetCurrentPage()
	ul.closeNthConversation(page)
}

func (ul *unifiedLayout) onSwitchPage(notebook gtki.Notebook, page gtki.Widget, idx int) {
	if ul.inPageSet {
		return
	}
	if csi := ul.itemMap[idx]; csi != nil {
		csi.bringToFront()
	}
	ul.cl.removeSelection()
}

func (ul *unifiedLayout) displayFirstConvo() bool {
	if iter, ok := ul.cl.model.GetIterFirst(); ok {
		if csi := ul.cl.getItemForIter(iter); csi != nil {
			csi.bringToFront()
			return true
		}
	}
	return false
}

func (ul *unifiedLayout) nextTab(gtki.Window) {
	page := ul.notebook.GetCurrentPage()
	np := (ul.notebook.GetNPages() - 1)
	if page < 0 || np < 0 {
		return
	}
	if page == np {
		ul.notebook.SetCurrentPage(0)
	} else {
		ul.notebook.NextPage()
	}
	ul.update()
}

func (ul *unifiedLayout) previousTab(gtki.Window) {
	page := ul.notebook.GetCurrentPage()
	np := (ul.notebook.GetNPages() - 1)
	if page < 0 || np < 0 {
		return
	}
	if page > 0 {
		ul.notebook.PrevPage()
	} else {
		ul.notebook.SetCurrentPage(np)
	}
	ul.update()
}

func (ul *unifiedLayout) toggleFullscreen(gtki.Window) {
	if ul.isFullscreen {
		ul.ui.mainUI.window.Unfullscreen()
	} else {
		ul.ui.mainUI.window.Fullscreen()
	}
	ul.isFullscreen = !ul.isFullscreen
}

func (ul *unifiedLayout) update() {
	for it, ok := ul.cl.model.GetIterFirst(); ok; ok = ul.cl.model.IterNext(it) {
		csi := ul.cl.getItemForIter(it)
		if csi != nil {
			ul.cl.updateItem(csi)
		}
	}
}

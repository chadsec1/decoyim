package gui

import (
	"github.com/chadsec1/decoyim/i18n"
	"github.com/coyim/gotk3adapter/gtki"
)

// In these notifications we will use the convention that:
// - "secure transfer" means that you are sending or receiving something encrypted from/to a peer that is verified
// - "encrypted transfer" means transfer to/from a peer that is not verified
// - "insecure transfer" is unencrypted

type fileNotification struct {
	area                      gtki.Box   `gtk-widget:"area-file-transfer-info"`
	label                     gtki.Label `gtk-widget:"name-file-transfer-info"`
	image                     gtki.Image `gtk-widget:"image-file-transfer-info"`
	name                      string
	progress                  float64
	state                     string
	directory                 bool
	sending                   bool
	receiving                 bool
	encrypted                 bool
	verifiedPeer              bool
	haveEncryptionInformation bool
	afterCancelHook           func()
	afterFailHook             func()
	afterSucceedHook          func()
	afterDeclinedHook         func()
	canceledProvider          gtki.CssProvider
	successProvider           gtki.CssProvider
}

type fileTransferNotification struct {
	area          gtki.Box         `gtk-widget:"file-transfer"`
	image         gtki.Image       `gtk-widget:"image-file-transfer"`
	label         gtki.Label       `gtk-widget:"label-file-transfer"`
	box           gtki.Box         `gtk-widget:"info-file-transfer"`
	progressBar   gtki.ProgressBar `gtk-widget:"bar-file-transfer"`
	button        gtki.Button      `gtk-widget:"button-file-transfer"`
	labelButton   gtki.Label       `gtk-widget:"button-label-file-transfer"`
	totalProgress float64
	files         []*fileNotification
	count         int
	canceled      bool
}

func resizeFileName(name string) string {
	var fileName string

	if len(name) > 20 {
		fileName = name[:20] + "..."
		return fileName
	}

	return name
}

func (file *fileNotification) afterCancel(f func()) {
	file.afterCancelHook = f
}

func (file *fileNotification) afterDeclined(f func()) {
	file.afterDeclinedHook = f
}

func (file *fileNotification) afterFail(f func()) {
	file.afterFailHook = f
}

func (file *fileNotification) afterSucceed(f func()) {
	file.afterSucceedHook = f
}

func (file *fileNotification) destroy() {
	file.cancel()
}

func (file *fileNotification) update(fileName string, prov gtki.CssProvider) {
	updateWithStyle(file.label, prov)
	file.label.SetLabel(fileName)
	file.image.Hide()
}

func (b *builder) fileTransferNotifInit() *fileTransferNotification {
	fileTransferNotif := &fileTransferNotification{}

	panicOnDevError(b.bindObjects(fileTransferNotif))

	return fileTransferNotif
}

func (file *fileNotification) setEncryptionInformation(encrypted, verifiedPeer bool) {
	file.encrypted = encrypted
	file.verifiedPeer = verifiedPeer
	file.haveEncryptionInformation = true
}

func (conv *conversationPane) newFileTransfer(fileName string, dir, send, receive bool) *fileNotification {
	if !conv.fileTransferNotif.area.IsVisible() {
		prov := providerFromCSSFile(conv.account, "file transfer notification", "file_transfer_notification_new.css")
		updateWithStyle(conv.fileTransferNotif.area, prov)

		conv.fileTransferNotif.progressBar.SetFraction(0.0)
		conv.fileTransferNotif.canceled = false
	}

	info := conv.createFileTransferNotification(fileName, dir, send, receive)
	info.updateLabel()

	conv.fileTransferNotif.area.SetVisible(true)

	countSending := 0
	countReceiving := 0

	label := i18n.Local("Transfer started")

	for _, f := range conv.fileTransferNotif.files {
		if f.sending {
			countSending++
		}
		if f.receiving {
			countReceiving++
		}
	}

	cc := i18n.Local("Cancel")

	if countSending > 0 && countReceiving == 0 {
		doInUIThread(func() {
			conv.updateFileTransferNotification(label, cc, "filetransfer_send.svg")
		})
	} else if countSending == 0 && countReceiving > 0 {
		doInUIThread(func() {
			conv.updateFileTransferNotification(label, cc, "filetransfer_receive.svg")
		})
	} else if countSending > 0 && countReceiving > 0 {
		doInUIThread(func() {
			conv.updateFileTransferNotification(label, cc, "filetransfer_receive_send.svg")
		})
	}

	return info
}

func (file *fileNotification) updateLabel() {
	var label string
	switch {
	case file.sending && !file.haveEncryptionInformation:
		label = i18n.Localf("Sending: %s", file.name)
	case !file.sending && !file.haveEncryptionInformation:
		label = i18n.Localf("Receiving: %s", file.name)
	case file.sending && file.encrypted && file.verifiedPeer:
		label = i18n.Localf("Sending securely: %s", file.name)
	case file.sending && file.encrypted:
		label = i18n.Localf("Sending encrypted: %s", file.name)
	case file.sending:
		label = i18n.Localf("Sending insecurely: %s", file.name)
	case file.encrypted && file.verifiedPeer:
		label = i18n.Localf("Receiving securely: %s", file.name)
	case file.encrypted:
		label = i18n.Localf("Receiving encrypted: %s", file.name)
	default:
		label = i18n.Localf("Receiving insecurely: %s", file.name)
	}

	doInUIThread(func() {
		file.label.SetLabel(label)
	})
}

func (conv *conversationPane) createFileTransferNotification(fileName string, dir, send, receive bool) *fileNotification {
	b := newBuilder("FileTransferNotification")

	file := &fileNotification{directory: dir, sending: send, receiving: receive, state: stateInProgress}

	panicOnDevError(b.bindObjects(file))

	b.ConnectSignals(map[string]interface{}{
		"on_destroy_single_file_transfer": file.destroy,
	})

	file.name = fileName

	file.updateLabel()

	conv.fileTransferNotif.count++
	conv.fileTransferNotif.canceled = false
	conv.fileTransferNotif.totalProgress = 0

	conv.fileTransferNotif.box.Add(file.area)
	file.area.ShowAll()

	conv.fileTransferNotif.files = append(conv.fileTransferNotif.files, file)

	file.canceledProvider = providerFromCSSFile(conv.account, "canceled file transfer", "file_transfer_notification_canceled.css")
	file.successProvider = providerFromCSSFile(conv.account, "succeeded file transfer", "file_transfer_notification_success.css")

	return file
}

func (conv *conversationPane) updateFileTransferNotification(label, buttonLabel, image string) {
	if buttonLabel == i18n.Local("Close") {
		prov := providerFromCSSFile(conv.account, "file transfer close", "file_transfer_notification_close_button.css")
		updateWithStyle(conv.fileTransferNotif.labelButton, prov)
	}
	conv.account.log.WithField("label", label).Info("Updating notification")

	conv.fileTransferNotif.label.SetLabel(label)
	conv.fileTransferNotif.labelButton.SetLabel(buttonLabel)
	setImageFromFile(conv.fileTransferNotif.image, image)
}

const stateInProgress = "in-progress"
const stateSuccess = "success"
const stateFailed = "failed"
const stateCanceled = "canceled"
const stateDeclined = "declined"

func (conv *conversationPane) updateFileTransfer(file *fileNotification) {
	conv.fileTransferNotif.totalProgress = 0
	count := 0
	haveSuccess := false
	for _, f := range conv.fileTransferNotif.files {
		switch f.state {
		case stateInProgress:
			count++
			conv.fileTransferNotif.totalProgress += f.progress
		case stateSuccess:
			haveSuccess = true
		}
	}

	var upd float64
	if count == 0 {
		if haveSuccess {
			upd = 100
		} else {
			upd = conv.fileTransferNotif.totalProgress
		}
	} else {
		upd = conv.fileTransferNotif.totalProgress / float64(count)
	}

	doInUIThread(func() {
		conv.fileTransferNotif.progressBar.SetFraction(upd)
	})
}

func fileTransferSuccessLabel(fileTransfer, dirTransfer bool) string {
	if fileTransfer && dirTransfer {
		return i18n.Local("File and directory transfer(s) successful")
	}
	if dirTransfer {
		return i18n.Local("Directory transfer(s) successful")
	}
	return i18n.Local("File transfer(s) successful")
}

func fileTransferFailedLabel(fileTransfer, dirTransfer bool) string {
	if fileTransfer && dirTransfer {
		return i18n.Local("File and directory transfer(s) failed")
	}
	if dirTransfer {
		return i18n.Local("Directory transfer(s) failed")
	}
	return i18n.Local("File transfer(s) failed")
}

func fileTransferCanceledLabel(fileTransfer, dirTransfer bool) string {
	if fileTransfer && dirTransfer {
		return i18n.Local("File and directory transfer(s) canceled")
	}
	if dirTransfer {
		return i18n.Local("Directory transfer(s) canceled")
	}
	return i18n.Local("File transfer(s) canceled")
}

func fileTransferDeclinedLabel(fileTransfer, dirTransfer bool) string {
	if fileTransfer && dirTransfer {
		return i18n.Local("File and directory transfer(s) declined")
	}
	if dirTransfer {
		return i18n.Local("Directory transfer(s) declined")
	}
	return i18n.Local("File transfer(s) declined")
}

func fileTransferCalculateStates(countCompleted, countCanceled, countFailed, countDeclined, countDirs, countDirsCompleted, countTotal int, canceledBefore bool) (label, image string, canceled bool) {
	generateLabel := fileTransferSuccessLabel
	image = "success.svg"
	canceled = canceledBefore
	if countCanceled+countFailed+countDeclined == countTotal {
		image = "failure.svg"
		canceled = true
		generateLabel = fileTransferFailedLabel
		if countCanceled > (countFailed + countDeclined) {
			generateLabel = fileTransferCanceledLabel
		} else if countDeclined > (countCanceled + countFailed) {
			generateLabel = fileTransferDeclinedLabel
		}
	}

	label = generateLabel(countCompleted != countDirsCompleted, countDirsCompleted > 0)

	return
}

func (conv *conversationPane) updateFileTransferNotificationCounts() {
	countCompleted := 0
	countCanceled := 0
	countFailed := 0
	countTotal := 0
	countDirs := 0
	countDirsCompleted := 0
	countDeclined := 0
	for _, f := range conv.fileTransferNotif.files {
		switch f.state {
		case stateInProgress:
		case stateSuccess:
			countCompleted++
			if f.directory {
				countDirsCompleted++
			}
		case stateCanceled:
			countCompleted++
			countCanceled++
		case stateFailed:
			countCompleted++
			countFailed++
		case stateDeclined:
			countCompleted++
			countDeclined++
		}
		if f.directory {
			countDirs++
		}
		countTotal++
	}

	conv.fileTransferNotif.count = countTotal - countCompleted
	if countCompleted == countTotal {
		label, image, c := fileTransferCalculateStates(countCompleted, countCanceled, countFailed, countDeclined, countDirs, countDirsCompleted, countTotal, conv.fileTransferNotif.canceled)
		conv.fileTransferNotif.canceled = c
		doInUIThread(func() {
			conv.updateFileTransferNotification(label, i18n.Local("Close"), image)
		})
	}
}

func (conv *conversationPane) isFileTransferNotifCanceled() bool {
	return conv.fileTransferNotif.canceled
}

func (file *fileNotification) decline() {
	if file.state != stateInProgress {
		return
	}
	file.state = stateDeclined
	file.progress = 0
	file.update(i18n.Localf("Declined: %s", file.name), file.canceledProvider)
	hook := file.afterDeclinedHook
	file.afterDeclinedHook = nil
	if hook != nil {
		hook()
	}
}

func (file *fileNotification) cancel() {
	if file.state != stateInProgress {
		return
	}
	file.state = stateCanceled
	file.progress = 0
	file.update(i18n.Localf("Canceled: %s", file.name), file.canceledProvider)
	hook := file.afterCancelHook
	file.afterCancelHook = nil
	if hook != nil {
		hook()
	}
}

func (file *fileNotification) fail() {
	if file.state != stateInProgress {
		return
	}
	file.state = stateFailed
	file.progress = 0
	file.update(i18n.Localf("Failed: %s", file.name), file.canceledProvider)
	hook := file.afterFailHook
	file.afterFailHook = nil
	if hook != nil {
		hook()
	}
}

func (file *fileNotification) succeed() {
	if file.state != stateInProgress {
		return
	}
	file.state = stateSuccess

	var label string

	switch {
	case file.sending && file.encrypted && file.verifiedPeer:
		label = i18n.Localf("Sent securely: %s", file.name)
	case file.sending && file.encrypted:
		label = i18n.Localf("Sent encrypted: %s", file.name)
	case file.sending:
		label = i18n.Localf("Sent insecurely: %s", file.name)
	case file.encrypted && file.verifiedPeer:
		label = i18n.Localf("Received securely: %s", file.name)
	case file.encrypted:
		label = i18n.Localf("Received encrypted: %s", file.name)
	default:
		label = i18n.Localf("Received insecurely: %s", file.name)
	}

	file.update(label, file.successProvider)

	hook := file.afterSucceedHook
	file.afterSucceedHook = nil
	if hook != nil {
		hook()
	}
}

func (conv *conversationPane) onDestroyFileTransferNotif() {
	label := conv.fileTransferNotif.labelButton.GetLabel()
	if label == i18n.Local("Cancel") {
		for _, f := range conv.fileTransferNotif.files {
			f.cancel()
		}
	} else {
		conv.fileTransferNotif.canceled = false
		conv.fileTransferNotif.area.SetVisible(false)
		conv.fileTransferNotif.progressBar.SetFraction(0.0)
		for i := range conv.fileTransferNotif.files {
			conv.fileTransferNotif.files[i].area.Destroy()
		}
		conv.fileTransferNotif.files = conv.fileTransferNotif.files[:0]
		conv.fileTransferNotif.count = 0
	}
}

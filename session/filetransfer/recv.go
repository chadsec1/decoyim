package filetransfer

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	sdata "github.com/chadsec1/decoyim/session/data"
	"github.com/chadsec1/decoyim/xmpp/data"
	"github.com/chadsec1/decoyim/xmpp/jid"
)

// This contains all valid file transfer methods. A higher value is better, and if possible, we will choose the method with the highest value
var supportedFileTransferMethods = map[string]int{}

var fileTransferCancelListeners = map[string]func(*recvContext){}

// registerRecieveFileTransferMethod registers a known method for receiving data
// prio is a number, higher is better
func registerRecieveFileTransferMethod(name string, prio int, cancelListener func(*recvContext)) {
	supportedFileTransferMethods[name] = prio
	fileTransferCancelListeners[name] = cancelListener
}

type recvContext struct {
	s           canSendIQAndHasLogAndConnection
	sid         string
	peer        string
	mime        string
	options     []string
	date        string
	hash        string
	name        string
	size        int64
	desc        string
	directory   bool
	destination string
	opaque      interface{}
	control     *sdata.FileTransferControl
	enc         *encryptionParameters
}

func extractFileTransferOptions(f data.Form) ([]string, error) {
	if f.Type != "form" || len(f.Fields) != 1 || f.Fields[0].Var != "stream-method" || f.Fields[0].Type != "list-single" {
		return nil, fmt.Errorf("Invalid form for file transfer initiation: %#v", f)
	}
	var result []string
	for _, opt := range f.Fields[0].Options {
		result = append(result, opt.Value)
	}
	return result, nil
}

// chooseAppropriateFileTransferOptionFrom returns the file transfer option that has the highest score
// or not OK if no acceptable options are available
func chooseAppropriateFileTransferOptionFrom(options []string) (best string, ok bool) {
	bestScore := -1
	for _, opt := range options {
		score, has := supportedFileTransferMethods[opt]
		if has {
			ok = true
			if score > bestScore {
				bestScore = score
				best = opt
			}
		}

	}
	return
}

func iqResultChosenStreamMethod(opt string) data.SI {
	return data.SI{
		File: &data.File{},
		Feature: data.FeatureNegotation{
			Form: data.Form{
				Type: "submit",
				Fields: []data.FormFieldX{
					{Var: "stream-method", Values: []string{opt}},
				},
			},
		},
	}
}

func (ctx *recvContext) finalizeFileTransfer(tempName string) error {
	if ctx.directory {
		defer func() {
			_ = os.Remove(tempName)
		}()

		if err := unpack(tempName, ctx.destination); err != nil {
			ctx.s.Log().WithField("destination", ctx.destination).WithError(err).Warn("problem unpacking data")
			ctx.control.ReportError(errors.New("Couldn't unpack final file"))
			return err
		}
	} else {
		if err := os.Rename(tempName, ctx.destination); err != nil {
			ctx.s.Log().WithField("destination", ctx.destination).WithError(err).Warn("couldn't rename file")
			ctx.control.ReportError(errors.New("Couldn't save final file"))
			return err
		}
	}

	ctx.control.ReportFinished()
	removeInflightRecv(ctx.sid)

	return nil
}

func (ctx *recvContext) openDestinationTempFile() (f *os.File, err error) {
	// By creating a temp file next to the place where the real file should be saved
	// we avoid problems on linux when trying to os.Rename later - if tmp filesystem is different
	// than the destination file system. It also serves as an early permissions check.
	// If the transfer is a directory, we will save the zip file here, instead of the actual file. But it should have the same result
	f, err = ioutil.TempFile(filepath.Dir(ctx.destination), filepath.Base(ctx.destination))
	if err != nil {
		ctx.opaque = nil
		ctx.s.Log().WithError(err).Warn("problem creating temporary file")
		ctx.control.ReportError(errors.New("Couldn't open local temporary file"))
		removeInflightRecv(ctx.sid)
	}
	return
}

func waitForFileTransferUserAcceptance(stanza *data.ClientIQ, si data.SI, acceptResult <-chan *string, ctx *recvContext) {
	result := <-acceptResult

	var error *data.ErrorReply
	if result != nil {
		opt, ok := chooseAppropriateFileTransferOptionFrom(ctx.options)
		if ok {
			setInflightRecvDestination(si.ID, *result)
			ctx.s.SendIQResult(stanza, iqResultChosenStreamMethod(opt))
			go fileTransferCancelListeners[opt](ctx)
			return
		}
		ctx.control.ReportError(errors.New("No mutually acceptable file transfer methods available"))
		error = &iqErrorBadRequest
	} else {
		error = &iqErrorForbidden
	}
	removeInflightRecv(si.ID)
	ctx.s.SendIQError(stanza, *error)
}

func registerNewFileTransfer(s hasLogConnectionIQSymmetricKeyAndIsPublisher, si data.SI, options []string, stanza *data.ClientIQ, ctl *sdata.FileTransferControl, isDir bool, encrypted bool) *recvContext {
	ctx := &recvContext{
		s:         s,
		sid:       si.ID,
		mime:      si.MIMEType,
		options:   options,
		peer:      stanza.From,
		directory: isDir,
		control:   ctl,
		enc:       generateEncryptionParameters(encrypted, func() []byte { return s.GetAndWipeSymmetricKeyFor(jid.Parse(stanza.From)) }, "external"),
	}

	if encrypted {
		ctx.name = si.EncryptedData.Name
		ctx.size = si.EncryptedData.Size
	} else {
		ctx.date = si.File.Date
		ctx.hash = si.File.Hash
		ctx.name = si.File.Name
		ctx.size = si.File.Size
		ctx.desc = si.File.Desc
	}

	addInflightRecv(ctx)
	return ctx
}

package session

import (
	"github.com/chadsec1/decoyim/session/access"
	"github.com/chadsec1/decoyim/session/data"
	"github.com/chadsec1/decoyim/session/filetransfer"
	xdata "github.com/chadsec1/decoyim/xmpp/data"
	"github.com/chadsec1/decoyim/xmpp/jid"
	log "github.com/sirupsen/logrus"
)

func init() {
	registerKnownIQ("set", "http://jabber.org/protocol/bytestreams query", func(s access.Session, ciq *xdata.ClientIQ) (interface{}, string, bool) {
		return filetransfer.BytestreamQuery(s, ciq)
	})

	registerKnownIQ("set", "http://jabber.org/protocol/ibb open", func(s access.Session, ciq *xdata.ClientIQ) (interface{}, string, bool) {
		return filetransfer.IbbOpen(s, ciq)
	})

	registerKnownIQ("set", "http://jabber.org/protocol/ibb data", func(s access.Session, ciq *xdata.ClientIQ) (interface{}, string, bool) {
		return filetransfer.IbbData(s, ciq)
	})

	registerKnownIQ("set", "http://jabber.org/protocol/ibb close", func(s access.Session, ciq *xdata.ClientIQ) (interface{}, string, bool) {
		return filetransfer.IbbClose(s, ciq)
	})

	registerKnownExtension("http://jabber.org/protocol/ibb data", func(s access.Session, cm *xdata.ClientMessage, ext *xdata.Extension) {
		filetransfer.IbbMessageData(s, cm, ext)
	})
}

func (s *session) SendFileTo(peer jid.Any, filename string, onNoEnc func() bool, encDecision func(bool)) *data.FileTransferControl {
	s.log.WithFields(log.Fields{
		"peer":     peer,
		"filename": filename,
	}).Info("SendFileTo()")
	return filetransfer.InitSend(s, peer, filename, onNoEnc, encDecision)
}

func (s *session) SendDirTo(peer jid.Any, dirname string, onNoEnc func() bool, encDecision func(bool)) *data.FileTransferControl {
	s.log.WithFields(log.Fields{
		"peer":    peer,
		"dirname": dirname,
	}).Info("SendDirTo()")
	return filetransfer.InitSendDir(s, peer, dirname, onNoEnc, encDecision)
}

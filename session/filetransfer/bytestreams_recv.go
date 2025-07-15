package filetransfer

import (
	"bytes"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"io"
	"os"

	"github.com/chadsec1/decoyim/digests"
	"github.com/chadsec1/decoyim/xmpp/data"
)

func init() {
	registerRecieveFileTransferMethod(BytestreamMethod, 1, bytestreamWaitForCancel)
}

func bytestreamWaitForCancel(ctx *recvContext) {
	ctx.control.WaitForCancel(func() {
		ch := ctx.opaque
		if ch != nil {
			ch.(chan bool) <- true
		}
		removeInflightRecv(ctx.sid)
	})
}

func bytestreamInitialSetup(s canSendIQErrorAndHasLog, stanza *data.ClientIQ) (tag data.BytestreamQuery, ctx *recvContext, earlyReturn bool) {
	if err := xml.NewDecoder(bytes.NewBuffer(stanza.Query)).Decode(&tag); err != nil || tag.Sid == "" {
		s.Log().WithError(err).Warn("Failed to parse bytestream open")
		s.SendIQError(stanza, iqErrorIBBBadRequest)
		return tag, ctx, true
	}

	ctx, ok := getInflightRecv(tag.Sid)

	if !ok || ctx.opaque != nil {
		s.Log().WithField("SID", tag.Sid).Warn("No file transfer associated with SID")
		s.SendIQError(stanza, iqErrorNotAcceptable)
		return tag, ctx, true
	}

	if tag.Mode == "udp" {
		// This shouldn't really be possible, since we don't advertise udp support
		// But we can always register the error anyway.
		s.Log().Warn("Received a request for UDP, even though we don't support or advertize UDP - this means the peer is using a non-conforming application")
		s.SendIQError(stanza, iqErrorIBBBadRequest)
		return tag, ctx, true
	}

	ctx.opaque = make(chan bool)

	return tag, ctx, false
}

func bytestreamCalculateDestinationAddress(tag data.BytestreamQuery, stanza *data.ClientIQ) string {
	if tag.DestinationAddress != "" {
		return tag.DestinationAddress
	}
	return hex.EncodeToString(digests.Sha1([]byte(tag.Sid + stanza.From + stanza.To)))
}

func (ctx *recvContext) bytestreamDoReceive(conn io.ReadWriteCloser) {
	recv := ctx.createReceiver()
	cancel := ctx.opaque.(chan bool)
	go func() {
		c, ok := <-cancel
		if c && ok {
			recv.cancel()
		}
	}()

	_, err := ioCopy(recv, conn)
	if err != nil && err != errLocalCancel {
		closeAndIgnore(conn)
		return
	}

	toSend, fname, ok, _ := recv.wait()
	if !ok {
		closeAndIgnore(conn)
		return
	}

	if toSend != nil {
		_, _ = conn.Write(toSend)
	}

	closeAndIgnore(conn)

	if err := ctx.finalizeFileTransfer(fname); err != nil {
		ctx.s.Log().WithError(err).Warn("Had error when trying to move the final file")
		ctx.control.ReportError(errors.New("Couldn't move the final file"))
		_ = os.Remove(fname)
	}
}

// BytestreamQuery is the hook function that will be called when we receive a bytestream query IQ
func BytestreamQuery(s canSendIQErrorHasConfigAndHasLog, stanza *data.ClientIQ) (ret interface{}, iqtype string, ignore bool) {
	tag, ctx, earlyReturn := bytestreamInitialSetup(s, stanza)
	if earlyReturn {
		return nil, "", true
	}

	dstAddr := bytestreamCalculateDestinationAddress(tag, stanza)

	k := func(c io.ReadWriteCloser) {
		go ctx.bytestreamDoReceive(c)
	}

	for _, sh := range tag.Streamhosts {
		if tryStreamhost(s, sh, dstAddr, k) {
			return data.BytestreamQuery{
				Sid:            tag.Sid,
				StreamhostUsed: &data.BytestreamStreamhostUsed{Jid: sh.Jid},
			}, "result", false
		}
	}

	s.SendIQError(stanza, iqErrorItemNotFound)
	return nil, "", true
}

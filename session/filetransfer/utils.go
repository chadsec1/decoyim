package filetransfer

import (
	"bytes"
	"encoding/xml"
	"errors"

	"github.com/chadsec1/decoyim/config"
	"github.com/chadsec1/decoyim/decoylog"
	"github.com/chadsec1/decoyim/session/access"
	"github.com/chadsec1/decoyim/xmpp/data"
	xi "github.com/chadsec1/decoyim/xmpp/interfaces"
)

var errChannelClosed = errors.New("channel closed")
var errNotResultIQ = errors.New("expected result IQ")
var errNotClientIQ = errors.New("expected Client IQ")

type hasConnectionAndConfig interface {
	xi.Has
	config.Has
}

type hasConnectionAndConfigAndLog interface {
	hasConnectionAndConfig
	decoylog.Has
}

type hasConfigAndLog interface {
	config.Has
	decoylog.Has
}

type canSendIQAndHasLog interface {
	access.CanSendIQ
	decoylog.Has
}

type canSendIQAndHasLogAndConnection interface {
	canSendIQAndHasLog
	xi.Has
}

type canSendIQErrorAndHasLog interface {
	decoylog.Has
	access.CanSendIQ
}

type canSendIQErrorHasConfigAndHasLog interface {
	hasConfigAndLog
	access.CanSendIQ
}

type hasConnectionAndConfigAndLogAndHasSymmetricKey interface {
	hasConnectionAndConfigAndLog
	access.HasSymmetricKey
}

type hasLogConnectionIQSymmetricKeyAndIsPublisher interface {
	canSendIQAndHasLogAndConnection
	access.HasSymmetricKey
	access.Publisher
}

func basicIQ(s xi.Has, to, tp string, toSend, unpackInto interface{}, onSuccess func(*data.ClientIQ)) error {
	done := make(chan error, 1)

	nonblockIQ(s, to, tp, toSend, unpackInto, func(ciq *data.ClientIQ) {
		onSuccess(ciq)
		done <- nil
	}, func(_ *data.ClientIQ, ee error) {
		done <- ee
	})

	return <-done
}

func nonblockIQ(s xi.Has, to, tp string, toSend, unpackInto interface{}, onSuccess func(*data.ClientIQ), onError func(*data.ClientIQ, error)) {
	rp, _, err := s.Conn().SendIQ(to, tp, toSend)
	if err != nil {
		onError(nil, err)
		return
	}
	go func() {
		r, ok := <-rp
		if !ok {
			onError(nil, errChannelClosed)
			return
		}

		switch ciq := r.Value.(type) {
		case *data.ClientIQ:
			if ciq.Type != "result" {
				onError(ciq, errNotResultIQ)
				return
			}
			if unpackInto != nil {
				if err := xml.NewDecoder(bytes.NewBuffer(ciq.Query)).Decode(unpackInto); err != nil {
					onError(ciq, err)
					return
				}
			}

			onSuccess(ciq)
			return
		}
		onError(nil, errNotClientIQ)
	}()
}

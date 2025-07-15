package interfaces

import (
	"github.com/chadsec1/decoyim/coylog"
	"github.com/chadsec1/decoyim/servers"
	"github.com/chadsec1/decoyim/tls"
	"github.com/chadsec1/decoyim/xmpp/data"
	"golang.org/x/net/proxy"
)

// Dialer connects and authenticates to an XMPP server
type Dialer interface {
	Config() data.Config
	Dial() (Conn, error)
	GetServer() string
	RegisterAccount(data.FormCallback) (Conn, error)
	ServerAddress() string
	SetConfig(data.Config)
	SetJID(string)
	SetPassword(string)
	SetProxy(proxy.Dialer)
	SetResource(string)
	SetServerAddress(v string)
	SetShouldConnectTLS(bool)
	SetShouldSendALPN(bool)
	SetLogger(coylog.Logger)
	SetKnown(*servers.Server)
}

// DialerFactory represents a function that can create a Dialer
type DialerFactory func(tls.Verifier, tls.Factory) Dialer

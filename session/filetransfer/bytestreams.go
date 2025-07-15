package filetransfer

import (
	"io"
	"net"
	"strconv"

	"github.com/chadsec1/decoyim/config"
	"github.com/chadsec1/decoyim/xmpp/data"
	"github.com/chadsec1/decoyim/xmpp/socks5"
	"golang.org/x/net/proxy"
)

var createTorProxy = func(a *config.Account) (proxy.Dialer, error) {
	return a.CreateTorProxy()
}

var socks5XMPP = socks5.XMPP

func tryStreamhost(s hasConfigAndLog, sh data.BytestreamStreamhost, dstAddr string, k func(io.ReadWriteCloser)) bool {
	port := sh.Port
	if port == 0 {
		port = 1080
	}

	p, err := createTorProxy(s.GetConfig())
	if err != nil {
		s.Log().WithError(err).Warn("Had error when trying to connect")
		return false
	}

	if p == nil {
		p = proxy.Direct
	}

	dialer, e := socks5XMPP("tcp", net.JoinHostPort(sh.Host, strconv.Itoa(port)), nil, p)
	if e != nil {
		s.Log().WithError(e).WithField("streamhost", sh).Info("Error setting up socks5")
		return false
	}

	conn, e2 := dialer.Dial("tcp", net.JoinHostPort(dstAddr, "0"))
	if e2 != nil {
		s.Log().WithError(e2).WithField("streamhost", sh).Info("Error connecting socks5")
		return false
	}

	k(conn)
	return true
}

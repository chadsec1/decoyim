package xmpp

import (
	"encoding/hex"
	"io"
	"io/ioutil"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"

	"github.com/chadsec1/decoyim/coylog"
	"github.com/chadsec1/decoyim/i18n"
	ourNet "github.com/chadsec1/decoyim/net"
	"github.com/chadsec1/decoyim/xmpp/data"
	"github.com/chadsec1/decoyim/xmpp/errors"
	"github.com/coyim/gotk3adapter/glib_mock"

	. "gopkg.in/check.v1"
)

type TCPSuite struct{}

func (*TCPSuite) SetUpSuite(c *C) {
	log.SetOutput(ioutil.Discard)
	i18n.InitLocalization(&glib_mock.Mock{})
}

var _ = Suite(&TCPSuite{})

func (s *TCPSuite) Test_newTCPConn_SkipsSRVAndConnectsToOriginDomain(c *C) {
	p := &mockProxy{}
	d := &dialer{
		JID: "foo@jabber.com",

		proxy: p,
		config: data.Config{
			SkipSRVLookup: true,
		},
		log: testLogger(),
	}

	expectedConn := &net.TCPConn{}
	p.Expects(func(network, addr string) (net.Conn, error) {
		c.Check(network, Equals, "tcp")
		c.Check(addr, Equals, "jabber.com:5222")

		return expectedConn, nil
	})

	conn, _, err := d.newTCPConn()
	c.Check(err, IsNil)
	c.Check(conn, Equals, expectedConn)

	c.Check(p, MatchesExpectations)
}

func (s *TCPSuite) Test_newTCPConn_SkipsSRVAndConnectsToConfiguredServerAddress(c *C) {
	p := &mockProxy{}
	d := &dialer{
		JID:           "foo@jabber.com",
		serverAddress: "jabber.im:5333",

		proxy: p,
		log:   testLogger(),
	}

	expectedConn := &net.TCPConn{}
	p.Expects(func(network, addr string) (net.Conn, error) {
		c.Check(network, Equals, "tcp")
		c.Check(addr, Equals, "jabber.im:5333")

		return expectedConn, nil
	})

	conn, _, err := d.newTCPConn()
	c.Check(err, IsNil)
	c.Check(conn, Equals, expectedConn)
	c.Check(d.config.SkipSRVLookup, Equals, true)

	c.Check(p, MatchesExpectations)
}

func (s *TCPSuite) Test_newTCPConn_ErrorsIfServiceIsNotAvailable(c *C) {
	p := &mockProxy{}
	d := &dialer{
		JID: "foo@jabber.com",

		proxy: p,
		log:   testLogger(),
	}

	// We exploit resolveSRVWithProxy forwarding conn errors
	// to fake an error it should generated.
	p.Expects(func(network, addr string) (net.Conn, error) {
		c.Check(network, Equals, "tcp")
		c.Check(addr, Equals, "208.67.222.222:53")

		return nil, ErrServiceNotAvailable
	})

	_, _, err := d.newTCPConn()
	c.Check(err, Equals, ErrServiceNotAvailable)

	c.Check(p, MatchesExpectations)
}

func (s *TCPSuite) Test_newTCPConn_usesDirectProxyIfNoneGiven(c *C) {
	orgsrvLookupAndFallback := srvLookupAndFallback
	defer func() {
		srvLookupAndFallback = orgsrvLookupAndFallback
	}()

	srvLookupAndFallback = func(*dialer) (net.Conn, bool, error) {
		return nil, false, nil
	}

	d := &dialer{
		proxy: nil,
	}

	_, _, _ = d.newTCPConn()

	c.Assert(d.proxy, Equals, proxy.Direct)
}

func testLogger() coylog.Logger {
	l := log.New()
	l.SetOutput(ioutil.Discard)
	return l
}

func (s *TCPSuite) Test_newTCPConn_DefaultsToOriginDomainAtDefaultPortAfterSRVFails(c *C) {
	p := &mockProxy{}
	d := &dialer{
		JID: "foo@jabber.com",

		proxy: p,
		log:   testLogger(),
	}

	// Connection for lookup of xmpps-client SRV record
	p.Expects(func(network, addr string) (net.Conn, error) {
		c.Check(network, Equals, "tcp")
		c.Check(addr, Equals, "208.67.222.222:53")

		return nil, io.EOF
	})

	// Connection for lookup of xmpp-client SRV record
	p.Expects(func(network, addr string) (net.Conn, error) {
		c.Check(network, Equals, "tcp")
		c.Check(addr, Equals, "208.67.222.222:53")

		return nil, io.EOF
	})

	expectedConn := &net.TCPConn{}
	p.Expects(func(network, addr string) (net.Conn, error) {
		c.Check(network, Equals, "tcp")
		c.Check(addr, Equals, "jabber.com:5222")

		return expectedConn, nil
	})

	conn, _, err := d.newTCPConn()
	c.Check(p.called, Equals, 3)
	c.Check(err, IsNil)
	c.Check(conn, Equals, expectedConn)

	c.Check(p, MatchesExpectations)
}

func (s *TCPSuite) Test_newTCPConn_ErrorsWhenTCPBindingFails(c *C) {
	p := &mockProxy{}
	d := &dialer{
		JID: "foo@jabber.com",

		proxy: p,
		log:   testLogger(),
	}

	// Connection for lookup of xmpps-client SRV record
	p.Expects(func(network, addr string) (net.Conn, error) {
		c.Check(network, Equals, "tcp")
		c.Check(addr, Equals, "208.67.222.222:53")

		return nil, io.EOF
	})

	// Connection for lookup of xmpp-client SRV record
	p.Expects(func(network, addr string) (net.Conn, error) {
		c.Check(network, Equals, "tcp")
		c.Check(addr, Equals, "208.67.222.222:53")

		return nil, io.EOF
	})

	p.Expects(func(network, addr string) (net.Conn, error) {
		c.Check(network, Equals, "tcp")
		c.Check(addr, Equals, "jabber.com:5222")

		return nil, io.EOF
	})

	_, _, err := d.newTCPConn()
	c.Check(p.called, Equals, 3)
	c.Check(err, Equals, errors.ErrTCPBindingFailed)

	c.Check(p, MatchesExpectations)
}

func (s *TCPSuite) Test_newTCPConn_ErrorsWhenTCPBindingSucceedsButConnectionFails(c *C) {
	dec, _ := hex.DecodeString("00511eea818000010001000000000c5f786d70702d636c69656e74045f746370076f6c6162696e690273650000210001c00c0021000100000258001700000005146604786d7070076f6c6162696e6902736500")

	p := &mockProxy{}
	d := &dialer{
		JID: "foo@olabini.se",

		proxy: p,
		log:   testLogger(),
	}

	p.Expects(func(network, addr string) (net.Conn, error) {
		c.Check(network, Equals, "tcp")
		c.Check(addr, Equals, "208.67.222.222:53")

		return fakeTCPConnToDNS(dec, 49)
	})

	p.Expects(func(network, addr string) (net.Conn, error) {
		c.Check(network, Equals, "tcp")
		c.Check(addr, Equals, "208.67.222.222:53")

		return fakeTCPConnToDNS(dec, 48)
	})

	p.Expects(func(network, addr string) (net.Conn, error) {
		c.Check(network, Equals, "tcp")
		c.Check(addr, Equals, "xmpp.olabini.se:5222")

		return nil, io.EOF
	})

	_, _, err := d.newTCPConn()

	c.Check(p.called, Equals, 3)
	c.Check(err, Equals, errors.ErrConnectionFailed)
	c.Check(p, MatchesExpectations)
}

type funcDialer struct {
	f func(string, string) (net.Conn, error)
}

func (fd *funcDialer) Dial(network, addr string) (net.Conn, error) {
	return fd.f(network, addr)
}

func (s *TCPSuite) Test_dialer_connectWithProxy_timesOut(c *C) {
	orgDefaultDialTimeout := defaultDialTimeout
	defer func() {
		defaultDialTimeout = orgDefaultDialTimeout
	}()

	defaultDialTimeout = 1 * time.Millisecond

	done := make(chan bool, 1)
	p := &funcDialer{
		f: func(string, string) (net.Conn, error) {
			<-done
			return nil, nil
		},
	}
	d := &dialer{
		JID:           "foo@jabber.com",
		serverAddress: "jabber.im:5333",

		proxy: p,
		log:   testLogger(),
	}

	_, _, e := d.connectWithProxy(&connectEntry{}, p)
	done <- true
	c.Assert(e, Equals, ourNet.ErrTimeout)
}

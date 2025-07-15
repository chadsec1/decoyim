// Package xmpp implements the XMPP IM protocol, as specified in RFC 6120 and
// 6121.
package xmpp

import (
	"fmt"

	"github.com/chadsec1/decoyim/xmpp/data"
)

// RequestVCard requests the user's vcard from the server. It returns a
// channel on which the reply can be read when received and a Cookie that can
// be used to cancel the request.
func (c *conn) RequestVCard() (<-chan data.Stanza, data.Cookie, error) {
	c.ioLock.Lock()
	defer c.ioLock.Unlock()

	cookie := c.getCookie()
	if _, err := fmt.Fprintf(c.out, "<iq type='get' id='%x'><vCard xmlns='vcard-temp'/></iq>", cookie); err != nil {
		return nil, 0, err
	}

	return c.createInflight(cookie, "")
}

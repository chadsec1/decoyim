// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package xmpp implements the XMPP IM protocol, as specified in RFC 6120 and
// 6121.
package xmpp

import (
	"bytes"
	"encoding/xml"
	"fmt"

	"github.com/chadsec1/decoyim/xmpp/data"
)

type rawXML []byte

// SendIQ sends an info/query message to the given user. It returns a channel
// on which the reply can be read when received and a Cookie that can be used
// to cancel the request.
func (c *conn) SendIQ(to, typ string, value interface{}) (reply <-chan data.Stanza, cookie data.Cookie, err error) {
	var outb bytes.Buffer
	out := &outb

	nextCookie := c.getCookie()
	toAttr := ""
	if len(to) > 0 {
		toAttr = "to='" + xmlEscape(to) + "' "
	}

	_, _ = fmt.Fprintf(out, "<iq xmlns='jabber:client' %sfrom='%s' type='%s' id='%x'>", toAttr, xmlEscape(c.jid), xmlEscape(typ), nextCookie)

	switch v := value.(type) {
	case data.EmptyReply:
		//nothing
	case rawXML:
		_, err = out.Write(v)
	default:
		err = xml.NewEncoder(out).Encode(value)
	}

	if err != nil {
		return
	}

	_, _ = fmt.Fprintf(out, "</iq>")

	_, err = c.safeWrite(outb.Bytes())
	if err != nil {
		return
	}

	return c.createInflight(nextCookie, to)
}

// SendIQReply sends a reply to an IQ query.
func (c *conn) SendIQReply(to, typ, id string, value interface{}) error {
	var outb bytes.Buffer
	out := &outb

	toAttr := ""
	if len(to) > 0 {
		toAttr = "to='" + xmlEscape(to) + "' "
	}

	_, _ = fmt.Fprintf(out, "<iq xmlns='jabber:client' %sfrom='%s' type='%s' id='%s'>", toAttr, xmlEscape(c.jid), xmlEscape(typ), xmlEscape(id))

	if _, ok := value.(data.EmptyReply); !ok {
		if err := xml.NewEncoder(out).Encode(value); err != nil {
			return err
		}
	}
	_, _ = fmt.Fprintf(out, "</iq>")

	_, err := c.safeWrite(outb.Bytes())
	return err
}

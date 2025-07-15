// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package xmpp implements the XMPP IM protocol, as specified in RFC 6120 and
// 6121.
package xmpp

import (
	"encoding/binary"
	"io"

	"github.com/chadsec1/decoyim/xmpp/data"
)

func (c *conn) getCookie() data.Cookie {
	var buf [8]byte
	if _, err := io.ReadFull(c.Rand(), buf[:]); err != nil {
		panic("Failed to read random bytes: " + err.Error())
	}
	return data.Cookie(binary.LittleEndian.Uint64(buf[:]))
}

func (c *conn) allInflightCookies() []data.Cookie {
	c.inflightsMutex.Lock()
	defer c.inflightsMutex.Unlock()

	allCookies := []data.Cookie{}
	for cookie := range c.inflights {
		allCookies = append(allCookies, cookie)
	}

	return allCookies
}

func (c *conn) cancelInflights() {
	for _, cookie := range c.allInflightCookies() {
		c.Cancel(cookie)
	}
}

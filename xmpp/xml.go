// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package xmpp implements the XMPP IM protocol, as specified in RFC 6120 and
// 6121.
package xmpp

import (
	"bytes"
	"encoding/xml"
	"errors"
	"reflect"
	"strings"
	"sync"

	"github.com/chadsec1/decoyim/coylog"
	"github.com/chadsec1/decoyim/xmpp/data"
	log "github.com/sirupsen/logrus"
)

var xmlSpecial = map[byte]string{
	'<':  "&lt;",
	'>':  "&gt;",
	'"':  "&quot;",
	'\'': "&apos;",
	'&':  "&amp;",
}

// xmlConn is a simplified subset of the Conn interface
// that only exposes the functionality that XML needs
type xmlConn interface {
	In() *xml.Decoder
	Lock() *sync.Mutex
	CustomStorage() map[xml.Name]reflect.Type
}

func xmlEscape(s string) string {
	//TODO: Why not using xml.EscapeText(), from stdlib?
	var b bytes.Buffer
	for i := 0; i < len(s); i++ {
		c := s[i]
		if s, ok := xmlSpecial[c]; ok {
			_, _ = b.WriteString(s)
		} else {
			_ = b.WriteByte(c)
		}
	}
	return b.String()
}

// Scan XML token stream to find next Element (start or end)
func nextElement(p *xml.Decoder, ll coylog.Logger) (xml.Token, error) {
	for {
		t, err := p.Token()
		if err != nil {
			return xml.StartElement{}, err
		}

		switch elem := t.(type) {
		case xml.StartElement, xml.EndElement:
			return t, nil
		case xml.CharData:
			// https://xmpp.org/rfcs/rfc6120.html#xml-whitespace
			// rfc6120, section 1.4: "whitespace" is used to refer to any character
			// or characters matching [...] SP, HTAB, CR, or LF.
			switch string(elem) {
			case " ", "\t", "\r", "\n": //TODO: consider more than one whitespace
				ll.Info("xmpp: received whitespace ping")
			}
		case xml.ProcInst:
			if !(elem.Target == "xml" && strings.HasPrefix(string(elem.Inst), "version=")) {
				ll.WithFields(log.Fields{"target": elem.Target, "inst": elem.Inst}).Warn("xmpp: received unhandled ProcInst element")
			}
		default:
			ll.WithField("element", elem).Warn("xmpp: received unhandled element")
		}
	}
}

// Scan XML token stream to find next StartElement.
func nextStart(p *xml.Decoder, log coylog.Logger) (xml.StartElement, error) {
	for {
		t, err := nextElement(p, log)
		if err != nil {
			return xml.StartElement{}, err
		}

		if start, ok := t.(xml.StartElement); ok {
			return start, nil
		}
	}
}

// Scan XML token stream for next element and save into val.
// If val == nil, allocate new element based on proto map.
// Either way, return val.
func next(c xmlConn, log coylog.Logger) (xml.Name, interface{}, error) {
	elem, err := nextElement(c.In(), log)
	if err != nil {
		return xml.Name{}, nil, err
	}

	c.Lock().Lock()
	defer c.Lock().Unlock()

	if el, ok := elem.(xml.StartElement); ok {
		return decodeStartElement(c, el)
	}

	return decodeEndElement(elem.(xml.EndElement))
}

func decodeStartElement(c xmlConn, se xml.StartElement) (xml.Name, interface{}, error) {
	// Put it in an interface and allocate one.
	var nv interface{}
	if t, e := c.CustomStorage()[se.Name]; e {
		nv = reflect.New(t).Interface()
	} else if t, e := defaultStorage[se.Name]; e {
		nv = reflect.New(t).Interface()
	} else {
		return xml.Name{}, nil, errors.New("unexpected XMPP message " +
			se.Name.Space + " <" + se.Name.Local + "/>")
	}

	// Unmarshal into that storage.
	if err := c.In().DecodeElement(nv, &se); err != nil {
		return xml.Name{}, nil, err
	}

	return se.Name, nv, nil
}

func decodeEndElement(ce xml.EndElement) (xml.Name, interface{}, error) {
	switch ce.Name {
	case xml.Name{Space: NsStream, Local: "stream"}:
		return ce.Name, &data.StreamClose{}, nil
	}

	return ce.Name, nil, nil
}

var defaultStorage = map[xml.Name]reflect.Type{
	{Space: NsStream, Local: "features"}: reflect.TypeOf(data.StreamFeatures{}),
	{Space: NsStream, Local: "error"}:    reflect.TypeOf(data.StreamError{}),
	{Space: NsTLS, Local: "starttls"}:    reflect.TypeOf(data.StartTLS{}),
	{Space: NsTLS, Local: "proceed"}:     reflect.TypeOf(data.ProceedTLS{}),
	{Space: NsTLS, Local: "failure"}:     reflect.TypeOf(data.FailureTLS{}),
	{Space: NsSASL, Local: "mechanisms"}: reflect.TypeOf(data.SaslMechanisms{}),
	{Space: NsSASL, Local: "challenge"}:  reflect.TypeOf(""),
	{Space: NsSASL, Local: "response"}:   reflect.TypeOf(""),
	{Space: NsSASL, Local: "abort"}:      reflect.TypeOf(data.SaslAbort{}),
	{Space: NsSASL, Local: "success"}:    reflect.TypeOf(data.SaslSuccess{}),
	{Space: NsSASL, Local: "failure"}:    reflect.TypeOf(data.SaslFailure{}),
	{Space: NsBind, Local: "bind"}:       reflect.TypeOf(data.BindBind{}),
	{Space: NsClient, Local: "message"}:  reflect.TypeOf(data.ClientMessage{}),
	{Space: NsClient, Local: "presence"}: reflect.TypeOf(data.ClientPresence{}),
	{Space: NsClient, Local: "iq"}:       reflect.TypeOf(data.ClientIQ{}),
	{Space: NsClient, Local: "error"}:    reflect.TypeOf(data.StanzaError{}),
}

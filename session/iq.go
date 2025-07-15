package session

import (
	"bytes"
	"fmt"

	"github.com/chadsec1/decoyim/session/access"
	"github.com/chadsec1/decoyim/xmpp/data"
)

func (s *session) sendIQError(stanza *data.ClientIQ, reply interface{}) {
	s.sendIQReply(stanza, "error", reply)
}

func (s *session) SendIQError(stanza *data.ClientIQ, reply interface{}) {
	s.sendIQError(stanza, reply)
}

func (s *session) sendIQResult(stanza *data.ClientIQ, reply interface{}) {
	s.sendIQReply(stanza, "result", reply)
}

func (s *session) SendIQResult(stanza *data.ClientIQ, reply interface{}) {
	s.sendIQResult(stanza, reply)
}

func (s *session) sendIQReply(stanza *data.ClientIQ, tp string, reply interface{}) {
	if err := s.conn.SendIQReply(stanza.From, tp, stanza.ID, reply); err != nil {
		s.log.WithError(err).Error("Failed to send IQ message")
	}
}

func versionIQ(s access.Session, _ *data.ClientIQ) (ret interface{}, iqtype string, ignore bool) {
	s.Log().Info("IQ: jabber:iq:version query")
	return s.(*session).receivedIQVersion(), "", false
}

func rosterIQ(s access.Session, stanza *data.ClientIQ) (ret interface{}, iqtype string, ignore bool) {
	s.Log().Info("IQ: jabber:iq:roster query")
	return s.(*session).receivedIQRosterQuery(stanza)
}

func unknownIQ(s access.Session, stanza *data.ClientIQ) (ret interface{}, iqtype string, ignore bool) {
	s.Log().Info(fmt.Sprintf("Unknown IQ: %s", bytes.NewBuffer(stanza.Query)))
	return nil, "", false
}

type iqFunction func(access.Session, *data.ClientIQ) (interface{}, string, bool)

var knownIQs = map[string]iqFunction{}

func registerKnownIQ(stanzaType, fullName string, f iqFunction) {
	knownIQs[stanzaType+" "+fullName] = f
}

func getIQHandler(stanzaType, namespace, local string) iqFunction {
	f, ok := knownIQs[fmt.Sprintf("%s %s %s", stanzaType, namespace, local)]
	if ok {
		return f
	}
	return unknownIQ
}

func init() {
	registerKnownIQ("get", "http://jabber.org/protocol/disco#items query", discoItemsIQ)
	registerKnownIQ("get", "http://jabber.org/protocol/disco#info query", discoIQ)
	registerKnownIQ("get", "jabber:iq:version query", versionIQ)
	registerKnownIQ("set", "jabber:iq:roster query", rosterIQ)
	registerKnownIQ("result", "jabber:iq:roster query", rosterIQ)
}

func (s *session) processIQ(stanza *data.ClientIQ) (ret interface{}, iqtype string, ignore bool) {
	if nspace, local, ok := tryDecodeXML(stanza.Query); ok {
		return getIQHandler(stanza.Type, nspace, local)(s, stanza)
	}
	return nil, "", false
}

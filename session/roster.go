package session

import "github.com/chadsec1/decoyim/xmpp/data"

// RemoveContact is used to remove a contact
func (s *session) RemoveContact(jid string) {
	_, _, _ = s.conn.SendIQ("" /* to the server */, "set", data.RosterRequest{
		Item: data.RosterRequestItem{
			Jid:          jid,
			Subscription: "remove",
		},
	})
}

package session

import (
	"errors"

	"github.com/chadsec1/decoyim/session/muc/data"
	xmppData "github.com/chadsec1/decoyim/xmpp/data"
	"github.com/chadsec1/decoyim/xmpp/jid"
	log "github.com/sirupsen/logrus"
)

var (
	// ErrNotAllowedKickOccupant represents error occurred when an occupant tries kick out to other with a higher affiliation
	ErrNotAllowedKickOccupant = errors.New("can't kick out an occupant with higher affiliation than the self-occupant")
)

func (s *session) UpdateOccupantRole(roomID jid.Bare, occupantNickname string, role data.Role, reason string) (<-chan bool, <-chan error) {
	return s.muc.updateOccupantRole(roomID, occupantNickname, role, reason)
}

func (m *mucManager) updateOccupantRole(roomID jid.Bare, occupantNickname string, role data.Role, reason string) (<-chan bool, <-chan error) {
	l := m.log.WithFields(log.Fields{
		"room": roomID,
		"nick": occupantNickname,
		"role": role.Name(),
	})

	rc := make(chan bool)
	ec := make(chan error)

	go func() {
		reply, _, err := m.conn().SendIQ(roomID.String(), "set", &xmppData.MUCAdmin{
			Items: []xmppData.MUCItem{
				{
					Nick:   occupantNickname,
					Role:   role.Name(),
					Reason: reason,
				},
			},
		})

		if err != nil {
			l.WithError(err).Error("An error occurred when updating the occupant role")
			ec <- ErrUpdateOccupantRequest
			return
		}

		err = validateIqResponse(reply)
		if err != nil {
			l.WithError(err).Error("An error occurred when trying to read the response from the room configuration rollback request")
			ec <- err
			return
		}

		rc <- true
	}()

	return rc, ec
}

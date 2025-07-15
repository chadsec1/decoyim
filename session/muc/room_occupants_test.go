package muc

import (
	"github.com/chadsec1/decoyim/session/muc/data"
	"github.com/chadsec1/decoyim/xmpp/jid"
	. "gopkg.in/check.v1"
)

type RoomOccupantsSuite struct{}

var _ = Suite(&RoomOccupantsSuite{})

func (*RoomOccupantsSuite) Test_RoomOccupants_ChangeAffiliationToNone(c *C) {
	roi := &RoomOccupantItem{
		Jid:         jid.Parse("batman@cave.org"),
		Affiliation: &data.OwnerAffiliation{},
		Reason:      "no reason",
	}

	c.Assert(roi.Affiliation.IsOwner(), Equals, true)
	roi.ChangeAffiliationToNone()
	c.Assert(roi.Affiliation.IsNone(), Equals, true)
}

func (*RoomOccupantsSuite) Test_RoomOccupants_IncludesJid(c *C) {

	roil := &RoomOccupantItemList{
		&RoomOccupantItem{
			Jid:         jid.Parse("batman@cave.org"),
			Affiliation: &data.OwnerAffiliation{},
			Reason:      "no reason",
		}}

	c.Assert(roil.IncludesJid(jid.Parse("batman@cave.org")), Equals, true)
	c.Assert(roil.IncludesJid(jid.Parse("odd@one.out")), Equals, false)
}

func (s *MucRoomConfigSuite) Test_RoomConfigForm_retrieveOccupantsToUpdate(c *C) {
	occupantsList := RoomOccupantItemList{
		&RoomOccupantItem{
			Jid:         jid.Parse("batman@cave.org"),
			Affiliation: &data.OwnerAffiliation{},
			Reason:      "no reason",
		},
		&RoomOccupantItem{
			Jid:           jid.Parse("super@man.org"),
			Affiliation:   &data.OwnerAffiliation{},
			Reason:        "no reason",
			MustBeUpdated: true,
		},
	}
	c.Assert(occupantsList.RetrieveOccupantsToUpdate(), HasLen, 1)
}

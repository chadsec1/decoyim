package muc

import (
	"github.com/chadsec1/decoyim/session/muc/data"
	"github.com/chadsec1/decoyim/xmpp/jid"
)

// RoomOccupantItem contains information related with occupants to be configured in a room with a specific affiliation
type RoomOccupantItem struct {
	Jid           jid.Any
	Affiliation   data.Affiliation
	Reason        string
	MustBeUpdated bool
}

// ChangeAffiliationToNone changes an occupant's affiliation to none
func (roi *RoomOccupantItem) ChangeAffiliationToNone() {
	roi.Affiliation = &data.NoneAffiliation{}
}

// RoomOccupantItemList represents a list of room occupant items
type RoomOccupantItemList []*RoomOccupantItem

// IncludesJid returns a boolean that indicates if the given account ID (jid) is in the list
func (l RoomOccupantItemList) IncludesJid(id jid.Any) bool {
	for _, itm := range l {
		if itm.Jid.String() == id.String() {
			return true
		}
	}
	return false
}

// RetrieveOccupantsToUpdate returns a list of occupants to be updated
func (l RoomOccupantItemList) RetrieveOccupantsToUpdate() RoomOccupantItemList {
	extracted := RoomOccupantItemList{}
	for _, o := range l {
		if o.MustBeUpdated {
			extracted = append(extracted, o)
		}
	}
	return extracted
}

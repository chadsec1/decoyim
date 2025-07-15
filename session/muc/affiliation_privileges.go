package muc

import (
	"github.com/chadsec1/decoyim/session/muc/data"
)

type affiliationNumberType int

const (
	affiliationTypeOutcast affiliationNumberType = iota
	affiliationTypeNone
	affiliationTypeMember
	affiliationTypeAdmin
	affiliationTypeOwner
)

func affiliationNumberTypeFrom(a data.Affiliation) affiliationNumberType {
	switch {
	case a.IsOwner():
		return affiliationTypeOwner
	case a.IsAdmin():
		return affiliationTypeAdmin
	case a.IsMember():
		return affiliationTypeMember
	case a.IsNone():
		return affiliationTypeNone
	}
	return affiliationTypeOutcast
}

const (
	enterOpenRoom privilege = iota
	registerWithOpenRoom
	retrieveMemberList
	enterMembersOnlyRoom
	banMembersAndUnaffiliatedUsers
	editMemberList
	assignAndRemoveModeratorRole
	editAdminList
	editOwnerList
	changeRoomConfiguration
	destroyRoom
)

var affiliationPrivileges = [][]bool{
	{false /*outcast*/, true /*none*/, true /*member*/, true /*administrator*/, true /*owner*/},    //enterOpenRoom
	{false /*outcast*/, true /*none*/, false /*member*/, false /*administrator*/, false /*owner*/}, //registerWithOpenRoom
	{false /*outcast*/, false /*none*/, true /*member*/, true /*administrator*/, true /*owner*/},   //retrieveMemberList
	{false /*outcast*/, false /*none*/, true /*member*/, true /*administrator*/, true /*owner*/},   //enterMembersOnlyRoom
	{false /*outcast*/, false /*none*/, false /*member*/, true /*administrator*/, true /*owner*/},  //banMembersAndUnaffiliatedUsers
	{false /*outcast*/, false /*none*/, false /*member*/, true /*administrator*/, true /*owner*/},  //editMemberList
	{false /*outcast*/, false /*none*/, false /*member*/, true /*administrator*/, true /*owner*/},  //assignAndRemoveModeratorRole
	{false /*outcast*/, false /*none*/, false /*member*/, false /*administrator*/, true /*owner*/}, //editAdminList
	{false /*outcast*/, false /*none*/, false /*member*/, false /*administrator*/, true /*owner*/}, //editOwnerList
	{false /*outcast*/, false /*none*/, false /*member*/, false /*administrator*/, true /*owner*/}, //changeRoomConfiguration
	{false /*outcast*/, false /*none*/, false /*member*/, false /*administrator*/, true /*owner*/}, //destroyRoom
}

func (o *Occupant) affiliationHasPrivilege(p privilege) bool {
	return affiliationPrivileges[p][affiliationNumberTypeFrom(o.Affiliation)]
}

// CanEnterOpenRoom returns true if the occupant can enter to an open room
// based on the occupant's affiliation
func (o *Occupant) CanEnterOpenRoom() bool {
	return o.affiliationHasPrivilege(enterOpenRoom)
}

// CanRegisterWithOpenRoom returns true if the occupant can register with open room
// based on the occupant's affiliation
func (o *Occupant) CanRegisterWithOpenRoom() bool {
	return o.affiliationHasPrivilege(registerWithOpenRoom)
}

// CanRetrieveMemberList returns true if the occupant can retrieve the members list
// based on the occupant's affiliation
func (o *Occupant) CanRetrieveMemberList() bool {
	return o.affiliationHasPrivilege(retrieveMemberList)
}

// CanEnterMembersOnlyRoom returns true if the occupant can enter to a members only room
// based on the occupant's affiliation
func (o *Occupant) CanEnterMembersOnlyRoom() bool {
	return o.affiliationHasPrivilege(enterMembersOnlyRoom)
}

// CanBanMembersAndUnaffiliatedUsers returns true if the occupant can ban members and unaffiliated users
// based on the occupant's affiliation
func (o *Occupant) CanBanMembersAndUnaffiliatedUsers() bool {
	return o.affiliationHasPrivilege(banMembersAndUnaffiliatedUsers)
}

// CanEditMemberList returns true if the occupant can edit the members list
// based on the occupant's affiliation
func (o *Occupant) CanEditMemberList() bool {
	return o.affiliationHasPrivilege(editMemberList)
}

// CanAssignAndRemoveModeratorRole returns true if the occupant can assign and remove moderator role
// based on the occupant's affiliation
func (o *Occupant) CanAssignAndRemoveModeratorRole() bool {
	return o.affiliationHasPrivilege(assignAndRemoveModeratorRole)
}

// CanEditAdminList returns true if the occupant can edit the admin list
// based on the occupant's affiliation
func (o *Occupant) CanEditAdminList() bool {
	return o.affiliationHasPrivilege(editAdminList)
}

// CanEditOwnerList returns true if the occupant can edit the owners list
// based on the occupant's affiliation
func (o *Occupant) CanEditOwnerList() bool {
	return o.affiliationHasPrivilege(editOwnerList)
}

// CanChangeRoomConfiguration returns true if the occupant can change the room configuration
// based on the occupant's affiliation
func (o *Occupant) CanChangeRoomConfiguration() bool {
	return o.affiliationHasPrivilege(changeRoomConfiguration)
}

// CanDestroyRoom returns true if the occupant can destroy the room
// based on the occupant's affiliation
func (o *Occupant) CanDestroyRoom() bool {
	return o.affiliationHasPrivilege(destroyRoom)
}

// CanChangeAffiliation returns a boolean indicating if the occupant can change the affiliation of the
// given occupant based on the occupant's affiliation
func (o *Occupant) CanChangeAffiliation(oc *Occupant) bool {
	return o.isOwner() || (o.isAdmin() && !oc.isOwnerOrAdmin())
}

func (o *Occupant) isOwner() bool {
	return o.Affiliation.IsOwner()
}

func (o *Occupant) isAdmin() bool {
	return o.Affiliation.IsAdmin()
}

func (o *Occupant) isOwnerOrAdmin() bool {
	return o.isOwner() || o.isAdmin()
}

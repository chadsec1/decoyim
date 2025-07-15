package muc

import (
	"github.com/chadsec1/decoyim/xmpp/jid"
	. "gopkg.in/check.v1"
)

type MucRoomManagerSuite struct{}

var _ = Suite(&MucRoomManagerSuite{})

func (*MucRoomManagerSuite) Test_NewRoomManager(c *C) {
	rr := NewRoomManager()
	c.Assert(rr.rooms, Not(IsNil))
}

func (*MucRoomManagerSuite) Test_RoomManager_GetRoom(c *C) {
	rr := NewRoomManager()
	room := &Room{}
	rr.rooms["foo@bar.com"] = room

	nr, ok := rr.GetRoom(jid.ParseBare("foo@bar.com"))

	c.Assert(ok, Equals, true)
	c.Assert(nr, Equals, room)
}

func (*MucRoomManagerSuite) Test_RoomManager_AddRoom(c *C) {
	rr := NewRoomManager()

	ok := rr.AddRoom(&Room{ID: jid.ParseBare("foo@bar.com")})
	c.Assert(ok, Equals, true)

	ok = rr.AddRoom(&Room{ID: jid.ParseBare("foo@bar.com")})
	c.Assert(ok, Equals, false)
}

func (*MucRoomManagerSuite) Test_RoomManager_LeaveRoom(c *C) {
	rr := NewRoomManager()

	_ = rr.AddRoom(&Room{ID: jid.ParseBare("somewhere@bar.com")})
	_ = rr.AddRoom(&Room{ID: jid.ParseBare("foo@bar.com")})

	rr.DeleteRoom(jid.ParseBare("somewhere@bar.com"))
	c.Assert(hasRoom(rr, jid.ParseBare("somewhere@bar.com")), Equals, false)

	rr.DeleteRoom(jid.ParseBare("foo@bar.com"))
	c.Assert(hasRoom(rr, jid.ParseBare("foo@bar.com")), Equals, false)
}

func (*MucRoomManagerSuite) Test_RoomManager_GetAllRooms(c *C) {
	rr := NewRoomManager()
	_ = rr.AddRoom(&Room{ID: jid.ParseBare("foo@bar.com")})
	_ = rr.AddRoom(&Room{ID: jid.ParseBare("bat@man.com")})

	ar := rr.GetAllRooms()

	c.Assert(ar, HasLen, 2)
	for _, r := range ar {
		_, ok := rr.GetRoom(r.ID)
		c.Assert(ok, Equals, true)
	}
}

func hasRoom(manager *RoomManager, roomID jid.Bare) bool {
	_, ok := manager.GetRoom(roomID)
	return ok
}

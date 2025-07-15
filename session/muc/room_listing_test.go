package muc

import (
	"bytes"
	"io"
	"os"

	"github.com/chadsec1/decoyim/session/muc/data"
	xmppData "github.com/chadsec1/decoyim/xmpp/data"
	. "gopkg.in/check.v1"
)

type MucRoomListingSuite struct{}

var _ = Suite(&MucRoomListingSuite{})

func (*MucRoomListingSuite) Test_NewRoomListing(c *C) {
	c.Assert(NewRoomListing(), Not(IsNil))
}

func (*MucRoomListingSuite) Test_RoomListing_GetDiscoInfo(c *C) {
	rl := &RoomListing{}
	dd := data.RoomDiscoInfo{
		AnonymityLevel: "stuff",
	}

	rl.RoomDiscoInfo = dd

	c.Assert(rl.GetDiscoInfo(), DeepEquals, dd)
}

func (*MucRoomListingSuite) Test_RoomListing_Updated(c *C) {
	rl := &RoomListing{}

	rl.Updated()

	called1 := false
	var data1 interface{}
	rl.OnUpdate(func(_ *RoomListing, dd interface{}) {
		called1 = true
		data1 = dd
	}, "data1")

	called2 := false
	var data2 interface{}
	rl.OnUpdate(func(_ *RoomListing, dd interface{}) {
		called2 = true
		data2 = dd
	}, "data2")

	rl.Updated()

	c.Assert(called1, Equals, true)
	c.Assert(called2, Equals, true)
	c.Assert(data1, DeepEquals, "data1")
	c.Assert(data2, DeepEquals, "data2")
}

func (*MucRoomListingSuite) Test_RoomListing_SetFeatures(c *C) {
	rl := &RoomListing{}

	rl.SetFeatures([]xmppData.DiscoveryFeature{})

	rl.SetFeatures([]xmppData.DiscoveryFeature{
		{Var: "http://jabber.org/protocol/muc"},
		{Var: "http://jabber.org/protocol/muc"},
		{Var: "http://jabber.org/protocol/muc#stable_id"},
		{Var: "http://jabber.org/protocol/muc#self-ping-optimization"},
		{Var: "http://jabber.org/protocol/disco#info"},
		{Var: "http://jabber.org/protocol/disco#items"},
		{Var: "urn:xmpp:mam:0"},
		{Var: "urn:xmpp:mam:1"},
		{Var: "urn:xmpp:mam:2"},
		{Var: "urn:xmpp:mam:tmp"},
		{Var: "urn:xmpp:mucsub:0"},
		{Var: "urn:xmpp:sid:0"},
		{Var: "vcard-temp"},
		{Var: "http://jabber.org/protocol/muc#request"},
		{Var: "jabber:iq:register"},
		{Var: "muc_semianonymous"},
		{Var: "muc_persistent"},
		{Var: "muc_unmoderated"},
		{Var: "muc_open"},
		{Var: "muc_passwordprotected"},
		{Var: "muc_public"},
	})

	c.Assert(rl.SupportsVoiceRequests, Equals, true)
	c.Assert(rl.AllowsRegistration, Equals, true)
	c.Assert(rl.AnonymityLevel, Equals, "semi")
	c.Assert(rl.Persistent, Equals, true)
	c.Assert(rl.Moderated, Equals, false)
	c.Assert(rl.Open, Equals, true)
	c.Assert(rl.PasswordProtected, Equals, true)
	c.Assert(rl.Public, Equals, true)

	rl.SetFeatures([]xmppData.DiscoveryFeature{
		{Var: "muc_nonanonymous"},
		{Var: "muc_temporary"},
		{Var: "muc_moderated"},
		{Var: "muc_membersonly"},
		{Var: "muc_unsecured"},
		{Var: "muc_hidden"},
	})

	c.Assert(rl.AnonymityLevel, Equals, "no")
	c.Assert(rl.Public, Equals, false)
	c.Assert(rl.PasswordProtected, Equals, false)
	c.Assert(rl.Open, Equals, false)
	c.Assert(rl.Moderated, Equals, true)
	c.Assert(rl.Persistent, Equals, false)
}

func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func (*MucRoomListingSuite) Test_RoomListing_SetFormsData(c *C) {
	rl := &RoomListing{}
	rl.SetFormsData(nil)

	rl.SetFormsData([]xmppData.Form{
		{
			Type: "result",
			Fields: []xmppData.FormFieldX{
				{Var: "FORM_TYPE", Values: []string{discoInfoFieldFormType}},
				{Var: discoInfoFieldLang, Values: []string{"eng"}},
			},
		},
	})

	c.Assert(rl.Language, Equals, "eng")
}

func (*MucRoomListingSuite) Test_RoomListing_updateWithFormField(c *C) {
	rl := &RoomListing{}

	rl.Language = "swe"
	rl.updateWithFormField("muc#roominfo_lang", []string{})
	c.Assert(rl.Language, Equals, "swe")
	rl.updateWithFormField("muc#roominfo_lang", []string{"en", "something"})
	c.Assert(rl.Language, Equals, "en")

	rl.OccupantsCanChangeSubject = false
	rl.updateWithFormField("muc#roominfo_changesubject", []string{"1"})
	c.Assert(rl.OccupantsCanChangeSubject, Equals, true)
	rl.updateWithFormField("muc#roominfo_changesubject", []string{"0", "1"})
	c.Assert(rl.OccupantsCanChangeSubject, Equals, false)

	rl.Logged = false
	rl.updateWithFormField("muc#roomconfig_enablelogging", []string{"1"})
	c.Assert(rl.Logged, Equals, true)
	rl.updateWithFormField("muc#roomconfig_enablelogging", []string{})
	c.Assert(rl.Logged, Equals, true)
	rl.updateWithFormField("muc#roomconfig_enablelogging", []string{"0", "1"})
	c.Assert(rl.Logged, Equals, false)

	rl.Title = "hello"
	rl.updateWithFormField("muc#roomconfig_roomname", []string{})
	c.Assert(rl.Title, Equals, "")
	rl.updateWithFormField("muc#roomconfig_roomname", []string{"something", "foo"})
	c.Assert(rl.Title, Equals, "something")

	rl.Description = "hello"
	rl.updateWithFormField("muc#roominfo_description", []string{})
	c.Assert(rl.Description, Equals, "hello")
	rl.updateWithFormField("muc#roominfo_description", []string{"something", "foo"})
	c.Assert(rl.Description, Equals, "something")

	rl.Occupants = 42
	rl.updateWithFormField("muc#roominfo_occupants", []string{})
	c.Assert(rl.Occupants, Equals, 42)
	rl.updateWithFormField("muc#roominfo_occupants", []string{"xq"})
	c.Assert(rl.Occupants, Equals, 42)
	rl.updateWithFormField("muc#roominfo_occupants", []string{"55"})
	c.Assert(rl.Occupants, Equals, 55)

	rl.MembersCanInvite = false
	rl.updateWithFormField("{http://prosody.im/protocol/muc}roomconfig_allowmemberinvites", []string{"1"})
	c.Assert(rl.MembersCanInvite, Equals, true)
	rl.updateWithFormField("{http://prosody.im/protocol/muc}roomconfig_allowmemberinvites", []string{})
	c.Assert(rl.MembersCanInvite, Equals, true)
	rl.updateWithFormField("{http://prosody.im/protocol/muc}roomconfig_allowmemberinvites", []string{"0", "1"})
	c.Assert(rl.MembersCanInvite, Equals, false)

	rl.OccupantsCanInvite = false
	rl.updateWithFormField("muc#roomconfig_allowinvites", []string{"1"})
	c.Assert(rl.OccupantsCanInvite, Equals, true)
	rl.updateWithFormField("muc#roomconfig_allowinvites", []string{})
	c.Assert(rl.OccupantsCanInvite, Equals, true)
	rl.updateWithFormField("muc#roomconfig_allowinvites", []string{"0", "1"})
	c.Assert(rl.OccupantsCanInvite, Equals, false)

	rl.AllowPrivateMessages = "somewhere"
	rl.updateWithFormField("muc#roomconfig_allowpm", []string{})
	c.Assert(rl.AllowPrivateMessages, Equals, "somewhere")
	rl.updateWithFormField("muc#roomconfig_allowpm", []string{"something", "foo"})
	c.Assert(rl.AllowPrivateMessages, Equals, "something")

	rl.ContactJid = "somewhere"
	rl.updateWithFormField("muc#roominfo_contactjid", []string{})
	c.Assert(rl.ContactJid, Equals, "somewhere")
	rl.updateWithFormField("muc#roominfo_contactjid", []string{"something", "foo"})
	c.Assert(rl.ContactJid, Equals, "something")

	rl.MaxHistoryFetch = 42
	rl.updateWithFormField("muc#maxhistoryfetch", []string{})
	c.Assert(rl.MaxHistoryFetch, Equals, 42)
	rl.updateWithFormField("muc#maxhistoryfetch", []string{"xq"})
	c.Assert(rl.MaxHistoryFetch, Equals, 42)
	rl.updateWithFormField("muc#maxhistoryfetch", []string{"55"})
	c.Assert(rl.MaxHistoryFetch, Equals, 55)
}

package muc

import (
	"strconv"
	"sync"

	"github.com/chadsec1/decoyim/session/muc/data"
	xmppData "github.com/chadsec1/decoyim/xmpp/data"
	"github.com/chadsec1/decoyim/xmpp/jid"
)

const (
	discoInfoFieldFormType             = "http://jabber.org/protocol/muc#roominfo"
	discoInfoFieldLang                 = "muc#roominfo_lang"
	discoInfoFieldChangeSubject        = "muc#roominfo_changesubject"
	discoInfoFieldCanChangeSubject     = configFieldCanChangeSubject
	discoInfoFieldEnableLogging        = configFieldEnableLogging
	discoInfoFieldRoomName             = configFieldRoomName
	discoInfoFieldDescription          = "muc#roominfo_description"
	discoInfoFieldOccupants            = "muc#roominfo_occupants"
	discoInfoFieldAllowMemberInvites   = configFieldAllowMemberInvites
	discoInfoFieldAllowInvites         = configFieldAllowInvites
	discoInfoFieldAllowPrivateMessages = configFieldAllowPM
	discoInfoFieldContactJid           = "muc#roominfo_contactjid"
	discoInfoFieldMaxHistoryFetch      = configFieldMaxHistoryFetch
)

// RoomListing contains the information about a room for listing it
type RoomListing struct {
	Service     jid.Any
	ServiceName string
	Jid         jid.Bare
	Name        string

	data.RoomDiscoInfo

	lockUpdates sync.RWMutex
	onUpdates   []func()
}

// NewRoomListing creates and returns a new room listing
func NewRoomListing() *RoomListing {
	return &RoomListing{}
}

// GetDiscoInfo returns the room disco info from the room listing
func (rl *RoomListing) GetDiscoInfo() data.RoomDiscoInfo {
	return rl.RoomDiscoInfo
}

// OnUpdate takes a function and some data, and when this room listing is updated, that function
// will be called with the current room listing and the associated data
func (rl *RoomListing) OnUpdate(f func(*RoomListing, interface{}), data interface{}) {
	rl.lockUpdates.Lock()
	defer rl.lockUpdates.Unlock()

	rl.onUpdates = append(rl.onUpdates, func() {
		f(rl, data)
	})
}

// Updated should be called after a room listing has been updated, to notify observers of the update
func (rl *RoomListing) Updated() {
	rl.lockUpdates.RLock()
	defer rl.lockUpdates.RUnlock()

	for _, f := range rl.onUpdates {
		f()
	}
}

// SetFeatures receive a list of features and updates the room listing properties based on each feature
func (rl *RoomListing) SetFeatures(features []xmppData.DiscoveryFeature) {
	rl.lockUpdates.Lock()
	defer rl.lockUpdates.Unlock()

	for _, feat := range features {
		rl.setFeature(feat.Var)
	}
}

// SetFeature updates a room listing propertie based on the given feature
func (rl *RoomListing) setFeature(feature string) {
	switch feature {
	case "http://jabber.org/protocol/muc":
		// Supports MUC - probably not useful for us
	case "http://jabber.org/protocol/muc#stable_id":
		// This means the server will use the same id in groupchat messages
	case "http://jabber.org/protocol/muc#self-ping-optimization":
		// This means the chat room supports XEP-0410, that allows
		// users to see if they're still connected to a chat room.
	case "http://jabber.org/protocol/disco#info":
		// Ignore
	case "http://jabber.org/protocol/disco#items":
		// Ignore
	case "urn:xmpp:mam:0":
		// Ignore
	case "urn:xmpp:mam:1":
		// Ignore
	case "urn:xmpp:mam:2":
		// Ignore
	case "urn:xmpp:mam:tmp":
		// Ignore
	case "urn:xmpp:mucsub:0":
		// Ignore
	case "urn:xmpp:sid:0":
		// Ignore
	case "vcard-temp":
		// Ignore
	case "http://jabber.org/protocol/muc#request":
		rl.SupportsVoiceRequests = true
	case "jabber:iq:register":
		rl.AllowsRegistration = true
	case "muc_semianonymous":
		rl.AnonymityLevel = "semi"
	case "muc_nonanonymous":
		rl.AnonymityLevel = "no"
	case "muc_persistent":
		rl.Persistent = true
	case "muc_temporary":
		rl.Persistent = false
	case "muc_unmoderated":
		rl.Moderated = false
	case "muc_moderated":
		rl.Moderated = true
	case "muc_open":
		rl.Open = true
	case "muc_membersonly":
		rl.Open = false
	case "muc_passwordprotected":
		rl.PasswordProtected = true
	case "muc_unsecured":
		rl.PasswordProtected = false
	case "muc_public":
		rl.Public = true
	case "muc_hidden":
		rl.Public = false
	}
}

// SetFormsData extract the forms data and updates the room listing properties based on each data
func (rl *RoomListing) SetFormsData(forms []xmppData.Form) {
	rl.lockUpdates.Lock()
	defer rl.lockUpdates.Unlock()

	for _, form := range forms {
		fields := formFieldsToKeyValue(form.Fields)
		if isValidRoomInfoForm(form, fields) {
			rl.updateWithFormFields(form, fields)
		}
	}
}

func (rl *RoomListing) updateWithFormFields(form xmppData.Form, fields map[string][]string) {
	for field, values := range fields {
		rl.updateWithFormField(field, values)
	}
}

func (rl *RoomListing) updateWithFormField(field string, values []string) {
	switch field {
	case "FORM_TYPE":
		// Ignore, we already checked
	case discoInfoFieldLang:
		if len(values) > 0 {
			rl.Language = values[0]
		}
	case discoInfoFieldChangeSubject, discoInfoFieldCanChangeSubject:
		// When the `roominfo_changesubject` field is changed to false,
		// the response is not 0 for false value, this response is `empty`.
		// For this reason, this field is `initialized` with false
		rl.OccupantsCanChangeSubject = false
		if len(values) > 0 {
			rl.OccupantsCanChangeSubject = values[0] == "1"
		}
	case discoInfoFieldEnableLogging:
		if len(values) > 0 {
			rl.Logged = values[0] == "1"
		}
	case discoInfoFieldRoomName:
		// Initialized with an empty string because when `muc#roomconfig_roomname`
		// has no value, the` Title` field is not updated
		rl.Title = ""
		if len(values) > 0 {
			rl.Title = values[0]
		}
	case discoInfoFieldDescription:
		if len(values) > 0 {
			rl.Description = values[0]
		}
	case discoInfoFieldOccupants:
		if len(values) > 0 {
			res, e := strconv.Atoi(values[0])
			if e == nil {
				rl.Occupants = res
			}
		}
	case discoInfoFieldAllowMemberInvites:
		if len(values) > 0 {
			rl.MembersCanInvite = values[0] == "1"
		}
	case discoInfoFieldAllowInvites:
		if len(values) > 0 {
			rl.OccupantsCanInvite = values[0] == "1"
		}
	case discoInfoFieldAllowPrivateMessages:
		if len(values) > 0 {
			rl.AllowPrivateMessages = values[0]
		}
	case discoInfoFieldContactJid:
		if len(values) > 0 {
			rl.ContactJid = values[0]
		}
	case discoInfoFieldMaxHistoryFetch:
		if len(values) > 0 {
			res, e := strconv.Atoi(values[0])
			if e == nil {
				rl.MaxHistoryFetch = res
			}
		}
	}
}

func formFieldsToKeyValue(fields []xmppData.FormFieldX) map[string][]string {
	result := make(map[string][]string)
	for _, field := range fields {
		result[field.Var] = field.Values
	}

	return result
}

func isValidRoomInfoForm(form xmppData.Form, fields map[string][]string) bool {
	return form.Type == "result" && hasRoomInfoFormType(fields)
}

func hasRoomInfoFormType(fields map[string][]string) bool {
	return len(fields["FORM_TYPE"]) > 0 && fields["FORM_TYPE"][0] == discoInfoFieldFormType
}

package scram

import (
	"io/ioutil"
	"testing"

	"github.com/chadsec1/decoyim/i18n"
	"github.com/coyim/gotk3adapter/glib_mock"
	log "github.com/sirupsen/logrus"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

func init() {
	log.SetOutput(ioutil.Discard)
	i18n.InitLocalization(&glib_mock.Mock{})
}

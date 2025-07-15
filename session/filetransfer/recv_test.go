package filetransfer

import (
	"io/ioutil"
	"os"

	sdata "github.com/chadsec1/decoyim/session/data"
	"github.com/chadsec1/decoyim/xmpp/data"
	"github.com/sirupsen/logrus/hooks/test"
	. "gopkg.in/check.v1"
)

type RecvSuite struct{}

var _ = Suite(&RecvSuite{})

func (s *RecvSuite) Test_chooseAppropriateFileTransferOptionFrom(c *C) {
	orgSupportedFileTransferMethods := supportedFileTransferMethods
	defer func() {
		supportedFileTransferMethods = orgSupportedFileTransferMethods
	}()

	supportedFileTransferMethods = map[string]int{}

	res, ok := chooseAppropriateFileTransferOptionFrom([]string{})
	c.Assert(res, Equals, "")
	c.Assert(ok, Equals, false)

	res, ok = chooseAppropriateFileTransferOptionFrom([]string{"foo", "bar"})
	c.Assert(res, Equals, "")
	c.Assert(ok, Equals, false)

	supportedFileTransferMethods["bar"] = 2
	res, ok = chooseAppropriateFileTransferOptionFrom([]string{"foo", "bar"})
	c.Assert(res, Equals, "bar")
	c.Assert(ok, Equals, true)

	supportedFileTransferMethods["bar"] = 2
	res, ok = chooseAppropriateFileTransferOptionFrom([]string{"bar", "foo"})
	c.Assert(res, Equals, "bar")
	c.Assert(ok, Equals, true)
}

func (s *RecvSuite) Test_iqResultChosenStreamMethod(c *C) {
	res := iqResultChosenStreamMethod("foo")
	c.Assert(res, DeepEquals, data.SI{
		File: &data.File{},
		Feature: data.FeatureNegotation{
			Form: data.Form{
				Type: "submit",
				Fields: []data.FormFieldX{
					{Var: "stream-method", Values: []string{"foo"}},
				},
			},
		},
	})
}

func (s *RecvSuite) Test_recvContext_finalizeFileTransfer_forFile(c *C) {
	tf, ex := ioutil.TempFile("", "coyim-session-7-")
	c.Assert(ex, IsNil)
	_, ex = tf.Write([]byte(`hello again`))
	c.Assert(ex, IsNil)
	ex = tf.Close()
	c.Assert(ex, IsNil)

	tf2, ex3 := ioutil.TempFile("", "coyim-session-8-")
	c.Assert(ex3, IsNil)
	ex3 = tf2.Close()
	c.Assert(ex3, IsNil)
	defer func() {
		ex4 := os.Remove(tf2.Name())
		c.Assert(ex4, IsNil)
	}()

	ctrl := sdata.CreateFileTransferControl(func() bool { return false }, func(bool) {})
	notDeclined := make(chan bool)
	go func() {
		ctrl.WaitForFinish(func(v bool) {
			notDeclined <- v
		})
	}()

	ctx := &recvContext{
		directory:   false,
		destination: tf2.Name(),
		control:     ctrl,
	}

	e := ctx.finalizeFileTransfer(tf.Name())
	c.Assert(e, IsNil)
	notDec := <-notDeclined

	c.Assert(notDec, Equals, true)
}

func (s *RecvSuite) Test_recvContext_finalizeFileTransfer_forFile_failsOnRename(c *C) {
	ctrl := sdata.CreateFileTransferControl(func() bool { return false }, func(bool) {})
	ee := make(chan error)
	go func() {
		ctrl.WaitForError(func(eee error) {
			ee <- eee
		})
	}()

	l, _ := test.NewNullLogger()
	sess := &sessionMockWithCustomLog{
		log: l,
	}
	ctx := &recvContext{
		s:           sess,
		directory:   false,
		destination: "hmm",
		control:     ctrl,
	}

	e := ctx.finalizeFileTransfer("file that hopefully doesn't exist")
	c.Assert(e, ErrorMatches, ".*(no such file or directory|cannot find the (file|path) specified).*")
	e2 := <-ee
	c.Assert(e2, ErrorMatches, "Couldn't save final file")
}

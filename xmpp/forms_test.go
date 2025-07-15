package xmpp

import (
	"encoding/base64"
	"encoding/xml"
	"errors"

	"github.com/chadsec1/decoyim/xmpp/data"

	. "gopkg.in/check.v1"
)

type FormsXMPPSuite struct{}

var _ = Suite(&FormsXMPPSuite{})

func (s *FormsXMPPSuite) Test_processForm_returnsErrorFromCallback(c *C) {
	e := errors.New("some kind of error")
	f := &data.Form{}
	_, err := processForm(f, nil, "", nil, func(title, instructions string, fields []interface{}, link *data.OobLink, hasForm bool) error {
		c.Assert(len(fields), Equals, 0)
		return e
	})

	c.Assert(err, Equals, e)
}

func (s *FormsXMPPSuite) Test_processForm_returnsEmptySubmitFormForEmptyForm(c *C) {
	f := &data.Form{}
	f2, err := processForm(f, nil, "", nil, func(title, instructions string, fields []interface{}, link *data.OobLink, hasForm bool) error {
		c.Assert(len(fields), Equals, 0)
		return nil
	})

	c.Assert(err, IsNil)
	c.Assert(*f2, DeepEquals, data.Form{Type: "submit"})
}

func (s *FormsXMPPSuite) Test_processForm_processButDoesNotReturnFixedFields(c *C) {
	f := &data.Form{}
	f.Fields = []data.FormFieldX{
		{
			Var:    "hello_field1",
			Label:  "hello",
			Type:   "fixed",
			Values: []string{"Something"},
		},
		//Malformed
		{
			Var:   "hello_field2",
			Label: "hello2",
			Type:  "fixed",
		},
	}
	f2, err := processForm(f, nil, "", nil, func(title, instructions string, fields []interface{}, link *data.OobLink, hasForm bool) error {
		c.Assert(len(fields), Equals, 1)
		c.Assert(fields[0], DeepEquals, &data.FixedFormField{
			FormField: data.FormField{
				Name:  "hello_field1",
				Label: "hello",
				Type:  "fixed",
			},
			Text: "Something",
		})
		return nil
	})

	c.Assert(err, IsNil)
	c.Assert(*f2, DeepEquals, data.Form{
		Type:   "submit",
		Fields: nil})
}

func (s *FormsXMPPSuite) Test_processForm_returnsBooleanFields(c *C) {
	f := &data.Form{}
	f.Fields = []data.FormFieldX{
		{
			Var:   "hello_field3",
			Label: "hello3",
			Type:  "boolean",
		},
	}
	f2, err := processForm(f, nil, "", nil, func(title, instructions string, fields []interface{}, link *data.OobLink, hasForm bool) error {
		c.Assert(len(fields), Equals, 1)
		c.Assert(fields[0], DeepEquals, &data.BooleanFormField{
			FormField: data.FormField{
				Name:  "hello_field3",
				Label: "hello3",
				Type:  "boolean",
			},
			Result: false,
		})

		return nil
	})

	c.Assert(err, IsNil)
	c.Assert(*f2, DeepEquals, data.Form{
		Type: "submit",
		Fields: []data.FormFieldX{
			{
				Var:    "hello_field3",
				Values: []string{"false"},
			}}},
	)
}

func (s *FormsXMPPSuite) Test_processForm_returnsMultiFields(c *C) {
	f := &data.Form{}
	f.Fields = []data.FormFieldX{
		{
			Var:   "hello_field4",
			Label: "hello4",
			Type:  "jid-multi",
		},
		{
			Var:   "hello_field5",
			Label: "hello5",
			Type:  "text-multi",
		},
	}
	f2, err := processForm(f, nil, "", nil, func(title, instructions string, fields []interface{}, link *data.OobLink, hasForm bool) error {
		c.Assert(len(fields), Equals, 2)

		c.Assert(fields[0], DeepEquals, &data.MultiTextFormField{
			FormField: data.FormField{
				Name:  "hello_field4",
				Label: "hello4",
				Type:  "jid-multi",
			},
		})

		c.Assert(fields[1], DeepEquals, &data.MultiTextFormField{
			FormField: data.FormField{
				Name:  "hello_field5",
				Label: "hello5",
				Type:  "text-multi",
			},
		})

		return nil
	})

	c.Assert(err, IsNil)
	c.Assert(*f2, DeepEquals, data.Form{
		Type: "submit",
		Fields: []data.FormFieldX{
			{
				Var: "hello_field4",
			},
			{
				Var: "hello_field5",
			},
		}})
}

func (s *FormsXMPPSuite) Test_processForm_returnsListSingle(c *C) {
	f := &data.Form{}
	f.Fields = []data.FormFieldX{
		{
			Var:   "hello_field7",
			Label: "hello7",
			Type:  "list-single",
			Options: []data.FormFieldOptionX{
				{Var: "One", Value: "Two"},
				{Var: "Three", Value: "Four"},
			},

			Values: []string{"Four"},
		},
	}
	f2, err := processForm(f, nil, "", nil, func(title, instructions string, fields []interface{}, link *data.OobLink, hasForm bool) error {
		c.Assert(len(fields), Equals, 1)

		c.Assert(fields[0], DeepEquals, &data.SelectionFormField{
			FormField: data.FormField{
				Name:  "hello_field7",
				Label: "hello7",
				Type:  "list-single",
			},
			Values: []string{"One", "Three"},
			Ids:    []string{"Two", "Four"},
			Result: 1,
		})

		return nil
	})

	c.Assert(err, IsNil)
	c.Assert(f2, DeepEquals, &data.Form{
		Type: "submit",
		Fields: []data.FormFieldX{
			{
				Var:    "hello_field7",
				Values: []string{"Four"},
			},
		}})
}

func (s *FormsXMPPSuite) Test_processForm_returnsListMulti(c *C) {
	f := &data.Form{}
	f.Fields = []data.FormFieldX{
		{
			Var:   "hello_field1o7",
			Label: "hello1o7",
			Type:  "list-multi",
			Options: []data.FormFieldOptionX{
				{Var: "One", Value: "Two"},
				{Var: "Three", Value: "Four"},
				{Var: "Five", Value: "Six"},
				{Var: "Seven", Value: "Eight"},
			},

			Values: []string{"Six", "Two"},
		},
	}
	f2, err := processForm(f, nil, "", nil, func(title, instructions string, fields []interface{}, link *data.OobLink, hasForm bool) error {
		c.Assert(len(fields), Equals, 1)

		c.Assert(fields[0], DeepEquals, &data.MultiSelectionFormField{
			FormField: data.FormField{
				Name:  "hello_field1o7",
				Label: "hello1o7",
				Type:  "list-multi",
			},
			Values:  []string{"One", "Three", "Five", "Seven"},
			Ids:     []string{"Two", "Four", "Six", "Eight"},
			Results: []int{0, 2},
		})

		return nil
	})

	c.Assert(err, IsNil)
	c.Assert(f2, DeepEquals, &data.Form{
		Type: "submit",
		Fields: []data.FormFieldX{
			{
				Var:    "hello_field1o7",
				Values: []string{"Two", "Six"},
			}}})
}

func (s *FormsXMPPSuite) Test_processForm_returnsHidden(c *C) {
	f := &data.Form{}
	f.Fields = []data.FormFieldX{
		{
			Var:    "hello_field1o71",
			Label:  "hello1o71",
			Type:   "hidden",
			Values: []string{"secret"},
		},
	}
	f2, err := processForm(f, nil, "", nil, func(title, instructions string, fields []interface{}, link *data.OobLink, hasForm bool) error {
		c.Assert(len(fields), Equals, 0)
		return nil
	})

	c.Assert(err, IsNil)
	c.Assert(*f2, DeepEquals, data.Form{
		Type: "submit",
		Fields: []data.FormFieldX{
			{
				Var:    "hello_field1o71",
				Values: []string{"secret"},
			}}})
}

func (s *FormsXMPPSuite) Test_processForm_returnsUnknown(c *C) {
	f := &data.Form{}
	f.Fields = []data.FormFieldX{
		{
			Var:   "hello_field1o71",
			Label: "hello1o71",
			Type:  "another-fancy-type",
		},
		{
			Var:    "hello_field1o73",
			Label:  "hello1o73",
			Type:   "another-fancy-type",
			Values: []string{"another one"},
		},
	}
	f2, err := processForm(f, nil, "", nil, func(title, instructions string, fields []interface{}, link *data.OobLink, hasForm bool) error {
		c.Assert(len(fields), Equals, 2)

		c.Assert(fields[0], DeepEquals, &data.TextFormField{
			FormField: data.FormField{
				Label: "hello1o71",
				Type:  "another-fancy-type",
				Name:  "hello_field1o71",
			},
		})

		c.Assert(fields[1], DeepEquals, &data.TextFormField{
			FormField: data.FormField{
				Label: "hello1o73",
				Type:  "another-fancy-type",
				Name:  "hello_field1o73",
			},
			Default: "another one",
		})

		//The UI should set the value, and it should be available on the submit form
		fields[0].(*data.TextFormField).Result = "Value from UI"

		return nil
	})

	c.Assert(err, IsNil)
	c.Assert(*f2, DeepEquals, data.Form{
		Type: "submit",
		Fields: []data.FormFieldX{
			{
				Var:    "hello_field1o71",
				Values: []string{"Value from UI"},
			},
			{
				Var:    "hello_field1o73",
				Values: []string{""}, // Value is lost because the UI does not set anything. Expected.
			}}})
}

type testOtherFormType struct{}

func (s *FormsXMPPSuite) Test_processForm_panicsWhenGivenAWeirdFormType(c *C) {
	f := &data.Form{}
	f.Fields = []data.FormFieldX{
		{
			Label: "hello1o71",
			Type:  "another-fancy-type",
		},
	}
	c.Assert(func() {
		_, _ = processForm(f, nil, "", nil, func(title, instructions string, fields []interface{}, link *data.OobLink, hasForm bool) error {
			fields[0] = testOtherFormType{}
			return nil
		})
	}, PanicMatches, "unknown field type in result from callback: xmpp.testOtherFormType")
}

func (s *FormsXMPPSuite) Test_processForm_setsAValidBooleanReturnValue(c *C) {
	f := &data.Form{}
	f.Fields = []data.FormFieldX{
		{
			Var:   "hello_field1o71",
			Label: "hello1o71",
			Type:  "boolean",
		},
	}
	f2, _ := processForm(f, nil, "", nil, func(title, instructions string, fields []interface{}, link *data.OobLink, hasForm bool) error {
		c.Assert(len(fields), Equals, 1)
		fields[0].(*data.BooleanFormField).Result = true
		return nil
	})
	c.Assert(*f2, DeepEquals, data.Form{
		Type: "submit",
		Fields: []data.FormFieldX{
			{
				XMLName: xml.Name{Space: "", Local: ""},
				Var:     "hello_field1o71",
				Values:  []string{"true"},
			}}})
}

func (s *FormsXMPPSuite) Test_processForm_returnsListMultiWithResults(c *C) {
	f := &data.Form{}
	f.Fields = []data.FormFieldX{
		{
			Var:   "hello_field1o7",
			Label: "hello1o7",
			Type:  "list-multi",
			Options: []data.FormFieldOptionX{
				{Var: "One", Value: "Two"},
				{Var: "Three", Value: "Four"},
			},
		},
	}
	f2, err := processForm(f, nil, "", nil, func(title, instructions string, fields []interface{}, link *data.OobLink, hasForm bool) error {
		c.Assert(len(fields), Equals, 1)
		fields[0].(*data.MultiSelectionFormField).Results = []int{1}
		return nil
	})

	c.Assert(err, IsNil)
	c.Assert(*f2, DeepEquals, data.Form{
		Type: "submit",
		Fields: []data.FormFieldX{
			{
				Var:    "hello_field1o7",
				Values: []string{"Four"},
			}}})
}

func (s *FormsXMPPSuite) Test_processForm_dealsWithMediaCorrectly(c *C) {
	fooBarDecoded := []byte("hello world")
	f := &data.Form{}
	datas := []data.BobData{
		{
			CID:    "foobax",
			Base64: ".....",
		},
		{
			CID:    "foobar",
			Base64: base64.StdEncoding.EncodeToString(fooBarDecoded),
		},
	}
	f.Fields = []data.FormFieldX{
		{
			Var:    "hello1",
			Label:  "hello",
			Type:   "fixed",
			Values: []string{"Something"},
			Media: []data.FormFieldMediaX{
				{
					URIs: []data.MediaURIX{
						{
							MIMEType: "application/not-a-uri",
							URI:      "",
						},
						{
							MIMEType: "application/not-a-cid-uri",
							URI:      "hello:world",
						},
						{
							MIMEType: "application/valid-encoding",
							URI:      "cid:foobar",
						},
						{
							MIMEType: "application/invalid-encoding",
							URI:      "cid:foobax",
						},
					},
				},
			},
		},
		{
			Var:   "hello2",
			Label: "hello1o7",
			Type:  "hidden",
			Media: []data.FormFieldMediaX{
				{
					URIs: []data.MediaURIX{
						{
							MIMEType: "application/does-not-matter-because-it-is-ignored",
							URI:      "hello:world",
						},
						{
							MIMEType: "application/does-not-matter-because-it-is-also-ignored",
							URI:      "cid:foobax",
						},
					},
				},
			},
		},
	}

	f2, err := processForm(f, datas, "", nil, func(title, instructions string, fields []interface{}, link *data.OobLink, hasForm bool) error {
		//NOTE: hidden fields are not passed to the callback so you can't have access to any media
		//in hidden fields.
		c.Assert(len(fields), Equals, 1)
		c.Assert(fields[0], DeepEquals, &data.FixedFormField{
			FormField: data.FormField{
				Name:  "hello1",
				Label: "hello",
				Type:  "fixed",
				Media: [][]data.Media{
					{
						{
							MIMEType: "application/not-a-cid-uri",
							URI:      "hello:world",
						},
						{
							MIMEType: "application/valid-encoding",
							Data:     fooBarDecoded,
						},
					}},
			},
			Text: "Something",
		})
		return nil
	})

	c.Assert(err, IsNil)
	c.Assert(*f2, DeepEquals, data.Form{
		Type: "submit",
		Fields: []data.FormFieldX{
			{
				Var: "hello2",
			}}})
}

func (s *FormsXMPPSuite) Test_toFormField_handlesBoolean(c *C) {
	f := data.FormFieldX{
		Type: "boolean",
	}

	res := toFormField(f, nil)
	resb := res.(*data.BooleanFormField)
	c.Assert(resb.Result, Equals, false)

	f = data.FormFieldX{
		Type:   "boolean",
		Values: []string{"true"},
	}

	res = toFormField(f, nil)
	resb = res.(*data.BooleanFormField)
	c.Assert(resb.Result, Equals, true)

	f = data.FormFieldX{
		Type:   "boolean",
		Values: []string{"false"},
	}

	res = toFormField(f, nil)
	resb = res.(*data.BooleanFormField)
	c.Assert(resb.Result, Equals, false)

	f = data.FormFieldX{
		Type:   "boolean",
		Values: []string{"1"},
	}

	res = toFormField(f, nil)
	resb = res.(*data.BooleanFormField)
	c.Assert(resb.Result, Equals, true)
}

func (s *FormsXMPPSuite) Test_toFormField_hidden(c *C) {
	f := data.FormFieldX{
		Type: "hidden",
	}

	res := toFormField(f, nil)
	c.Assert(res, IsNil)
}

func (s *FormsXMPPSuite) Test_toFormField_ocr_var(c *C) {
	f := data.FormFieldX{
		Type: "bla",
		Var:  "ocr",
	}

	res := toFormField(f, [][]data.Media{
		{
			{
				MIMEType: "foo",
				Data:     []byte("bar"),
			},
		},
	})

	resb := res.(*data.CaptchaFormField)
	c.Assert(resb.MediaForm.MIMEType, Equals, "foo")
	c.Assert(resb.MediaForm.Data, DeepEquals, []byte("bar"))
}

func (s *FormsXMPPSuite) Test_toFormFieldX_CaptchaFormField(c *C) {
	ff := &data.CaptchaFormField{
		TextForm: &data.TextFormField{
			FormField: data.FormField{
				Name: "hello",
			},
			Result: "foo",
		},
	}

	res := toFormFieldX(ff)
	c.Assert(*res, DeepEquals, data.FormFieldX{
		Var:    "hello",
		Values: []string{"foo"},
	})
}

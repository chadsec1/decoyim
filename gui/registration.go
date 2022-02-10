package gui

import (
	log "github.com/sirupsen/logrus"

	"github.com/coyim/coyim/config"
	"github.com/coyim/coyim/i18n"
	ourNet "github.com/coyim/coyim/net"
	"github.com/coyim/coyim/servers"
	"github.com/coyim/coyim/session"
	"github.com/coyim/coyim/tls"
	"github.com/coyim/coyim/xmpp"
	"github.com/coyim/coyim/xmpp/data"
	xmppErr "github.com/coyim/coyim/xmpp/errors"
	"github.com/coyim/coyim/xmpp/interfaces"
	"github.com/coyim/gotk3adapter/gtki"
)

type registrationForm struct {
	grid gtki.Grid

	server string
	conf   *config.Account
	fields []formField

	l withLog
}

func (f *registrationForm) accepted() error {
	conf, err := config.NewAccount()
	if err != nil {
		f.l.Log().WithError(err).Debug("error while creating new account")
		return err
	}

	//Find the fields we need to copy from the form to the account
	for _, field := range f.fields {
		switch ff := field.field.(type) {
		case *data.FixedFormField:
			switch ff.Name {
			case "captcha-fallback-text":
				f.l.Log().WithField("text", ff.Label).Debug("Captcha fallback text")
			default:
				f.l.Log().WithField("text", ff.Label).Debug("A field")
			}
		case *data.TextFormField:
			w := field.widget.(gtki.Entry)
			ff.Result, _ = w.GetText()

			switch ff.Label {
			case "User", "Username":
				conf.Account = ff.Result + "@" + f.server
			case "Password":
				conf.Password = ff.Result
			case "Enter the text you see":
				conf.Password = ff.Result
			default:
				f.l.Log().WithField("text", ff.Label).Debug("A field")
			}
		}
	}

	f.conf = conf
	return nil
}

func (f *registrationForm) addFields(fields []interface{}) {
	f.fields = buildWidgetsForFields(fields)
}

func (f *registrationForm) renderForm(title string, fields []interface{}) error {
	doInUIThread(func() {
		f.addFields(fields)

		var i int
		for _, field := range f.fields {
			f.grid.Attach(field.label, 0, i+1, 1, 1)
			f.grid.Attach(field.widget, 1, i+1, 1, 1)
			f.grid.Attach(field.required, 2, i+1, 1, 1)
			i++
		}

		li, _ := g.gtk.LabelNew(i18n.Local("* The field is required."))
		f.grid.Attach(li, 0, i+1, 1, i+1)

		f.grid.ShowAll()
	})

	return nil
}

func requestAndRenderRegistrationForm(server string, formHandler data.FormCallback, df interfaces.DialerFactory, verifier tls.Verifier, c *config.ApplicationConfig) error {
	_, xmppLog, _ := session.CreateXMPPLogger(c.RawLogFile)
	ll := log.StandardLogger().WithFields(log.Fields{
		"server":    server,
		"component": "registration",
	})
	policy := config.ConnectionPolicy{DialerFactory: df, XMPPLogger: xmppLog, Logger: ll.Writer(), Log: ll}

	//TODO: this would not be necessary if RegisterAccount did not use it
	//TODO: we should give the choice of using Tor to the user
	conf := &config.Account{
		Account: "@" + server,
		Proxies: []string{"tor-auto://"},
	}

	//TODO: this should receive only a JID domainpart
	conn, err := policy.RegisterAccount(formHandler, conf, verifier)
	if conn != nil {
		defer func() {
			_ = conn.Close()
		}()
	}

	return err
}

func renderError(message gtki.Label, errorMessage, logMessage string, err error, l withLog) {
	l.Log().WithError(err).Warn(logMessage)
	message.SetLabel(errorMessage)
}

func (w *serverSelectionWindow) renderConnectionErrorFor(err error) {
	w.spinner.Stop()
	setImageFromFile(w.formImage, "failure.svg")

	switch err {

	case ourNet.ErrTimeout:
		renderError(w.formMessage, i18n.Local("We had an error:\n\nTimeout."), "Error when trying to get registration form", err, w.u)
	case config.ErrTorNotRunning:
		renderError(w.formMessage, i18n.Local("The registration process currently requires Tor in order to ensure your safety.\n\n"+
			"You don't have Tor running. Please, start it.\n\n"), "We had an error when trying to register your account: Tor is not running.", err, w.u)
	default:
		renderError(w.formMessage, i18n.Local("Could not contact the server.\n\nPlease, correct your server choice and try again."), "Error when trying to get registration form", err, w.u)
	}
}

func (w *serverSelectionWindow) renderErrorFor(err error) {
	setImageFromFile(w.doneImage, "failure.svg")

	switch err {
	case xmpp.ErrMissingRequiredRegistrationInfo:
		renderError(w.doneMessage, i18n.Local("We had an error:\n\nSome required fields are missing. Please, try again and fill all fields."), "Error when trying to get registration form", err, w.u)
	case xmpp.ErrUsernameConflict:
		renderError(w.doneMessage, i18n.Local("We had an error:\n\nIncorrect username"), "We had an error when trying to create your account", err, w.u)
	case xmpp.ErrWrongCaptcha:
		renderError(w.doneMessage, i18n.Local("We had an error:\n\nThe captcha entered is wrong"), "We had an error when trying to create your account", err, w.u)
	case xmpp.ErrResourceConstraint:
		renderError(w.doneMessage, i18n.Local("We had an error:\n\nThe server received too many requests to create an account at the same time."), "We had an error when trying to create your account", err, w.u)
	default:
		renderError(w.doneMessage, i18n.Local("Could not contact the server.\n\nPlease, correct your server choice and try again."), "Error when trying to get registration form", err, w.u)
	}
}

type serverSelectionWindow struct {
	b           *builder
	assistant   gtki.Assistant    `gtk-widget:"assistant"`
	formMessage gtki.Label        `gtk-widget:"formMessage"`
	doneMessage gtki.Label        `gtk-widget:"doneMessage"`
	serverBox   gtki.ComboBoxText `gtk-widget:"server"`
	spinner     gtki.Spinner      `gtk-widget:"spinner"`
	grid        gtki.Grid         `gtk-widget:"formGrid"`
	formImage   gtki.Image        `gtk-widget:"formImage"`
	doneImage   gtki.Image        `gtk-widget:"doneImage"`

	formSubmitted chan error
	done          chan error

	form *registrationForm

	u *gtkUI
}

func createServerSelectionWindow(u *gtkUI) *serverSelectionWindow {
	w := &serverSelectionWindow{b: newBuilder("AccountRegistration"), u: u}

	panicOnDevError(w.b.bindObjects(w))

	w.assistant.SetTransientFor(u.window)

	w.formSubmitted = make(chan error)
	w.done = make(chan error)

	w.form = &registrationForm{grid: w.grid, l: u}

	return w
}

func (w *serverSelectionWindow) initializeServers() {
	for _, s := range servers.GetServersForRegistration() {
		w.serverBox.AppendText(s.Name)
	}
	w.serverBox.SetActive(0)
}

func (w *serverSelectionWindow) initialPage() {
	w.serverBox.SetSensitive(true)
	w.form.server = ""

	w.grid.Hide()
}

func (w *serverSelectionWindow) renderForm(pg gtki.Widget) func(string, string, []interface{}) error {
	return func(title, instructions string, fields []interface{}) error {
		w.spinner.Stop()
		w.formMessage.SetLabel("")
		w.doneMessage.SetLabel("")

		_ = w.form.renderForm(title, fields)
		w.assistant.SetPageComplete(pg, true)

		return <-w.formSubmitted
	}
}

func (w *serverSelectionWindow) doRendering(pg gtki.Widget) {
	err := requestAndRenderRegistrationForm(w.form.server, w.renderForm(pg), w.u.dialerFactory, w.u.unassociatedVerifier(), w.u.config())
	if err != nil {
		w.u.hasLog.log.WithError(err).Debug("error while rendering registration form")
		// TODO: refactor me!
		if err == config.ErrTorNotRunning || err == xmppErr.ErrAuthenticationFailed || err == xmpp.ErrRegistrationFailed || err == ourNet.ErrTimeout {
			w.assistant.SetPageType(pg, gtki.ASSISTANT_PAGE_SUMMARY)
			w.assistant.SetPageComplete(pg, true)
			w.renderConnectionErrorFor(err)
			return
		}
	}

	go w.assistant.SetCurrentPage(2)

	w.done <- err
}

func (w *serverSelectionWindow) serverChosenPage(pg gtki.Widget) {
	w.serverBox.SetSensitive(false)
	w.form.server = w.serverBox.GetActiveText()
	w.spinner.Start()

	w.formMessage.SetLabel(i18n.Local("Connecting to server for registration... \n\n" +
		"This might take a while."))

	go w.doRendering(pg)
}

func (w *serverSelectionWindow) formSubmittedPage() {
	w.grid.Show()
	w.formSubmitted <- w.form.accepted()

	err := <-w.done
	w.spinner.Stop()

	if err != nil {
		w.u.hasLog.log.WithError(err).Debug("error for submitted page")
		w.renderErrorFor(err)
		return
	}

	// Save the account
	err = w.u.addAndSaveAccountConfig(w.form.conf)

	if err != nil {
		w.u.hasLog.log.WithError(err).Debug("error when adding or saving account config")
		renderError(w.doneMessage, i18n.Local("We had an error when trying to store your account information."), "We had an error when trying to store your account information. Please, try again.", err, w.u)
		return
	}

	if acc, ok := w.u.getAccountByID(w.form.conf.ID()); ok {
		acc.Connect()
	}

	setImageFromFile(w.doneImage, "success.svg")
	w.doneMessage.SetMarkup(i18n.Localf("<b>%s</b> successfully created.", w.form.conf.Account))
}

func (w *serverSelectionWindow) onPageChange(_ gtki.Assistant, pg gtki.Widget) {
	switch w.assistant.GetCurrentPage() {
	case 0:
		w.initialPage()
	case 1:
		w.serverChosenPage(pg)
	case 2:
		w.formSubmittedPage()
	}
}

func (u *gtkUI) showServerSelectionWindow() {
	w := createServerSelectionWindow(u)
	w.initializeServers()

	w.b.ConnectSignals(map[string]interface{}{
		"on_prepare": w.onPageChange,
		"on_cancel":  w.assistant.Destroy,
	})

	entry := w.serverBox.GetChild().(gtki.Widget)
	_ = entry.Connect("activate", func() {
		w.assistant.SetCurrentPage(1)
	})

	w.assistant.ShowAll()
}

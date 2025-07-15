package gui

import (
	"github.com/chadsec1/decoyim/i18n"
	"github.com/chadsec1/decoyim/net"
	"github.com/coyim/gotk3adapter/gtki"
)

var proxyTypes = [][]string{
	{"tor-auto", "Automatic Tor"},
	{"socks5", "SOCKS5"},
	{"socks5+unix", "SOCKS5 Unix Domain"},
}

func init() {
	// Force translation of these strings
	_ = i18n.Local("Automatic Tor")
	_ = i18n.Local("SOCKS5")
	_ = i18n.Local("SOCKS5 Unix Domain")
}

// findProxyTypeFor returns the index of the proxy type given
func findProxyTypeFor(s string) int {
	for ix, px := range proxyTypes {
		if px[0] == s {
			return ix
		}
	}

	return -1
}

// getProxyTypeNames will yield all i18n proxy names to the function
func getProxyTypeNames(f func(string)) {
	for _, px := range proxyTypes {
		f(i18n.Local(px[1]))
	}
}

// getProxyTypeFor will return the proxy type for the given i18n proxy name
func getProxyTypeFor(act string) string {
	for _, px := range proxyTypes {
		if act == i18n.Local(px[1]) {
			return px[0]
		}
	}
	return ""
}

func orNil(s string) *string {
	if s != "" {
		return &s
	}
	return nil
}

func updateSensitivity(v bool, es ...gtki.Widget) {
	for _, ee := range es {
		ee.SetSensitive(v)
	}
}

func (u *gtkUI) editProxy(proxy string, w gtki.Dialog, onSave func(net.Proxy), onCancel func()) {
	prox := net.ParseProxy(proxy)

	b := newBuilder("EditProxy")
	dialog := b.getObj("EditProxy").(gtki.Dialog)
	scheme := b.getObj("protocol-type").(gtki.ComboBoxText)
	user := b.getObj("user").(gtki.Entry)
	pass := b.getObj("password").(gtki.Entry)
	server := b.getObj("server").(gtki.Entry)
	serverLabel := b.getObj("serverLabel").(gtki.Label)
	port := b.getObj("port").(gtki.Entry)
	portLabel := b.getObj("portLabel").(gtki.Label)
	path := b.getObj("path").(gtki.Entry)
	pathLabel := b.getObj("pathLabel").(gtki.Label)

	getProxyTypeNames(func(name string) {
		scheme.AppendText(name)
	})
	scheme.SetActive(findProxyTypeFor(prox.Scheme))

	if prox.User != nil {
		user.SetText(*prox.User)
	}

	if prox.Pass != nil {
		pass.SetText(*prox.Pass)
	}

	if prox.Host != nil {
		server.SetText(*prox.Host)
	}

	if prox.Port != nil {
		port.SetText(*prox.Port)
	}

	if prox.Path != nil {
		path.SetText(*prox.Path)
	}

	isUD := getProxyTypeFor(scheme.GetActiveText()) == "socks5+unix"
	updateSensitivity(isUD, path, pathLabel)
	updateSensitivity(!isUD, server, serverLabel, port, portLabel)

	b.ConnectSignals(map[string]interface{}{
		"on_protocol_type_changed": func() {
			isUD := getProxyTypeFor(scheme.GetActiveText()) == "socks5+unix"
			updateSensitivity(isUD, path, pathLabel)
			updateSensitivity(!isUD, server, serverLabel, port, portLabel)
		},
		"on_save": func() {
			userTxt, _ := user.GetText()
			passTxt, _ := pass.GetText()
			servTxt, _ := server.GetText()
			portTxt, _ := port.GetText()
			pathTxt, _ := path.GetText()

			prox.Scheme = getProxyTypeFor(scheme.GetActiveText())
			isUD := prox.Scheme == "socks5+unix"

			prox.User = orNil(userTxt)
			prox.Pass = orNil(passTxt)
			if isUD {
				prox.Path = orNil(pathTxt)
			} else {
				prox.Host = orNil(servTxt)
				prox.Port = orNil(portTxt)
			}

			go onSave(prox)
			dialog.Destroy()
		},
		"on_cancel": func() {
			go onCancel()
			dialog.Destroy()
		},
	})

	dialog.SetTransientFor(w)
	dialog.ShowAll()
}

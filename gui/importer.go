package gui

import (
	"bufio"
	"fmt"
	"os"
	"sort"

	log "github.com/sirupsen/logrus"

	"github.com/coyim/coyim/config"
	"github.com/coyim/coyim/config/importer"
	"github.com/coyim/coyim/i18n"
	"github.com/coyim/gotk3adapter/gtki"
	"github.com/coyim/otr3"
)

func valAt(s gtki.ListStore, iter gtki.TreeIter, col int) interface{} {
	gv, _ := s.GetValue(iter, col)
	vv, _ := gv.GoValue()
	return vv
}

type applicationAndAccount struct {
	app string
	acc string
}

// TODO: refactor
func (u *gtkUI) doActualImportOf(choices map[applicationAndAccount]bool, potential map[string][]*config.ApplicationConfig) {
	for k, v := range choices {
		if !v {
			continue
		}

		for _, accs := range potential[k.app] {
			for _, a := range accs.Accounts {
				if a.Account == k.acc {
					u.hasLog.log.WithFields(log.Fields{
						"feature":       "import",
						"importAccount": k.acc,
						"importApp":     k.app,
					}).Info("Doing import")
					u.config().WhenLoaded(func(conf *config.ApplicationConfig) {
						_, exists := conf.GetAccount(k.acc)
						if exists {
							u.hasLog.log.WithFields(log.Fields{
								"feature":       "import",
								"importAccount": k.acc,
							}).Warn("Can't import account since you already have an account " +
								"configured with the same name. Remove that account and import again if you " +
								"really want to overwrite it.")
							u.notify(i18n.Local("Unable to Import Account"), i18n.Localf("Can't import account:\n\n"+
								"You already have an account with this name."))
							return
						}

						if conf.RawLogFile == "" {
							conf.RawLogFile = accs.RawLogFile
						}
						if len(conf.NotifyCommand) == 0 {
							conf.NotifyCommand = accs.NotifyCommand
						}
						if conf.IdleSecondsBeforeNotification == 0 {
							conf.IdleSecondsBeforeNotification = accs.IdleSecondsBeforeNotification
						}
						if !conf.Bell {
							conf.Bell = accs.Bell
						}

						err := u.addAndSaveAccountConfig(a)
						if err != nil {
							// TODO: maybe show a message with error
							return
						}
					})
				}
			}
		}
	}
}

func (u *gtkUI) runImporter() {
	importSettings := make(map[applicationAndAccount]bool)
	allImports := importer.TryImportAll()

	builder := newBuilder("Importer")

	win := builder.getObj("importerWindow")
	w := win.(gtki.Dialog)

	store := builder.getObj("importAccountsStore")
	s := store.(gtki.ListStore)

	for appName, v := range allImports {
		for _, vv := range v {
			for _, a := range vv.Accounts {
				it := s.Append()
				_ = s.SetValue(it, 0, appName)
				_ = s.SetValue(it, 1, a.Account)
				_ = s.SetValue(it, 2, false)
			}
		}
	}

	rend := builder.getObj("import-this-account-renderer")
	rr := rend.(gtki.CellRendererToggle)

	_ = rr.Connect("toggled", func(_ interface{}, path string) {
		iter, _ := s.GetIterFromString(path)
		current, _ := valAt(s, iter, 2).(bool)
		app, _ := valAt(s, iter, 0).(string)
		acc, _ := valAt(s, iter, 1).(string)

		importSettings[applicationAndAccount{app, acc}] = !current

		_ = s.SetValue(iter, 2, !current)
	})

	_ = w.Connect("response", func(_ interface{}, rid int) {
		if gtki.ResponseType(rid) == gtki.RESPONSE_OK {
			u.doActualImportOf(importSettings, allImports)
		}
		w.Destroy()
	})

	u.connectShortcutsChildWindow(w)
	doInUIThread(func() {
		w.SetTransientFor(u.mainUI.window)
		w.ShowAll()
	})
}

func (u *gtkUI) importFingerprintsFor(account *config.Account, file string) (int, bool) {
	fprs, ok := importer.ImportFingerprintsFromPidginStyle(file, func(string) bool { return true })
	if !ok {
		return 0, false
	}

	num := 0
	for _, kfprs := range fprs {
		for _, kfpr := range kfprs {
			u.hasLog.log.WithFields(log.Fields{
				"feature":     "import",
				"fingerprint": fmt.Sprintf("%X", kfpr.Fingerprint),
				"user":        kfpr.UserID,
			}).Info("Importing fingerprint")
			fpr, _ := account.EnsurePeer(kfpr.UserID).EnsureHasFingerprint(kfpr.Fingerprint)
			num = num + 1
			if !kfpr.Untrusted {
				fpr.Trusted = true
			}
		}
	}

	return num, true
}

func firstItem(mm map[string][]byte) []byte {
	for _, v := range mm {
		return v
	}
	return nil
}

func sortKeys(keys map[string]otr3.PrivateKey) []string {
	res := []string{}

	for k := range keys {
		res = append(res, k)
	}

	sort.Strings(res)

	return res
}

func (u *gtkUI) chooseKeyToImport(keys map[string][]byte) ([]byte, bool) {
	result := make(chan int)
	parsedKeys := make(map[string]otr3.PrivateKey)
	for v, vv := range keys {
		_, ok, parsedKey := otr3.ParsePrivateKey(vv)
		if ok {
			parsedKeys[v] = parsedKey
		}
	}
	sortedKeys := sortKeys(parsedKeys)

	doInUIThread(func() {
		builder := newBuilder("ChooseKeyToImport")
		d := builder.getObj("dialog").(gtki.Dialog)
		d.SetTransientFor(u.mainUI.window)
		keyBox := builder.getObj("keys").(gtki.ComboBoxText)

		for _, s := range sortedKeys {
			kval := parsedKeys[s]
			keyBox.AppendText(fmt.Sprintf("%s -- %s", s, config.FormatFingerprint(kval.PublicKey().Fingerprint())))
		}

		keyBox.SetActive(0)

		builder.ConnectSignals(map[string]interface{}{
			"on_import": func() {
				ix := keyBox.GetActive()
				if ix != -1 {
					result <- ix
					close(result)
					d.Destroy()
				}
			},

			"on_cancel": func() {
				close(result)
				d.Destroy()
			},
		})

		d.ShowAll()
	})

	res, ok := <-result
	if ok {
		return keys[sortedKeys[res]], true
	}
	return nil, false
}

func (u *gtkUI) importKeysFor(account *config.Account, file string) (int, bool) {
	keys, ok := importer.ImportKeysFromPidginStyle(file, func(string) bool { return true })
	if !ok {
		return 0, false
	}

	switch len(keys) {
	case 0:
		return 0, true
	case 1:
		account.PrivateKeys = [][]byte{firstItem(keys)}
		return 1, true
	default:
		kk, ok := u.chooseKeyToImport(keys)
		if ok {
			account.PrivateKeys = [][]byte{kk}
			return 1, true
		}
		return 0, false
	}
}

func (u *gtkUI) exportFingerprintsFor(account *config.Account, file string) bool {
	f, err := os.Create(file)
	if err != nil {
		return false
	}
	defer func() {
		_ = f.Close()
	}()
	bw := bufio.NewWriter(f)

	for _, p := range account.Peers {
		for _, fpr := range p.Fingerprints {
			trusted := ""
			if fpr.Trusted {
				trusted = "\tverified"
			}
			_, _ = bw.WriteString(fmt.Sprintf("%s\t%s/\tprpl-jabber\t%x%s\n", p.UserID, account.Account, fpr.Fingerprint, trusted))
		}
	}

	_ = bw.Flush()
	return true
}

func (u *gtkUI) exportKeysFor(account *config.Account, file string) bool {
	var result []*otr3.Account

	allKeys := account.AllPrivateKeys()

	for _, pp := range allKeys {
		_, ok, parsedKey := otr3.ParsePrivateKey(pp)
		if ok {
			result = append(result, &otr3.Account{
				Name:     account.Account,
				Protocol: "prpl-jabber",
				Key:      parsedKey,
			})
		}
	}

	err := otr3.ExportKeysToFile(result, file)
	return err == nil
}

func (u *gtkUI) importKeysForDialog(account *config.Account, w gtki.Dialog) {
	dialog, _ := g.gtk.FileChooserDialogNewWith2Buttons(
		i18n.Local("Import private keys"),
		w,
		gtki.FILE_CHOOSER_ACTION_OPEN,
		i18n.Local("_Cancel"),
		gtki.RESPONSE_CANCEL,
		i18n.Local("_Import"),
		gtki.RESPONSE_OK,
	)

	if gtki.ResponseType(dialog.Run()) == gtki.RESPONSE_OK {
		fname := dialog.GetFilename()
		go func() {
			_, ok := u.importKeysFor(account, fname)
			if ok {
				u.notify(i18n.Local("Keys imported"), i18n.Local("The key was imported correctly."))
			} else {
				u.notify(i18n.Local("Failure importing keys"), i18n.Localf("Couldn't import any keys from %s.", fname))
			}
		}()
	}
	dialog.Destroy()
}

func (u *gtkUI) exportKeysForDialog(account *config.Account, w gtki.Dialog) {
	dialog, _ := g.gtk.FileChooserDialogNewWith2Buttons(
		i18n.Local("Export private keys"),
		w,
		gtki.FILE_CHOOSER_ACTION_SAVE,
		i18n.Local("_Cancel"),
		gtki.RESPONSE_CANCEL,
		i18n.Local("_Export"),
		gtki.RESPONSE_OK,
	)

	dialog.SetCurrentName("otr.private_key")

	if gtki.ResponseType(dialog.Run()) == gtki.RESPONSE_OK {
		ok := u.exportKeysFor(account, dialog.GetFilename())
		if ok {
			u.notify(i18n.Local("Keys exported"), i18n.Local("Keys were exported correctly."))
		} else {
			u.notify(i18n.Local("Failure exporting keys"), i18n.Localf("Couldn't export keys to %s.", dialog.GetFilename()))
		}
	}
	dialog.Destroy()
}

func (u *gtkUI) importFingerprintsForDialog(account *config.Account, w gtki.Dialog) {
	dialog, _ := g.gtk.FileChooserDialogNewWith2Buttons(
		i18n.Local("Import fingerprints"),
		w,
		gtki.FILE_CHOOSER_ACTION_OPEN,
		i18n.Local("_Cancel"),
		gtki.RESPONSE_CANCEL,
		i18n.Local("_Import"),
		gtki.RESPONSE_OK,
	)

	if gtki.ResponseType(dialog.Run()) == gtki.RESPONSE_OK {
		num, ok := u.importFingerprintsFor(account, dialog.GetFilename())
		if ok {
			u.notify(i18n.Local("Fingerprints imported"), i18n.Localf("%d fingerprint(s) were imported correctly.", num))
		} else {
			u.notify(i18n.Local("Failure importing fingerprints"), i18n.Localf("Couldn't import any fingerprints from %s.", dialog.GetFilename()))
		}
	}
	dialog.Destroy()
}

func (u *gtkUI) exportFingerprintsForDialog(account *config.Account, w gtki.Dialog) {
	dialog, _ := g.gtk.FileChooserDialogNewWith2Buttons(
		i18n.Local("Export fingerprints"),
		w,
		gtki.FILE_CHOOSER_ACTION_SAVE,
		i18n.Local("_Cancel"),
		gtki.RESPONSE_CANCEL,
		i18n.Local("_Export"),
		gtki.RESPONSE_OK,
	)

	dialog.SetCurrentName("otr.fingerprints")

	if gtki.ResponseType(dialog.Run()) == gtki.RESPONSE_OK {
		ok := u.exportFingerprintsFor(account, dialog.GetFilename())
		if ok {
			u.notify(i18n.Local("Fingerprints exported"), i18n.Local("Fingerprints were exported correctly."))
		} else {
			u.notify(i18n.Local("Failure exporting fingerprints"), i18n.Localf("Couldn't export fingerprints to %s.", dialog.GetFilename()))
		}
	}
	dialog.Destroy()
}

# DecoyIM – Secure XMPP Chat Client

[![Build](https://github.com/chadsec1/decoyim/workflows/DecoyIM%20CI/badge.svg)](https://github.com/chadsec1/decoyim/actions)
[![Coverage](https://coveralls.io/repos/coyim/coyim/badge.svg?branch=main)](https://coveralls.io/github/chadsec1/decoyim?branch=main)
[![Go Report](https://goreportcard.com/badge/github.com/chadsec1/decoyim)](https://goreportcard.com/report/github.com/chadsec1/decoyim)

<p align="center">
  <img src="build/osx/mac-bundle/coyim.iconset/icon_256x256.png" height="128">
</p>

**DecoyIM** (a fork of CoyIM) is a secure, privacy-first XMPP chat client built in Go with a minimalist GTK UI. It defaults to strong, sane security settings—**OTR encryption**, **Tor routing**, and **TLS validation**—with no setup required.

We aim to make encrypted chat accessible and safe, even for high-risk users.

---

## 🚨 Security Warning

OTR3 code has been audited; the rest hasn't. Use with caution if you're a high-risk target.

All binary releases are compiled on Github servers, and not our machines. This ensure highest security in event developers are compromised

---

## 🧪 Quick Start

- 📦 [Download latest release binaries](https://github.com/chadsec1/decoyim/releases) or build from source.

First launch shows a setup wizard. 
<p align="left">
  <img src="images/wizard.png" height="242" width="242">
</p>

You can:

- Import Jabber/OTR accounts from other clients (like Pidgin via `~/.purple`)
- Skip it and import later via `Accounts → Import`

<p align="left">
  <img src="images/main_window.png">
</p>


---

## ⚙️ Building

Requires:
- Go ≥ 1.19
- GTK+ ≥ 3.12

### Ubuntu
```bash
sudo apt-get install libgtk-3-dev ruby-full
```

### MacOS
```sh
brew install gnome-icon-theme
brew install gtk+3 gtk-mac-integration
```


In order to build DecoyIM, you should check out the source code, and run:

```sh
make deps
make build
```


It might be possible to build DecoyIM using `go get` but we currently do not support this method.

NOTE: If `esc` isn't found after `make deps`, it's likely in `$HOME/go/bin`.  
Run `export PATH="$HOME/go/bin:$PATH"` to fix it.


## Contributing to DecoyIM

We have instructions here on how you [can get started contributing to CoyIM](CONTRIBUTING.md).


## Reproducibility

DecoyIM supports reproducible builds for Linux on AMD64. See [REPRODUCIBILITY](REPRODUCIBILITY.md) for instructions on how
to build and verify these builds.


## License

The DecoyIM project and all source files licensed under the [GPL version 3](https://www.gnu.org/licenses/gpl-3.0.html),
unless otherwise noted.

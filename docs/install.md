<!--
SPDX-FileCopyrightText: 2021 Markus Sommer

SPDX-License-Identifier: GPL-3.0-or-later
-->

# Installation Instructions

## Manual

- Install the *Go Programming language*, either through your package manager, or from [here](https://golang.org/dl/).
- Clone the [source repository](https://github.com/dtn7/dtn7-go).
- Check out the most recent release tag. Or don't and just build the `HEAD`, but we don't promise that it won't be broken.
- (Optional) Run `GO111MODULE=on go test ./...` in the repository root to make sur we didn't screw up too badly.
- Run `GO111MODULE=on go build ./...` in the repositry root to build both `dtnd` and `dtn-tool`

If you want to run `dtnd` as a service, you might also want to do the following:

- Put the config-file (`cmd/dtnd/configuration.toml`) in `/etc/dtn7`
- Put the systemd-service (`contrib/systemd/service/dtn7.service`) in `/etc/systemd/system`
- Probably run `systemctl daemon-reload` for good measure
- Create the working directory `/var/lib/dtn7`
- Create a `dtn7` user like this: `sudo useradd -r -s /sbin/nologin -d /var/lib/dtn7 dtn7`
- Set ownership of the working directory `chown dtn7:dtn7 /var/lib/dtn7`
- Start the service: `systemctl start dtn7`

If you install the arch-package, we do all of that for you.

## Automatic

For some select platforms, packages are provided. (Footnote: ''select'' in this case means those platforms that don't make it 
prohibitively complicated to package for them - looking at you, Debian)

### NixOS

If you're the kind of person who uses Nix, I assume you know how to install software on your system.
Because I most certainl don't.

### Arch Linux

Install it from the [AUR](https://aur.archlinux.org/packages/dtn7) either manually, or using your favourite aur-helper.

The package also installs the config file & systemd-service.
Also takes care of directory & user creation.

Start daemon using `systemctl start dtn7`

### Other Linux

We would like to provide packages for all distributions, however we're not really sure how.
Attempts at using the *Open SUSE Build Service* were unsuccessfull, since the build VMs don't have internet access and therefore can't get the dependencies from `go.mod`.

If you know how to (automatically) build and packe go applications for other distros, please contact us.

### MacOS

Use [brew](https://github.com/jonashoechst/homebrew-hoechst/blob/master/dtn7.rb), I guess.
If there's anything special about doing it, someone should probably describe that here.

<!--
SPDX-FileCopyrightText: 2021 Markus Sommer
SPDX-FileCopyrightText: 2021 Alvar Penning

SPDX-License-Identifier: GPL-3.0-or-later
-->

# Installation Instructions

## Manual

- Install the *Go Programming language*, either through your package manager, or from [here](https://golang.org/dl/).
- Clone the [source repository](https://github.com/dtn7/dtn7-go).
- Check out the most recent release tag. Or don't and just build the `HEAD`, but we don't promise that it won't be broken.
- (Optional) Run `go test ./...` in the repository root to make sure we didn't screw up too badly.
- Run `go build ./cmd/dtnd && go build ./cmd/dtn-tool` in the repositry root to build both `dtnd` and `dtn-tool`

If you want to run `dtnd` as a service, you might also want to do the following:

- Put the config-file (`cmd/dtnd/configuration.toml`) in `/etc/dtn7`
- Put the systemd-service (`contrib/systemd/service/dtn7.service`) in `/etc/systemd/system`
- Probably run `systemctl daemon-reload` for good measure
- Create the working directory `/var/lib/dtn7`
- Create a `dtn7` user like this: `useradd -r -s /sbin/nologin -d /var/lib/dtn7 dtn7`
- Set ownership of the working directory `chown dtn7:dtn7 /var/lib/dtn7`
- Start the service: `systemctl start dtn7`

If you install the arch-package, we do all of that for you.

## Automatic

For some select platforms, packages are provided. (Footnote: ''select'' in this case means those platforms that don't make it 
prohibitively complicated to package for them - looking at you, Debian)

### Nix / NixOS

Nix is a purely functional package manager for Linux or macOS.
NixOS is a Linux distribution based on Nix.

There is a [dtn7 Nix User Repository (NUR)](https://github.com/dtn7/nur-packages) which contains two versions of dtn7-go.

- The `dtn7-go` package for the latest release,
- and the `dtn7-go-unstbale` package which is always the latest `master` branch's `HEAD`.

Both packages are automatically bumped after changes in the dtn7-go repository.

You can import and install one of those dtn7 packages as described in the [dtn7 NUR's README](https://github.com/dtn7/nur-packages).
Alternatively, all [NURs](https://github.com/nix-community/NUR) can be included and a dtn7-go version installed from those.

### Arch Linux

Install it from the [AUR](https://aur.archlinux.org/packages/dtn7) either manually, or using your favourite aur-helper.

The package also installs the config file & systemd-service.
Also takes care of directory & user creation.

Start daemon using `systemctl start dtn7`

### Other Linux

We would like to provide packages for all distributions, however we're not really sure how.
Attempts at using the *Open SUSE Build Service* were unsuccessful, since the build VMs don't have internet access and therefore can't get the dependencies from `go.mod`.

If you know how to (automatically) build and package go applications for other distributions or package managers, please contact us.

### macOS

We provide a package for macOS through [Homebrew](https://brew.sh). To install from the [provided package](https://github.com/dtn7/homebrew-dtn7): 
```
$ brew install dtn7/dtn7/dtn7-gp
```
A configuration file (`/usr/local/etc/dtn7-go/configuration.toml`) will be created, as well as a brew services / launchd compatible service file will be created. dtn7-d's store, as well as the runtime logs will appear in `/usr/local/var/dtn7-go/`.

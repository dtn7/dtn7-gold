<!--
SPDX-FileCopyrightText: 2020 Markus Sommer

SPDX-License-Identifier: GPL-3.0-or-later
-->

# `systemd` service unit

Use this to run `dtn7` as a systemd-service.

Place this file in either `/usr/lib/systemd/system/` if you are building a package, or in `/etc/systemd/system/` if you are installing manually.

If installing manually, you might need to run `systemctl daemon-reload` before starting the service.

The service expects the following things:

- The `dtnd` binary installed to `/usr/bin/`
- An existing user & group `dtn7:dtn7`
- An existing working directory in `/var/lib/dtn7`
- Configuration in `/etc/dtn7/configuration.toml`

<!--
SPDX-FileCopyrightText: 2020 Markus Sommer

SPDX-License-Identifier: GPL-3.0-or-later
-->

# Declare system user & group

The included `systemd` service expects an existing user & group `dtn7:dtn7` to run. `sysusers.d` allows for declarative creation of users.

Place in `/usr/lib/sysusers.d/` if packaging, or in `/etc/sysusers.d/` if installing manually.

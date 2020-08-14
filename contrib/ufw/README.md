<!--
SPDX-FileCopyrightText: 2020 Markus Sommer

SPDX-License-Identifier: GPL-3.0-or-later
-->

# `ufw` Macro

If your system is using [ufw](https://launchpad.net/ufw) as its firewall, you can use this macro to allow the required ports.

To do so, place this file in `/etc/ufw/applications.d/` and run `ufw allow dtn7` as root.

NOTE: this macro only allows the ports for peer-discovery and the default `TCPCL`. If you are running different/additional CLAs, you need to add the respective ports to the macro.
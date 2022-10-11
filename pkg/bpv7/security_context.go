// SPDX-FileCopyrightText: 2020 Matthias Axel Kröll
// SPDX-FileCopyrightText: 2020 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package bpv7

// Sorted list of all known security contexts identifiers.
const (
	// SecConIdentBIBIOPHMACSHA BIB-IOP-HMAC-SHA as described in draft-ietf-dtn-bpsec-interop-sc-01#section-3 .
	SecConIdentBIBIOPHMACSHA uint64 = 0

	// SecConIdentBCBIOPAESGCM BCB-IOP-AES-GCM-256 as described in  draft-ietf-dtn-bpsec-interop-sc-01#section-3 .
	SecConIdentBCBIOPAESGCM uint64 = 1
)

// Sorted list of all known security context names.
const (
	// SecConNameBIBIOPHMACSHA BIB-IOP-HMAC-SHA as described in draft-ietf-dtn-bpsec-interop-sc-01#section-3 .
	SecConNameBIBIOPHMACSHA string = "BIB-HMAC-SHA2"

	// SecConNameBCBIOPAESGCM BCB-IOP-AES-GCM as described in  draft-ietf-dtn-bpsec-interop-sc-01#section-3 .
	SecConNameBCBIOPAESGCM string = "BCB-IOP-AES-GCM"
)

// SPDX-FileCopyrightText: 2022 Matthias Axel Kröll
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"io"
	"os"

	"github.com/dtn7/dtn7-go/pkg/bpv7"
	log "github.com/sirupsen/logrus"
)

// signBundle for the "sign" CLI option.
func signBundle(args []string) {
	if len(args) != 3 {
		printUsage()
	}

	var (
		input  = args[0]
		psk    = args[1]
		output = args[2]
		err    error
		f      io.ReadCloser
		b      bpv7.Bundle
	)

	if input == "-" {
		f = os.Stdin
	} else if f, err = os.Open(input); err != nil {
		printFatal(err, "Opening file for reading erred")
	}

	if err = b.UnmarshalCbor(f); err != nil {
		printFatal(err, "Unmarshaling Bundle erred")
	}
	if err = f.Close(); err != nil {
		printFatal(err, "Closing file erred")
	}

	payloadSecurityTarget, _ := b.ExtensionBlock(bpv7.ExtBlockTypePayloadBlock)

	securityTargets := []uint64{payloadSecurityTarget.BlockNumber}

	shaVariant := bpv7.HMAC256SHA256

	bib := bpv7.NewBIBIOPHMACSHA2(&shaVariant, nil, nil, securityTargets, b.PrimaryBlock.SourceNode)

	eb := bpv7.CanonicalBlock{
		BlockNumber:       0,
		BlockControlFlags: 0,
		CRCType:           bpv7.CRCNo,
		CRC:               nil,
		Value:             bib,
	}

	err = b.AddExtensionBlock(eb)
	if err != nil {
		printFatal(err, "Add Extension Block failed")

	}

	bibBlockAdded, _ := b.ExtensionBlock(bib.BlockTypeCode())

	err = bibBlockAdded.Value.(*bpv7.BIBIOPHMACSHA2).SignTargets(b, bibBlockAdded.BlockNumber, []byte(psk))
	if err != nil {
		printFatal(err, "Signing Targets erred")
	}

	logger := log.WithFields(log.Fields{
		"bundle": b.ID(),
		"file":   output,
	})

	if f, err := os.Create(output); err != nil {
		logger.WithError(err).Error("Creating file erred")
	} else if err := b.MarshalCbor(f); err != nil {
		logger.WithError(err).Error("Marshalling Bundle erred")
	} else if err := f.Close(); err != nil {
		logger.WithError(err).Error("Closing file erred")
	}

}

// verifyBundle for the "verify" CLI option.
func verifyBundle(args []string) {
	if len(args) != 2 {
		printUsage()
	}

	var (
		input    = args[0]
		psk      = args[1]
		err      error
		f        io.ReadCloser
		b        bpv7.Bundle
		bibBlock *bpv7.CanonicalBlock
	)

	if input == "-" {
		f = os.Stdin
	} else if f, err = os.Open(input); err != nil {
		printFatal(err, "Opening file for reading erred")
	}

	if err = b.UnmarshalCbor(f); err != nil {
		printFatal(err, "Unmarshaling Bundle erred")
	}
	if err = f.Close(); err != nil {
		printFatal(err, "Closing file erred")
	}

	bibBlock, err = b.ExtensionBlock(bpv7.ExtBlockTypeBlockIntegrityBlock)
	if err != nil {
		printFatal(err, "Could not get BIB Extension Block")
	}

	err = bibBlock.Value.(*bpv7.BIBIOPHMACSHA2).VerifyTargets(b, bibBlock.BlockNumber, []byte(psk))
	if err != nil {
		printFatal(err, "Verfication Error")
	}

	log.Info("Verify OK")

}

// encryptBundle for the "sign" CLI option.
func encryptBundle(args []string) {
	if len(args) != 3 {
		printUsage()
	}

	var (
		input  = args[0]
		psk    = args[1]
		output = args[2]
		err    error
		f      io.ReadCloser
		b      bpv7.Bundle
	)

	if input == "-" {
		f = os.Stdin
	} else if f, err = os.Open(input); err != nil {
		printFatal(err, "Opening file for reading erred")
	}

	if err = b.UnmarshalCbor(f); err != nil {
		printFatal(err, "Unmarshaling Bundle erred")
	}
	if err = f.Close(); err != nil {
		printFatal(err, "Closing file erred")
	}

	payloadSecurityTarget, _ := b.ExtensionBlock(bpv7.ExtBlockTypePayloadBlock)

	aesVariant := bpv7.A256GCM

	bcb := bpv7.NewBCBIOPAESGCM(&aesVariant, nil, nil, payloadSecurityTarget.BlockNumber, b.PrimaryBlock.SourceNode)

	eb := bpv7.CanonicalBlock{
		BlockNumber:       0,
		BlockControlFlags: 0,
		CRCType:           bpv7.CRCNo,
		CRC:               nil,
		Value:             bcb,
	}

	err = b.AddExtensionBlock(eb)
	if err != nil {
		printFatal(err, "Add Extension Block failed")

	}

	bcbBlockAdded, _ := b.ExtensionBlock(bcb.BlockTypeCode())

	err = bcbBlockAdded.Value.(*bpv7.BCBIOPAESGCM).EncryptTarget(b, bcbBlockAdded.BlockNumber, []byte(psk))
	if err != nil {
		printFatal(err, "Encrypting Target erred")
	}

	logger := log.WithFields(log.Fields{
		"bundle": b.ID(),
		"file":   output,
	})

	if f, err := os.Create(output); err != nil {
		logger.WithError(err).Error("Creating file erred")
	} else if err := b.MarshalCbor(f); err != nil {
		logger.WithError(err).Error("Marshalling Bundle erred")
	} else if err := f.Close(); err != nil {
		logger.WithError(err).Error("Closing file erred")
	}

}

// decryptBundle for the "verify" CLI option.
func decryptBundle(args []string) {
	if len(args) != 3 {
		printUsage()
	}

	var (
		input   = args[0]
		psk     = args[1]
		output  = args[2]
		err     error
		fInput  io.ReadCloser
		fOutput io.WriteCloser
		b       bpv7.Bundle
	)

	if input == "-" {
		fInput = os.Stdin
	} else if fInput, err = os.Open(input); err != nil {
		printFatal(err, "Opening file for reading erred")
	}

	if err = b.UnmarshalCbor(fInput); err != nil {
		printFatal(err, "Unmarshaling Encrypted Bundle erred")
	}
	if err = fInput.Close(); err != nil {
		printFatal(err, "Closing file erred")
	}

	bcbBlock, err := b.ExtensionBlock(bpv7.ExtBlockTypeBlockConfidentialityBlock)
	if err != nil {
		printFatal(err, "Could not get BIB Extension Block")
	}

	err = bcbBlock.Value.(*bpv7.BCBIOPAESGCM).DecryptTarget(b, bcbBlock.BlockNumber, []byte(psk))
	if err != nil {
		printFatal(err, "Decryption Error")
	}

	b.RemoveExtensionBlockByBlockNumber(bcbBlock.BlockNumber)

	logger := log.WithFields(log.Fields{
		"bundle": b.ID(),
		"file":   output,
	})

	if output == "-" {
		fOutput = os.Stdout
	} else if fOutput, err = os.Create(output); err != nil {
		logger.WithError(err).Error("Creating file errored")
	} else if err := b.MarshalCbor(fOutput); err != nil {
		logger.WithError(err).Error("Marshalling Bundle erred")
	} else if err := fOutput.Close(); err != nil {
		logger.WithError(err).Error("Closing file erred")
	}
}

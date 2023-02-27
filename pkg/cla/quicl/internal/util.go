// SPDX-FileCopyrightText: 2022 Markus Sommer
//
// SPDX-License-Identifier: GPL-3.0-or-later

package internal

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"time"

	"github.com/quic-go/quic-go"

	log "github.com/sirupsen/logrus"
)

// GenerateSimpleListenerTLSConfig generates a bare-bones TLS config for the listener
// This uses a self-signed certificate, so the dialer will have to ignore verification issues
func GenerateSimpleListenerTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.WithError(err).Fatal("Error generating private key")
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		log.WithError(err).Fatal("Error generating certificate")
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.WithError(err).Fatal("Error generating combined certificate")
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"bpv7-quicl"},
		MinVersion:   tls.VersionTLS13,
	}
}

// GenerateSimpleDialerTLSConfig generates a bare-bones TLS config for the dialer
// This configuration assumes that the listener is using a self-signed certificate and thus does not verify it
func GenerateSimpleDialerTLSConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"bpv7-quicl"},
	}
}

func GenerateQUICConfig() *quic.Config {
	return &quic.Config{
		KeepAlivePeriod:    1 * time.Second,
		MaxIdleTimeout:     5 * time.Second,
		EnableDatagrams:    false,
		MaxIncomingStreams: 2048,
	}
}

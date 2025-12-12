// Copyright (C) by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

func getCertificate(certPath string) (tls.Certificate, error) {
	if _, err := os.Stat(certPath); err == nil {
		certPEM, err := os.ReadFile(certPath)
		if err == nil {
			cert, err := tls.X509KeyPair(certPEM, certPEM)
			if err == nil {
				return cert, nil
			}
			appLogger.Printf("Failed to load existing certificate, regenerating: %v", err)
		}
	}

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	notBefore := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	notAfter := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"TAZ"},
		},
		Issuer: pkix.Name{
			Organization: []string{"TAZ"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	out := &bytes.Buffer{}
	if err := pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return tls.Certificate{}, err
	}

	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	if err := pem.Encode(out, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes}); err != nil {
		return tls.Certificate{}, err
	}

	if err := os.WriteFile(certPath, out.Bytes(), 0600); err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to save certificate to %s: %v", certPath, err)
	}

	return tls.X509KeyPair(out.Bytes(), out.Bytes())
}

type bufferedConn struct {
	net.Conn
	peek []byte
}

func (b *bufferedConn) Read(p []byte) (int, error) {
	if len(b.peek) > 0 {
		n := copy(p, b.peek)
		b.peek = b.peek[n:]
		return n, nil
	}
	return b.Conn.Read(p)
}

type muxListener struct {
	net.Listener
	config *tls.Config
}

func (l *muxListener) Accept() (net.Conn, error) {
	for {
		c, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}

		c.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		peek := make([]byte, 1)
		n, err := c.Read(peek)
		c.SetReadDeadline(time.Time{})

		if err != nil {
			c.Close()
			continue
		}

		bc := &bufferedConn{Conn: c, peek: peek[:n]}

		// 0x16 is the handshake byte for TLS
		if n > 0 && peek[0] == 0x16 {
			return tls.Server(bc, l.config), nil
		}

		return bc, nil
	}
}

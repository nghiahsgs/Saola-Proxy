// Package proxy provides CA certificate generation and loading for MITM proxy.
package proxy

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// CADir returns the directory where CA cert/key are stored (~/.saola/).
func CADir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, ".saola"), nil
}

// CACertPath returns the path to the CA certificate file.
func CACertPath() (string, error) {
	dir, err := CADir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ca.crt"), nil
}

// CAKeyPath returns the path to the CA private key file.
func CAKeyPath() (string, error) {
	dir, err := CADir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ca.key"), nil
}

// CAExists checks if both CA cert and key files exist.
func CAExists() bool {
	certPath, err := CACertPath()
	if err != nil {
		return false
	}
	keyPath, err := CAKeyPath()
	if err != nil {
		return false
	}
	_, errC := os.Stat(certPath)
	_, errK := os.Stat(keyPath)
	return errC == nil && errK == nil
}

// GenerateCA creates a new CA certificate and private key, saves to ~/.saola/.
func GenerateCA() (certPath, keyPath string, err error) {
	dir, err := CADir()
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", "", fmt.Errorf("create dir: %w", err)
	}

	// Generate ECDSA P-256 private key.
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate key: %w", err)
	}

	// Create self-signed CA certificate valid for 10 years.
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "Saola Proxy CA",
			Organization: []string{"Saola Proxy"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return "", "", fmt.Errorf("create cert: %w", err)
	}

	// Write certificate PEM.
	certPath = filepath.Join(dir, "ca.crt")
	certFile, err := os.OpenFile(certPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", "", fmt.Errorf("write cert: %w", err)
	}
	defer certFile.Close()
	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return "", "", fmt.Errorf("encode cert: %w", err)
	}

	// Write private key PEM.
	keyPath = filepath.Join(dir, "ca.key")
	keyFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", "", fmt.Errorf("write key: %w", err)
	}
	defer keyFile.Close()
	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return "", "", fmt.Errorf("marshal key: %w", err)
	}
	if err := pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}); err != nil {
		return "", "", fmt.Errorf("encode key: %w", err)
	}

	return certPath, keyPath, nil
}

// LoadCA loads the CA cert/key from ~/.saola/ and returns a tls.Certificate.
func LoadCA() (*tls.Certificate, error) {
	certPath, err := CACertPath()
	if err != nil {
		return nil, err
	}
	keyPath, err := CAKeyPath()
	if err != nil {
		return nil, err
	}

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("load CA keypair: %w", err)
	}

	// Parse the certificate so goproxy can use it as CA.
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("parse CA cert: %w", err)
	}

	return &cert, nil
}

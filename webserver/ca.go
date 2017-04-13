package webserver

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"os"
	"time"
)

func checkCertFilesAreReadable(path string) bool {
	certPath := path + "/cert.pem"
	keyPath := path + "/key.pem"
	var readableCount = 0
	if _, err := os.Stat(certPath); err == nil {
		readableCount = readableCount + 1
	}
	if _, err := os.Stat(keyPath); err == nil {
		readableCount = readableCount + 1
	}

	if readableCount == 2 {
		return true
	} else if readableCount == 1 {
		log.Fatal("FATAL: Weird -- Only one of cert.pem and key.pem are readable")
	}
	// both unreadable, should create 'em
	log.Printf("No TLS cert.pem and key.pem found in %s, will try to create...", path)
	return false
}

func createSelfSignedCert(path string) {
	max := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, max)
	subject := pkix.Name{
		Organization:       []string{"SnakeOil Ltd."},
		OrganizationalUnit: []string{"amtgo"},
		CommonName:         "amtgo self-signed cert",
	}
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}

	pk, _ := rsa.GenerateKey(rand.Reader, 2048)
	derBytes, _ := x509.CreateCertificate(rand.Reader, &template, &template, &pk.PublicKey, pk)
	certOut, _ := os.Create(path + "/cert.pem")
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyOut, _ := os.Create(path + "/key.pem")
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(pk)})
	keyOut.Close()
}

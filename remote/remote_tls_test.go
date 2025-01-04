package remote

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
	"net"
	"sync"
	"testing"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/stretchr/testify/assert"
)

type sharedConfig struct {
	peer1Config *tls.Config
	peer2Config *tls.Config
}

// TestSend_TLS tests sending messages between two remote engines using TLS on the loopback interface.
func TestSend_TLS(t *testing.T) {
	const msgs = 10
	tlsTestConfig, err := generateTLSConfig()
	assert.NoError(t, err)
	aAddr := getRandomLocalhostAddr()
	a, ra, err := makeRemoteEngineTls(aAddr, tlsTestConfig.peer1Config)
	assert.NoError(t, err)
	bAddr := getRandomLocalhostAddr()
	b, rb, err := makeRemoteEngineTls(bAddr, tlsTestConfig.peer2Config)
	assert.NoError(t, err)
	wg := &sync.WaitGroup{}

	wg.Add(msgs) // send msgs messages
	pida := a.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		case *TestMessage:
			assert.Equal(t, msg.Data, []byte("foo"))
			wg.Done()
		}
	}, "actor on a")

	for i := 0; i < msgs; i++ {
		b.Send(pida, &TestMessage{Data: []byte("foo")})
	}
	wg.Add(msgs) // send msgs more messages
	pidb := b.SpawnFunc(func(c *actor.Context) {
		switch msg := c.Message().(type) {
		case *TestMessage:
			assert.Equal(t, msg.Data, []byte("foo"))
			wg.Done()
		}
	}, "actor on b")
	for i := 0; i < msgs; i++ {
		a.Send(pidb, &TestMessage{Data: []byte("foo")})
	}
	wg.Wait()        // wait for messages to be received by the actor.
	ra.Stop().Wait() // shutdown the remotes
	rb.Stop().Wait()
	<-a.Poison(pida).Done()
	<-b.Poison(pidb).Done()
}

func generateTLSConfig() (*sharedConfig, error) {
	// Create a new ECDSA private key for CA
	caPrivKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("ecdsa.GenerateKey: %w", err)
	}

	// Create a CA certificate
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2023),
		Subject: pkix.Name{
			Organization: []string{"Hollywood Testing CA"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, 1),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caCertBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, fmt.Errorf("x509.CreateCertificate: %w", err)
	}

	// Parse the CA certificate for inclusion in tls.Config
	caCert, err := x509.ParseCertificate(caCertBytes)
	if err != nil {
		return nil, fmt.Errorf("x509.ParseCertificate: %w", err)
	}
	// Create the CertPool and add the CA certificate
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCert)

	peer1Pair, err := generateCert(caCert, caPrivKey)
	if err != nil {
		return nil, fmt.Errorf("generateCert(peer1): %w", err)
	}
	peer1TlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*peer1Pair},
		ClientCAs:    caCertPool,
		RootCAs:      caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}
	peer2Pair, err := generateCert(caCert, caPrivKey)
	if err != nil {
		return nil, fmt.Errorf("generateCert(peer2): %w", err)
	}
	peer2TlsConfig := &tls.Config{
		Certificates: []tls.Certificate{*peer2Pair},
		ClientCAs:    caCertPool,
		RootCAs:      caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}

	return &sharedConfig{peer1Config: peer1TlsConfig, peer2Config: peer2TlsConfig}, nil
}

// generateCert takes a CA, makes a new private key and certificate, and returns a tls.Certificate
func generateCert(ca *x509.Certificate, caKey *ecdsa.PrivateKey) (*tls.Certificate, error) {
	// Create a new ECDSA private key for peer1
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("ecdsa.GenerateKey: %w", err)
	}
	certificate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName: "localhost",
		},
		DNSNames:    []string{"localhost"},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(0, 0, 1),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, certificate, ca, &key.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("x509.CreateCertificate: %w", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("x509.MarshalECPrivateKey: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("tls.X509KeyPair: %w", err)
	}
	return &tlsCert, nil
}

func makeRemoteEngineTls(listenAddr string, config *tls.Config) (*actor.Engine, *Remote, error) {
	var eng *actor.Engine
	var err error
	rem := New(listenAddr, NewConfig().WithTLS(config))
	eng, err = actor.NewEngine(actor.NewEngineConfig().WithRemote(rem))
	if err != nil {
		return nil, nil, fmt.Errorf("actor.NewEngine: %w", err)
	}
	return eng, rem, nil
}

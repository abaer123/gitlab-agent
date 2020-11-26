package tlstool

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
)

func LoadCACert(caCertFile string) (*x509.CertPool, error) {
	certPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("SystemCertPool: %v", err)
	}
	caCert, err := ioutil.ReadFile(caCertFile) // nolint: gosec
	if err != nil {
		return nil, fmt.Errorf("CA certificate file: %v", err)
	}
	ok := certPool.AppendCertsFromPEM(caCert)
	if !ok {
		return nil, fmt.Errorf("AppendCertsFromPEM(%s) failed", caCertFile)
	}
	return certPool, nil
}

func DefaultClientTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
}

func DefaultClientTLSConfigWithCACert(caCertFile string) (*tls.Config, error) {
	tlsConfig := DefaultClientTLSConfig()
	if caCertFile != "" {
		certPool, err := LoadCACert(caCertFile)
		if err != nil {
			return nil, err
		}
		tlsConfig.RootCAs = certPool
	}
	return tlsConfig, nil
}

func DefaultServerTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("loading certificate (%s) and key (%s) files: %v", certFile, keyFile, err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

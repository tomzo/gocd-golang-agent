package libgogoagent

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

var (
	goServerCAFile = "go-server-ca.pem"
	privateKeyFile = "agent-private-key.pem"
	certFile       = "agent-cert.pem"
)

type Registration struct {
	AgentPrivateKey, AgentCertificate string
}

func ReadGoServerCACert() error {
	_, err := os.Stat(goServerCAFile)
	if err == nil {
		return nil
	}

	conn, err := tls.Dial("tcp", sslHostAndPort(), &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		LogInfo("failed to connect: " + err.Error())
		return err
	}
	defer conn.Close()
	state := conn.ConnectionState()
	certOut, err := os.Create(goServerCAFile)
	if err != nil {
		LogInfo("failed to open %v for writing: %s", goServerCAFile, err)
		return err
	}
	defer certOut.Close()
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: state.PeerCertificates[0].Raw})
	return nil
}

func GoServerDN() string {
	return extractServerDN(goServerCAFile)
}

func GoServerRootCAs() (*x509.CertPool, error) {
	caCert, err := ioutil.ReadFile(goServerCAFile)
	if err != nil {
		return nil, err
	}
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(caCert))
	if !ok {
		return nil, errors.New("failed to parse root certificate")
	}
	return roots, nil
}

func GoServerTlsConfig(withClientCert bool) (*tls.Config, error) {
	certs := make([]tls.Certificate, 0)
	if withClientCert {
		cert, err := tls.LoadX509KeyPair(certFile, privateKeyFile)
		if err != nil {
			panic(err)
		}
		certs = append(certs, cert)
	}
	roots, err := GoServerRootCAs()
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: certs,
		RootCAs:      roots,
		ServerName:   extractServerDN(goServerCAFile),
	}, nil
}

func GoServerRemoteClient(withClientCert bool) (*http.Client, error) {
	config, err := GoServerTlsConfig(withClientCert)
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{
		TLSClientConfig: config,
	}
	return &http.Client{Transport: tr}, nil
}

func Register(params map[string]string) error {
	if err := ReadGoServerCACert(); err != nil {
		return err
	}
	if err := readAgentKeyAndCerts(params); err != nil {
		return err
	}
	return nil
}

func CleanRegistration() error {
	files := []string{goServerCAFile, privateKeyFile, certFile}
	for _, f := range files {
		_, err := os.Stat(f)
		if err == nil {
			err := os.Remove(f)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func readAgentKeyAndCerts(params map[string]string) error {
	_, privateKeyFileErr := os.Stat(privateKeyFile)
	_, certFileErr := os.Stat(certFile)
	if privateKeyFileErr == nil && certFileErr == nil {
		return nil
	}

	form := url.Values{}
	for k, v := range params {
		form.Add(k, v)
	}

	client, err := GoServerRemoteClient(false)
	if err != nil {
		return err
	}
	resp, err := client.PostForm(httpsServerURL("/go/admin/agent"), form)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	var registration Registration

	dec := json.NewDecoder(resp.Body)

	if err := dec.Decode(&registration); err != nil {
		return err
	}

	ioutil.WriteFile(privateKeyFile, []byte(registration.AgentPrivateKey), 0600)
	ioutil.WriteFile(certFile, []byte(registration.AgentCertificate), 0600)
	return nil
}

func extractServerDN(certFileName string) string {
	pemBlock, err := ioutil.ReadFile(certFileName)
	if err != nil {
		panic(err)
	}

	der, _ := pem.Decode(pemBlock)
	cert, _ := x509.ParseCertificate(der.Bytes)
	return cert.Subject.CommonName
}

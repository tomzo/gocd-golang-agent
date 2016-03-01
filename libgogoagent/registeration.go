package libgogoagent

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io/ioutil"
	"log"
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

func ReadGoServerCACert() {
	conn, err := tls.Dial("tcp", sslHostAndPort(), &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		panic("failed to connect: " + err.Error())
	}
	defer conn.Close()
	state := conn.ConnectionState()
	certOut, err := os.Create(goServerCAFile)
	if err != nil {
		log.Fatalf("failed to open %v for writing: %s", goServerCAFile, err)
	}
	defer certOut.Close()
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: state.PeerCertificates[0].Raw})
}

func GoServerDN() string {
	return extractServerDN(goServerCAFile)
}

func GoServerRootCAs() *x509.CertPool {
	caCert, err := ioutil.ReadFile(goServerCAFile)
	if err != nil {
		panic(err)
	}
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(caCert))
	if !ok {
		panic("failed to parse root certificate")
	}
	return roots
}

func GoServerTlsConfig(withClientCert bool) *tls.Config {
	certs := make([]tls.Certificate, 0)
	if withClientCert {
		cert, err := tls.LoadX509KeyPair(certFile, privateKeyFile)
		if err != nil {
			panic(err)
		}
		certs = append(certs, cert)
	}
	return &tls.Config{
		Certificates: certs,
		RootCAs:      GoServerRootCAs(),
		ServerName:   extractServerDN(goServerCAFile),
	}
}

func GoServerRemoteClient(withClientCert bool) *http.Client {
	tr := &http.Transport{
		TLSClientConfig: GoServerTlsConfig(withClientCert),
	}
	return &http.Client{Transport: tr}
}

func Register(params map[string]string) {
	form := url.Values{}
	for k, v := range params {
		form.Add(k, v)
	}

	client := GoServerRemoteClient(false)
	resp, err := client.PostForm(httpsServerURL("/go/admin/agent"), form)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	var registration Registration

	dec := json.NewDecoder(resp.Body)

	if err := dec.Decode(&registration); err != nil {
		panic(err)
	}

	ioutil.WriteFile(privateKeyFile, []byte(registration.AgentPrivateKey), 0600)
	ioutil.WriteFile(certFile, []byte(registration.AgentCertificate), 0600)
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

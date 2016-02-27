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

func GoServerTlsConfig() *tls.Config {
	caCert, err := ioutil.ReadFile(goServerCAFile)
	if err != nil {
		panic(err)
	}
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(caCert))
	if !ok {
		panic("failed to parse root certificate")
	}

	cert, err := tls.LoadX509KeyPair(certFile, privateKeyFile)
	if err != nil {
		panic(err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      roots,
		ServerName:   extractServerDN(goServerCAFile),
	}

	tlsConfig.BuildNameToCertificate()
	return tlsConfig
}

func Register(params map[string]string) {
	form := url.Values{}
	for k, v := range params {
		form.Add(k, v)
	}
	resp, _ := http.PostForm(httpServerURL("/go/admin/agent"), form)

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

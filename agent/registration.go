/*
 * Copyright 2016 ThoughtWorks, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package agent

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"github.com/gocd-contrib/gocd-golang-agent/protocal"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"runtime"
)

func ReadGoServerCACert() error {
	_, err := os.Stat(config.GoServerCAFile)
	if err == nil {
		return nil
	}

	LogInfo("fetching Go server[%v] CA certificate", config.ServerHostAndPort)
	conn, err := tls.Dial("tcp", config.ServerHostAndPort, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		logger.Error.Printf("failed to connect: " + err.Error())
		return err
	}
	defer conn.Close()
	state := conn.ConnectionState()
	certOut, err := os.Create(config.GoServerCAFile)
	if err != nil {
		logger.Error.Printf("failed to open %v for writing: %s", config.GoServerCAFile, err)
		return err
	}
	defer certOut.Close()
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: state.PeerCertificates[0].Raw})
	return nil
}

func GoServerRootCAs() (*x509.CertPool, error) {
	caCert, err := ioutil.ReadFile(config.GoServerCAFile)
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
		cert, err := tls.LoadX509KeyPair(config.AgentCertFile, config.AgentPrivateKeyFile)
		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
	}
	roots, err := GoServerRootCAs()
	if err != nil {
		return nil, err
	}
	serverName, err := extractServerDN(config.GoServerCAFile)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: certs,
		RootCAs:      roots,
		ServerName:   serverName,
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

func Register() error {
	if err := ReadGoServerCACert(); err != nil {
		return err
	}
	if err := readAgentKeyAndCerts(registerData()); err != nil {
		return err
	}
	return nil
}

func CleanRegistration() error {
	files := []string{config.GoServerCAFile,
		config.AgentPrivateKeyFile,
		config.AgentCertFile}
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

func registerData() map[string]string {
	hostname, _ := os.Hostname()
	workingDir, _ := os.Getwd()

	return map[string]string{
		"hostname":                      hostname,
		"uuid":                          AgentId,
		"location":                      workingDir,
		"operatingSystem":               runtime.GOOS,
		"usablespace":                   UsableSpaceString(),
		"agentAutoRegisterKey":          config.AgentAutoRegisterKey,
		"agentAutoRegisterResources":    config.AgentAutoRegisterResources,
		"agentAutoRegisterEnvironments": config.AgentAutoRegisterEnvironments,
		"agentAutoRegisterHostname":     hostname,
		"elasticAgentId":                config.AgentAutoRegisterElasticAgentId,
		"elasticPluginId":               config.AgentAutoRegisterElasticPluginId,
	}
}

func readAgentKeyAndCerts(params map[string]string) error {
	_, agentPrivateKeyFileErr := os.Stat(config.AgentPrivateKeyFile)
	_, agentCertFileErr := os.Stat(config.AgentCertFile)
	if agentPrivateKeyFileErr == nil && agentCertFileErr == nil {
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

	LogInfo("fetching agent key and certificates")
	url, err := config.RegistrationURL()
	if err != nil {
		return err
	}
	resp, err := client.PostForm(url.String(), form)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	var registration protocal.Registration

	dec := json.NewDecoder(resp.Body)

	if err := dec.Decode(&registration); err != nil {
		return err
	}

	ioutil.WriteFile(config.AgentPrivateKeyFile, []byte(registration.AgentPrivateKey), 0600)
	ioutil.WriteFile(config.AgentCertFile, []byte(registration.AgentCertificate), 0600)
	return nil
}

func extractServerDN(certFileName string) (string, error) {
	pemBlock, err := ioutil.ReadFile(certFileName)
	if err != nil {
		return "", err
	}

	der, _ := pem.Decode(pemBlock)
	cert, err := x509.ParseCertificate(der.Bytes)
	if err != nil {
		return "", err
	}
	return cert.Subject.CommonName, nil
}

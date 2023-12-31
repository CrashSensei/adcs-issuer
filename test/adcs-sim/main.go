/*
Copyright 2019 The Jetstack cert-manager contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"github.com/nokia/adcs-issuer/test/adcs-sim/certserv"

)

var (

	//caWorkDir = flag.Lookup("workdir").Value.(flag.Getter).Get().(string)
   caWorkDir="/usr/local/adcs-sim"
   serverPem = caWorkDir + "/ca/server.pem"
   serverKey = caWorkDir + "/ca/server.key"
   serverCsr = caWorkDir + "/ca/server.csr"

)

func getEnv(key, fallback string) string {
    if value, ok := os.LookupEnv(key); ok {
        return value
    }
    return fallback
}

func main() {
	port := flag.Int("port", 8443, "Port to listen on")
	dns := flag.String("dns", "", "Comma separated list of domains for the simulator server certificate")
	ips := flag.String("ips", "", "Comma separated list of IPs for the simulator server certificate")

	flag.Parse()


	fmt.Printf("Using %s directory for simulator in main\n", caWorkDir)
    fmt.Printf("Directories in %s\n",caWorkDir)
	files, err := ioutil.ReadDir(caWorkDir)
    if err != nil {
        log.Fatal(err)
    }

    for _, file := range files {
        fmt.Println(file.Name(), file.IsDir())
    }
	fmt.Printf("Files in %s/ca\n",caWorkDir)
	filesca, err := ioutil.ReadDir(caWorkDir+"/ca")
    if err != nil {
        log.Fatal(err)
    }
    for _, fileca := range filesca {
        fmt.Println(fileca.Name(), fileca.IsDir())
    }

	fmt.Printf("Files in %s/templates\n",caWorkDir)

	filestemplate, err := ioutil.ReadDir(caWorkDir+"/templates")
    if err != nil {
        log.Fatal(err)
    }

    for _, filetemplate := range filestemplate {
        fmt.Println(filetemplate.Name(), filetemplate.IsDir())
    }

	certserv, err := certserv.NewCertserv()
	if err != nil {
		fmt.Printf("Cannot initialize NewCertserv() : %s\n", err.Error())
	}
	err = generateServerCertificate(certserv, ips, dns)
	if err != nil {
		fmt.Printf("Cannot generate server certificate generateServerCertificate() : %s\n", err.Error())
	}

	http.HandleFunc("/certnew.cer", certserv.HandleCertnewCer)
	http.HandleFunc("/certnew.p7b", certserv.HandleCertnewP7b)
	http.HandleFunc("/certcarc.asp", certserv.HandleCertcarcAsp)
	http.HandleFunc("/certfnsh.asp", certserv.HandleCertfnshAsp)
	log.Fatal(http.ListenAndServeTLS(fmt.Sprintf(":%d", *port), serverPem, serverKey, nil))
}

// Generate certificate for the simulator server TLS
func generateServerCertificate(cs *certserv.Certserv, ips *string, dns *string) error {
	fmt.Printf("Using %s directory for simulator in generateServerCertificate: ip: %s, dns %s \n", caWorkDir, *ips,*dns)
	var ipAddresses []net.IP
	if ips != nil && len(*ips) > 0 {
		for _, ipString := range strings.Split(*ips, ",") {

			ip := net.ParseIP(ipString)
			if ip == nil {
				fmt.Printf("Error parsing ip=%s\n", ipString)
				continue
			}
			ipAddresses = append(ipAddresses, ip)
		}
	}
	var dnsNames []string
	if dns != nil && len(*dns) > 0 {
		dnsNames = strings.Split(*dns, ",")
	}

	organization := []string{"ADCS simulator for cert-manager testing"}

	if len(ipAddresses) == 0 && len(dnsNames) == 0 {
		return fmt.Errorf("no subjects specified on certificate")
	}
	var commonName string
	if len(dnsNames) > 0 {
		commonName = dnsNames[0]
	} else {
		commonName = ipAddresses[0].String()
	}
	// CSR
	pubKeyAlgo := x509.RSA
	sigAlgo := x509.SHA512WithRSA
	csr := &x509.CertificateRequest{
		Version:            3,
		SignatureAlgorithm: sigAlgo,
		PublicKeyAlgorithm: pubKeyAlgo,
		Subject: pkix.Name{
			Organization: organization,
			CommonName:   commonName,
		},
		DNSNames:    dnsNames,
		IPAddresses: ipAddresses,
		// TODO: work out how best to handle extensions/key usages here
		ExtraExtensions: []pkix.Extension{},
	}
	// Private key
	keySize := 2048
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return fmt.Errorf("error creating x509 key: %s", err.Error())
	}
	keyBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	fmt.Printf("Writing server key file: %s\n", serverKey)
	err = os.WriteFile(serverKey, keyBytes, 0644)
	if err != nil {
		return fmt.Errorf("error writing key file: %s", err.Error())
	}

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, csr, privateKey)
	if err != nil {
		return fmt.Errorf("error creating x509 certificate request: %s", err.Error())
	}
	err = ioutil.WriteFile(serverCsr, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes}), 0644)
	if err != nil {
		return fmt.Errorf("error writing CSR file: %s", err.Error())
	}

	certData, err := x509.ParseCertificateRequest(csrBytes)
	if err != nil {
		return fmt.Errorf("error parsing x509 certificate request: %s", err.Error())
	}
	certPem, err := cs.CreateCertificateChainPem(certData)

	if err != nil {
		return fmt.Errorf("error creating x509 certificate: %s", err.Error())
	}
	err = ioutil.WriteFile(serverPem, []byte(certPem), 0644)
	if err != nil {
		return fmt.Errorf("error writing certificate file: %s", err.Error())
	}
	return nil
}

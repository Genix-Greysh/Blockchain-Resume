/*
Copyright IBM Corp. 2016 All Rights Reserved.

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
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

func checkInit(t *testing.T, stub *shim.MockStub, args []string) {
	_, err := stub.MockInit("1", "Init", args)
	if err != nil {
		fmt.Println("Init failed", err)
		t.FailNow()
	}
}

func checkState(t *testing.T, stub *shim.MockStub, name string) {
	bytes := stub.State[name]
	if bytes == nil {
		fmt.Println("State", name, "failed to get value")
		t.FailNow()
	}
	t.Log("State value", name, "is", string(bytes))
}

func checkQuery(t *testing.T, stub *shim.MockStub, function string, args []string) ([]byte, error) {
	bytes, err := stub.MockQuery(function, args)
	if err != nil {
		fmt.Println("Query failed!", err)
		t.FailNow()
	}
	return bytes, err
}

func checkInvoke(t *testing.T, stub *shim.MockStub, function string, args []string) ([]byte, error) {
	ret, err := stub.MockInvoke("1", function, args)
	if err != nil {
		fmt.Println("Invoke", args, "failed", err)
		t.FailNow()
	}
	fmt.Print(string(ret))

	return ret, err
}

type KeyPair struct {
	PubKey  string
	PrivKey string
}

// Generate a new 1024-bit RSA key pair
func newKeyPair() (rsa.PublicKey, rsa.PrivateKey) {
	r := rand.New(rand.NewSource(time.Now().UTC().UnixNano()))
	prikey, _ := rsa.GenerateKey(r, 1024)
	pubkey := prikey.Public()

	return *pubkey.(*rsa.PublicKey), *prikey
}
func keyPairToPem(pub rsa.PublicKey, priv rsa.PrivateKey) (string, string) {
	priv_pem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(&priv),
	})
	marshaled_pub, _ := x509.MarshalPKIXPublicKey(&pub)
	pub_pem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: marshaled_pub,
	})

	return string(pub_pem), string(priv_pem)
}

func TestChainCode_0(t *testing.T) {
	scc := new(Chaincode)
	stub := shim.NewMockStub("ex02", scc)

	checkInit(t, stub, []string{})

	checkState(t, stub, "#Counter#")

	RpPub, RpPriv := newKeyPair()
	RpPubPem, _ := keyPairToPem(RpPub, RpPriv)

	RpPubPem = strings.Replace(RpPubPem, "\n", "\\n", -1)
	data, err := checkInvoke(t, stub, "AddRecipient", []string{`{"Rp" : {"ID":"jj", "Name":"jj"}, "PubKeyPem": "` + RpPubPem + `"}`})

	IsPub, IsPriv := newKeyPair()
	IsPubPem, _ := keyPairToPem(IsPub, IsPriv)

	IsPubPem = strings.Replace(IsPubPem, "\n", "\\n", -1)
	data, err = checkInvoke(t, stub, "AddIssuer", []string{`{"Issuer":"IssuerA", "PubKeyPem": "` + IsPubPem + `"}`})

	certJson := `{"Issuer":"IssuerA", "Link": "111", "Hash": "xxx", "Description": "jjj", "Recipient" : {"ID": "jj", "Name": "jj"}}`
	hashed := sha256.Sum256([]byte(certJson))
	signed, err := rsa.SignPKCS1v15(nil, &IsPriv, crypto.SHA256, hashed[:])
	if err != nil {
		fmt.Print(err)
	}
	checkInvoke(t, stub, "IssueCert", []string{certJson, string(signed)})

	var cert_list []CertListResponseElem
	data, err = checkQuery(t, stub, "GetCertList", []string{`{"ID":"jj", "Name":"jj"}`})
	if err != nil {
		fmt.Print(err)
	}
	err = json.Unmarshal(data, &cert_list)
	if err != nil {
		fmt.Print(err)
	}
	fmt.Print(cert_list)

	t.FailNow()
}

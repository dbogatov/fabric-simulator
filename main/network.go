package main

import (
	"fmt"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
)

// Network ...
type Network struct {
	root          CredentialsHolder
	organizations []Organization
	users         []User
}

// MakeNetwork ...
func MakeNetwork(prg *amcl.RAND, rootSk dac.SK) (network *Network) {

	network = &Network{}

	network.root = CredentialsHolder{
		KeysHolder: KeysHolder{
			pk: sysParams.rootPk,
			sk: rootSk,
		},
		credentials: *dac.MakeCredentials(sysParams.rootPk),
		name:        "Root",
	}
	credStarter := network.root.credentials.ToBytes()

	orgLevel := 1

	for org := 0; org < sysParams.orgs; org++ {

		orgSk, orgPk := dac.GenerateKeys(prg, orgLevel)

		// Credential request

		rootNonce := randomBytes(prg, NonceSize)
		recordBandwidth("root", fmt.Sprintf("org-%d", org), Nonce{rootNonce})

		credRequest := dac.MakeCredRequest(prg, orgSk, rootNonce, orgLevel)
		recordBandwidth(fmt.Sprintf("org-%d", org), "root", CredRequest{credRequest})

		if e := credRequest.Validate(); e != nil {
			panic(e)
		}

		// Root CA delegates the credentials

		attributes := []interface{}{
			dac.ProduceAttributes(orgLevel, fmt.Sprintf("org-%d", org))[0],
			dac.ProduceAttributes(orgLevel, "has-right-to-post")[0],
		}

		credsOrg := dac.CredentialsFromBytes(credStarter)
		if e := credsOrg.Delegate(rootSk, orgPk, attributes, prg, sysParams.ys); e != nil {
			panic(e)
		}
		recordBandwidth("root", fmt.Sprintf("org-%d", org), Credentials{credsOrg})

		if v := dac.VerifyKeyPair(orgSk, orgPk); !v {
			panic(v)
		}

		if e := credsOrg.Verify(orgSk, sysParams.rootPk, sysParams.ys); e != nil {
			panic(e)
		}

		network.organizations = append(
			network.organizations,
			Organization{
				CredentialsHolder: CredentialsHolder{
					KeysHolder: KeysHolder{
						pk: orgPk,
						sk: orgSk,
					},
					credentials: *credsOrg,
					name:        fmt.Sprintf("org-%d", org),
				},
			},
		)
	}

	return
}

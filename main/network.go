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
	userLevel := 2

	log.Info("Root CA has been initialized")

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

	log.Info("All organizations have received their credentials")

	for i, org := range network.organizations {

		credStarter = org.credentials.ToBytes()

		for user := 0; user < sysParams.users; user++ {

			userName := fmt.Sprintf("user-%d-%d", i, user)

			userSk, userPk := dac.GenerateKeys(prg, userLevel)

			// Credential request

			orgNonce := randomBytes(prg, NonceSize)
			recordBandwidth(org.name, userName, Nonce{orgNonce})

			credRequest := dac.MakeCredRequest(prg, userSk, orgNonce, userLevel)
			recordBandwidth(userName, org.name, CredRequest{credRequest})

			if e := credRequest.Validate(); e != nil {
				panic(e)
			}

			// Organization delegates the credentials

			attributes := []interface{}{
				dac.ProduceAttributes(userLevel, userName)[0],
				dac.ProduceAttributes(userLevel, "has-right-to-post")[0],
				dac.ProduceAttributes(userLevel, "something-else")[0],
			}

			credsUser := dac.CredentialsFromBytes(credStarter)
			if e := credsUser.Delegate(org.sk, userPk, attributes, prg, sysParams.ys); e != nil {
				panic(e)
			}
			recordBandwidth(org.name, userName, Credentials{credsUser})

			if e := credsUser.Verify(userSk, sysParams.rootPk, sysParams.ys); e != nil {
				panic(e)
			}

			network.users = append(
				network.users,
				User{
					CredentialsHolder: CredentialsHolder{
						KeysHolder: KeysHolder{
							pk: userPk,
							sk: userSk,
						},
						credentials: *credsUser,
						name:        userName,
					},
					org: &org,
				},
			)
		}
	}

	log.Info("All users have received their credentials")

	return
}

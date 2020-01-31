package main

import (
	"fmt"
	"sync"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
)

// Network ...
type Network struct {
	root          CredentialsHolder
	auditor       KeysHolder
	organizations []Organization
	users         []User
	peers         []Peer
	transactions  []Transaction

	revocationAuthority RevocationAuthority
	epoch               int

	transactionRecordLock *sync.Mutex
}

// MakeNetwork ...
func MakeNetwork(prg *amcl.RAND, rootSk dac.SK) (network *Network) {

	auditSk, auditPk := dac.GenerateKeys(prg, 2) // user level

	network = &Network{
		root: CredentialsHolder{
			KeysHolder: KeysHolder{
				pk: sysParams.rootPk,
				sk: rootSk,
			},
			credentials: *dac.MakeCredentials(sysParams.rootPk),
			name:        "Root",
		},
		auditor: KeysHolder{
			pk: auditPk,
			sk: auditSk,
		},
		transactionRecordLock: &sync.Mutex{},
		revocationAuthority:   *MakeRevocationAuthority(),
		epoch:                 1,
	}
	credStarter := network.root.credentials.ToBytes()

	logger.Info("Root CA has been initialized")

	network.generateOrganizations(prg, credStarter, rootSk)
	network.generateUsers(prg)
	network.generatePeers()

	return
}

func (network *Network) generateOrganizations(prg *amcl.RAND, credStarter []byte, rootSk dac.SK) {
	const orgLevel = 1

	organizations := make(chan Organization, sysParams.orgs)
	var wgOrg sync.WaitGroup
	wgOrg.Add(sysParams.orgs)

	for org := 0; org < sysParams.orgs; org++ {

		go func(org int, seed []byte) {
			defer wgOrg.Done()

			prg := amcl.NewRAND()
			prg.Clean()
			prg.Seed(len(seed), seed)

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

			organizations <- Organization{
				CredentialsHolder: CredentialsHolder{
					KeysHolder: KeysHolder{
						pk: orgPk,
						sk: orgSk,
					},
					credentials: *credsOrg,
					name:        fmt.Sprintf("org-%d", org),
				},
				id: org,
			}

		}(org, randomBytes(prg, 32))
	}

	wgOrg.Wait()
	close(organizations)

	for org := range organizations {
		network.organizations = append(network.organizations, org)
	}

	logger.Info("All organizations have received their credentials")
}

func (network *Network) generateUsers(prg *amcl.RAND) {
	const userLevel = 2

	users := make(chan *User, sysParams.users*sysParams.orgs)
	var wgUser sync.WaitGroup
	wgUser.Add(sysParams.orgs * sysParams.users)

	for org := 0; org < sysParams.orgs; org++ {

		for user := 0; user < sysParams.users; user++ {

			go func(user, org int, seed []byte) {

				defer wgUser.Done()

				prg := amcl.NewRAND()
				prg.Clean()
				prg.Seed(len(seed), seed)

				userName := fmt.Sprintf("user-%d-%d", org, user)
				organization := network.organizations[org]

				userSk, userPk := dac.GenerateKeys(prg, userLevel)

				// Credential request

				orgNonce := randomBytes(prg, NonceSize)
				recordBandwidth(organization.name, userName, Nonce{orgNonce})

				credRequest := dac.MakeCredRequest(prg, userSk, orgNonce, userLevel)
				recordBandwidth(userName, organization.name, CredRequest{credRequest})

				if e := credRequest.Validate(); e != nil {
					panic(e)
				}

				// Organization delegates the credentials

				attributes := []interface{}{
					dac.ProduceAttributes(userLevel, userName)[0],
					dac.ProduceAttributes(userLevel, "has-right-to-post")[0],
					dac.ProduceAttributes(userLevel, "something-else")[0],
				}

				credsUser := dac.CredentialsFromBytes(organization.credentials.ToBytes())
				if e := credsUser.Delegate(organization.sk, userPk, attributes, prg, sysParams.ys); e != nil {
					panic(e)
				}
				recordBandwidth(organization.name, userName, Credentials{credsUser})

				if e := credsUser.Verify(userSk, sysParams.rootPk, sysParams.ys); e != nil {
					panic(e)
				}

				users <- MakeUser(
					CredentialsHolder{
						KeysHolder: KeysHolder{
							pk: userPk,
							sk: userSk,
						},
						credentials: *credsUser,
						name:        userName,
					},
					user,
					org,
				)

			}(user, org, randomBytes(prg, 32))
		}
	}

	wgUser.Wait()
	close(users)

	for user := range users {
		network.users = append(network.users, *user)
	}

	logger.Info("All users have received their credentials")
}

func (network *Network) generatePeers() {
	for peer := 0; peer < sysParams.peers; peer++ {
		network.peers = append(network.peers, *MakePeer(peer))
	}

	logger.Info("All peers have been spinned up")
}

func (network *Network) stop() {
	for _, peer := range network.peers {
		peer.exitChannel <- true
	}

	logger.Info("All peers have been shut down")
}

func (network *Network) recordTransaction(tx *Transaction) {
	network.transactionRecordLock.Lock()

	defer network.transactionRecordLock.Unlock()

	network.transactions = append(network.transactions, *tx)
}

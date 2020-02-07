package main

import (
	"fmt"
	"sync"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
	"github.com/dbogatov/fabric-amcl/amcl/FP256BN"
	"gonum.org/v1/gonum/stat/distuv"
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
			kind:        "root",
			id:          0,
		},
		auditor: KeysHolder{
			pk: auditPk,
			sk: auditSk,
		},
		transactionRecordLock: &sync.Mutex{},
		revocationAuthority:   *MakeRevocationAuthority(),
		epoch:                 1,
		users:                 make([]User, sysParams.orgs*sysParams.users),
	}
	credStarter := network.root.credentials.ToBytes()

	logger.Notice("Root CA has been initialized")

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

			prg := newRandSeed(seed)

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
					kind:        "org",
					id:          org,
				},
			}

		}(org, randomBytes(prg, 32))
	}

	wgOrg.Wait()
	close(organizations)

	for org := range organizations {
		network.organizations = append(network.organizations, org)
	}

	logger.Notice("All organizations have received their credentials")
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

				prg := newRandSeed(seed)

				userName := fmt.Sprintf("user-%d", org*sysParams.users+user)
				organization := network.organizations[org]
				orgName := fmt.Sprintf("org-%d", organization.id)

				userSk, userPk := dac.GenerateKeys(prg, userLevel)

				// Credential request

				orgNonce := randomBytes(prg, NonceSize)
				recordBandwidth(orgName, userName, Nonce{orgNonce})

				credRequest := dac.MakeCredRequest(prg, userSk, orgNonce, userLevel)
				recordBandwidth(userName, orgName, CredRequest{credRequest})

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
				recordBandwidth(orgName, userName, Credentials{credsUser})

				if e := credsUser.Verify(userSk, sysParams.rootPk, sysParams.ys); e != nil {
					panic(e)
				}

				users <- &User{
					CredentialsHolder: CredentialsHolder{
						KeysHolder: KeysHolder{
							pk: userPk,
							sk: userSk,
						},
						credentials: *credsUser,
						kind:        "user",
						id:          org*sysParams.users + user,
					},
					revocationPK: FP256BN.ECP_generator().Mul(userSk),
					org:          org,
					poisson: distuv.Poisson{
						Lambda: 3600.0 / float64(sysParams.frequency),
					},
				}

			}(user, org, randomBytes(prg, 32))
		}
	}

	wgUser.Wait()
	close(users)

	for user := range users {
		network.users[user.id] = *user
	}

	logger.Notice("All users have received their credentials")
}

func (network *Network) generatePeers() {
	for peer := 0; peer < sysParams.peers; peer++ {
		network.peers = append(network.peers, *MakePeer(peer))
	}

	logger.Notice("All peers have been spinned up")
}

func (network *Network) stop() {
	for _, peer := range network.peers {
		peer.exitChannel <- true
	}
	network.revocationAuthority.exitChannel <- true

	logger.Notice("All peers and the revocation authority have been shut down")
}

func (network *Network) recordTransaction(tx *Transaction) {
	network.transactionRecordLock.Lock()

	defer network.transactionRecordLock.Unlock()

	network.transactions = append(network.transactions, *tx)

	current := len(network.transactions)
	total := sysParams.transactions * len(sysParams.network.users)

	logger.Noticef("%4.1f%% - transaction %d / %d", 100*float64(current)/float64(total), current, total)
}

package main

func simulate(orgs, users, peers, epoch int, revoke, audit bool, idemix string) (e error) {

	log.Infof("%d organizations %d users each managed by %d peers\n", orgs, users, peers)
	log.Infof("Epochs are %d seconds long\n", epoch)
	log.Infof("Revocations enabled: %t, auditings enabled %t\n", revoke, audit)
	log.Infof("\"%s\" version of idemix is used\n", idemix)

	return
}

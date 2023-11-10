package infrastructure

import (
	"os"
)

func ReadSSHKeysFromDisk(pubKeyPath string, privKeyPath string) (string, string, error) {
	pubKey, err := os.ReadFile(pubKeyPath)
	pubKeyString := string(pubKey)
	if err != nil {
		return "", "", err
	}
	privKey, err := os.ReadFile(privKeyPath)
	privKeyString := string(privKey)
	if err != nil {
		return "", "", err
	}

	return pubKeyString, privKeyString, nil
}

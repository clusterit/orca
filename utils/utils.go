package utils

import (
	"crypto/md5"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

// Fingerprint generates a readable md5 fingerprint of a public key
func Fingerprint(k ssh.PublicKey) string {
	hash := md5.Sum(k.Marshal())
	return strings.Replace(fmt.Sprintf("% x", hash), " ", ":", -1)
}

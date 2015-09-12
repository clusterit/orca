package utils

import (
	"crypto/md5"
	"fmt"
	"strings"

	"github.com/satori/go.uuid"

	"golang.org/x/crypto/ssh"
)

// Fingerprint generates a readable md5 fingerprint of a public key
func Fingerprint(k ssh.PublicKey) string {
	hash := md5.Sum(k.Marshal())
	return strings.Replace(fmt.Sprintf("% x", hash), " ", ":", -1)
}

// NetworkUID returns a uid and a network concatenated, so this id
// should be really unique
func NetworkUID(n, u string) string {
	return fmt.Sprintf("%s@%s", u, n)
}

// GenerateUUID returns a unique identifier
func GenerateUUID() string {
	return uuid.NewV4().String()
}

package popura

import (
	"crypto/ed25519"
	"encoding/hex"
	"net"

	"github.com/yggdrasil-network/yggdrasil-go/src/address"
)

func decodeKey(input string) ed25519.PublicKey {
	keyData, _ := hex.DecodeString(input)
	return ed25519.PrivateKey(keyData).Public().(ed25519.PublicKey)
}

// Get the subnet information from PublicKey
func SubnetFromKey(inputString string) *net.IPNet {
	key := decodeKey(inputString)

	snet := address.SubnetForKey(key)
	ipnet := net.IPNet{
		IP:   append(snet[:], 0, 0, 0, 0, 0, 0, 0, 0),
		Mask: net.CIDRMask(len(snet)*8, 128),
	}

	return &ipnet
}

// Get the address information from PublicKey
func AddressFromKey(inputString string) *net.IP {
	key := decodeKey(inputString)

	addr := address.AddrForKey(key)
	ip := net.IP(addr[:])
	return &ip
}

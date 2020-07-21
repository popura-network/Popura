// +build !linux,!mobile

package radv

import (
	"errors"
)

// Display a warning if IPv6 forwarding is disabled
func (s *RAdv) checkForwardingEnabled() {
	s.log.Debugln("Not implemented")
}

// Add IP address to the network interface
func (s *RAdv) setGatewayIP() error {
	return errors.New("RAdv: setGatewayIP is not implemented on your platform")
}

// Remove IP address to the network interface
func (s *RAdv) removeGatewayIP() error {
	return errors.New("RAdv: removeGatewayIP is not implemented on your platform")
}

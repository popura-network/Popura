// +build !linux,!mobile

package radv

// Display a warning if IPv6 forwarding is disabled
func (s *RAdv) checkForwardingEnabled() {
	s.log.Debugln("Not implemented")
}

// Add IP address to the network interface
func (s *RAdv) setGatewayIP() error {
	s.log.Debugln("Not implemented")

	return nil
}

// Remove IP address to the network interface
func (s *RAdv) removeGatewayIP() error {
	s.log.Debugln("Not implemented")

	return nil
}

// +build !mobile

package radv

// The linux platform specific tun parts

import (
	"io/ioutil"
	"net"

	"github.com/vishvananda/netlink"
)

// Display a warning if IPv6 forwarding is disabled
func (s *RAdv) checkForwardingEnabled() {
	dat, err := ioutil.ReadFile("/proc/sys/net/ipv6/conf/" + s.config.Interface + "/forwarding")
	if err != nil {
		panic(err)
	}
	if string(dat[0]) != "1" {
		s.log.Warnln("IPv6 forwarding is disabled. To enable it, run: sudo sysctl -w net.ipv6.conf.all.forwarding=1")
	}
}

// Get gateway IP address for the subnet
func getGatewayNetlinkAddr(subnet *net.IPNet) *netlink.Addr {
	gwIP := make(net.IP, len(subnet.IP))
	copy(gwIP, subnet.IP)
	gwIP = append(gwIP[:len(gwIP)-1], 1)
	return &netlink.Addr{IPNet: &net.IPNet{IP: gwIP, Mask: subnet.Mask}}
}

// Add IP address to the network interface
func (s *RAdv) setGatewayIP() error {
	nladdr := getGatewayNetlinkAddr(s.subnet)
	nlintf, err := netlink.LinkByName(s.config.Interface)
	if err != nil {
		return err
	}
	if err := netlink.AddrAdd(nlintf, nladdr); err != nil {
		return err
	}

	return nil
}

// Remove IP address to the network interface
func (s *RAdv) removeGatewayIP() error {
	nladdr := getGatewayNetlinkAddr(s.subnet)
	nlintf, err := netlink.LinkByName(s.config.Interface)
	if err != nil {
		return err
	}
	if err := netlink.AddrDel(nlintf, nladdr); err != nil {
		return err
	}

	return nil
}

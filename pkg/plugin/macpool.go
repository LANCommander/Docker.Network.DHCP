package plugin

import (
	"encoding/hex"
	"fmt"
	"net"
	"strings"

	"github.com/devplayer0/docker-net-dhcp/pkg/util"
)

// macPool describes a contiguous range of MAC addresses: a fixed prefix followed by an index in [0, size).
//
// The pool itself holds no state. Which addresses are free is decided by the kernel: a macvlan interface cannot be
// brought up with a MAC that is already active on the same parent (EADDRINUSE). Allocating the lowest free index
// therefore reuses addresses left behind by stopped containers (so they keep getting their old DHCP leases), copes
// with concurrent endpoint creation and other networks on the same parent, and survives plugin restarts.
type macPool struct {
	prefix net.HardwareAddr
	size   uint64
}

// parseMACPrefix parses 1-5 colon- or dash-separated hex octets.
func parseMACPrefix(s string) (net.HardwareAddr, error) {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == ':' || r == '-' })
	if len(parts) < 1 || len(parts) > 5 {
		return nil, util.ErrMACPrefix
	}

	prefix := make(net.HardwareAddr, len(parts))
	for i, p := range parts {
		if len(p) != 2 {
			return nil, util.ErrMACPrefix
		}
		b, err := hex.DecodeString(p)
		if err != nil {
			return nil, util.ErrMACPrefix
		}
		prefix[i] = b[0]
	}

	if prefix[0]&0x01 != 0 {
		// Multicast bit set; such an address can't be assigned to an interface
		return nil, util.ErrMACPrefix
	}

	return prefix, nil
}

// defaultMACPrefix derives a prefix from the parent interface's MAC: a locally administered 02 octet followed by the
// parent's last three octets. This keeps pools on different hosts sharing a LAN from colliding.
func defaultMACPrefix(parentMAC net.HardwareAddr) (net.HardwareAddr, error) {
	if len(parentMAC) != 6 {
		return nil, fmt.Errorf("cannot derive default mac_prefix from parent MAC %q, please set mac_prefix", parentMAC)
	}
	return net.HardwareAddr{0x02, parentMAC[3], parentMAC[4], parentMAC[5]}, nil
}

// newMACPool builds a pool from the network options. parentMAC is only used if no prefix is configured.
func newMACPool(opts DHCPNetworkOptions, parentMAC net.HardwareAddr) (*macPool, error) {
	var prefix net.HardwareAddr
	var err error
	if opts.MACPrefix != "" {
		prefix, err = parseMACPrefix(opts.MACPrefix)
	} else {
		prefix, err = defaultMACPrefix(parentMAC)
	}
	if err != nil {
		return nil, err
	}

	maxSize := uint64(1) << (8 * uint(6-len(prefix)))
	size := maxSize
	if opts.MACPoolSize != 0 {
		if opts.MACPoolSize < 0 || uint64(opts.MACPoolSize) > maxSize {
			return nil, util.ErrMACPoolSize
		}
		size = uint64(opts.MACPoolSize)
	}

	return &macPool{prefix: prefix, size: size}, nil
}

// addr returns the i'th address in the pool.
func (m *macPool) addr(i uint64) net.HardwareAddr {
	mac := make(net.HardwareAddr, 6)
	copy(mac, m.prefix)
	for b := 5; b >= len(m.prefix); b-- {
		mac[b] = byte(i)
		i >>= 8
	}
	return mac
}

// String describes the pool's range for logging.
func (m *macPool) String() string {
	return fmt.Sprintf("%v-%v", m.addr(0), m.addr(m.size-1))
}

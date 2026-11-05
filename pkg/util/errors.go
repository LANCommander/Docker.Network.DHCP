package util

import (
	"errors"
	"net/http"
)

var (
	// ErrIPAM indicates an unsupported IPAM driver was used
	ErrIPAM = errors.New("only the null IPAM driver is supported")
	// ErrBridgeRequired indicates a network bridge was not provided for network creation
	ErrBridgeRequired = errors.New("bridge required")
	// ErrNotBridge indicates that the provided network interface is not a bridge
	ErrNotBridge = errors.New("network interface is not a bridge")
	// ErrBridgeUsed indicates that a bridge is already in use
	ErrBridgeUsed = errors.New("bridge already in use by Docker")
	// ErrMACAddress indicates an invalid MAC address
	ErrMACAddress = errors.New("invalid MAC address")
	// ErrInvalidMode indicates an unrecognised network mode was specified
	ErrInvalidMode = errors.New("invalid mode: must be 'bridge', 'macvlan', or 'ipvlan'")
	// ErrMACPoolMode indicates MAC pool options were used with a mode other than macvlan
	ErrMACPoolMode = errors.New("mac_prefix and mac_pool_size are only supported in macvlan mode")
	// ErrMACPrefix indicates an invalid MAC address prefix
	ErrMACPrefix = errors.New("invalid mac_prefix: must be 1-5 hex octets separated by ':' (e.g. 02:dc:00) and must not be a multicast address")
	// ErrMACPoolSize indicates the MAC pool size is out of range for the given prefix
	ErrMACPoolSize = errors.New("invalid mac_pool_size: must be at least 1 and fit in the octets not covered by mac_prefix")
	// ErrMACPoolExhausted indicates every MAC address in a network's pool is in use
	ErrMACPoolExhausted = errors.New("all MAC addresses in the pool are in use")
	// ErrNoLease indicates a DHCP lease was not obtained from udhcpc
	ErrNoLease = errors.New("udhcpc did not output a lease")
	// ErrNoHint indicates missing state from the CreateEndpoint stage in Join
	ErrNoHint = errors.New("missing CreateEndpoint hints")
	// ErrNotVEth indicates a host link was unexpectedly not a veth interface
	ErrNotVEth = errors.New("host link is not a veth interface")
	// ErrNoContainer indicates a container was unexpectedly not found
	ErrNoContainer = errors.New("couldn't find container by endpoint on the network")
	// ErrNoSandbox indicates missing state from the Join stage
	ErrNoSandbox = errors.New("missing joined endpoint state")
)

func ErrToStatus(err error) int {
	switch {
	case errors.Is(err, ErrIPAM), errors.Is(err, ErrBridgeRequired), errors.Is(err, ErrNotBridge),
		errors.Is(err, ErrBridgeUsed), errors.Is(err, ErrMACAddress), errors.Is(err, ErrInvalidMode),
		errors.Is(err, ErrMACPoolMode), errors.Is(err, ErrMACPrefix), errors.Is(err, ErrMACPoolSize):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

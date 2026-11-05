package plugin

import (
	"errors"
	"net"
	"testing"

	"github.com/devplayer0/docker-net-dhcp/pkg/util"
)

func TestParseMACPrefix(t *testing.T) {
	valid := map[string]string{
		"02":             "02",
		"02:dc:00":       "02:dc:00",
		"02-DC-00":       "02:dc:00",
		"02:dc:00:11:22": "02:dc:00:11:22",
	}
	for in, want := range valid {
		got, err := parseMACPrefix(in)
		if err != nil {
			t.Errorf("parseMACPrefix(%q) error: %v", in, err)
			continue
		}
		if got.String() != want {
			t.Errorf("parseMACPrefix(%q) = %v, want %v", in, got, want)
		}
	}

	for _, in := range []string{"", ":", "02:dc:00:11:22:33", "2:dc", "02:zz", "023", "01:00:5e"} {
		if _, err := parseMACPrefix(in); !errors.Is(err, util.ErrMACPrefix) {
			t.Errorf("parseMACPrefix(%q) = %v, want ErrMACPrefix", in, err)
		}
	}
}

func TestNewMACPool(t *testing.T) {
	parent, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")

	p, err := newMACPool(DHCPNetworkOptions{MACPoolSize: 16}, parent)
	if err != nil {
		t.Fatal(err)
	}
	if p.prefix.String() != "02:dd:ee:ff" || p.size != 16 {
		t.Errorf("default prefix pool = %v size %v", p.prefix, p.size)
	}
	if got := p.String(); got != "02:dd:ee:ff:00:00-02:dd:ee:ff:00:0f" {
		t.Errorf("pool range = %v", got)
	}

	p, err = newMACPool(DHCPNetworkOptions{MACPrefix: "02:dc:00"}, parent)
	if err != nil {
		t.Fatal(err)
	}
	if p.size != 1<<24 {
		t.Errorf("unbounded pool size = %v, want %v", p.size, 1<<24)
	}
	if got := p.addr(0x012345).String(); got != "02:dc:00:01:23:45" {
		t.Errorf("addr = %v", got)
	}

	for _, opts := range []DHCPNetworkOptions{
		{MACPrefix: "02:dc:00:11:22", MACPoolSize: 257},
		{MACPrefix: "02:dc", MACPoolSize: -1},
	} {
		if _, err := newMACPool(opts, parent); !errors.Is(err, util.ErrMACPoolSize) {
			t.Errorf("newMACPool(%+v) = %v, want ErrMACPoolSize", opts, err)
		}
	}

	if _, err := newMACPool(DHCPNetworkOptions{MACPoolSize: 1}, nil); err == nil {
		t.Error("expected error deriving prefix from empty parent MAC")
	}
}

func TestDecodeMACPoolOpts(t *testing.T) {
	opts, err := decodeOpts(map[string]interface{}{
		"bridge":        "eth0",
		"mode":          "macvlan",
		"mac_prefix":    "02:dc:00",
		"mac_pool_size": "32",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.MACPoolEnabled() || opts.MACPrefix != "02:dc:00" || opts.MACPoolSize != 32 {
		t.Errorf("decoded %+v", opts)
	}
}

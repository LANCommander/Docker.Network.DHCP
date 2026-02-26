# docker-net-dhcp

`docker-net-dhcp` is a Docker plugin providing a network driver which allocates IP addresses (IPv4 and optionally IPv6)
via an existing DHCP server (e.g. your router).

When configured correctly, this allows you to spin up a container (e.g. `docker run ...` or `docker-compose up ...`) and
access it on your network as if it was any other machine! _Probably_ not a great idea for production, but it's pretty
handy for home deployment.

# Usage

## Installation

The plugin can be installed with the `docker plugin install` command:

```
$ docker plugin install ghcr.io/devplayer0/docker-net-dhcp:release-linux-amd64
Plugin "ghcr.io/devplayer0/docker-net-dhcp:release-linux-amd64" is requesting the following privileges:
 - network: [host]
 - host pid namespace: [true]
 - mount: [/var/run/docker.sock]
 - capabilities: [CAP_NET_ADMIN CAP_SYS_ADMIN CAP_SYS_PTRACE]
Do you grant the above permissions? [y/N] y
release-linux-amd64: Pulling from ghcr.io/devplayer0/docker-net-dhcp
Digest: sha256:<some hash>
<some id>: Complete
Installed plugin ghcr.io/devplayer0/docker-net-dhcp:release-linux-amd64
$
```

Note: If you get an error like `invalid rootfs in image configuration`, try upgrading your Docker installation.

## Other tags

There are a number of supported tags for different architectures and versions, the format is
`<version>-<os>-<architecture>`. For example, `latest-linux-arm-v7` would install the newest build for ARMv7 (e.g. for
Raspberry Pi).

### Version

- `release`: The latest release (can be upgraded via `docker plugin upgrade`)
- `x.y.z`: A specific ([semver](https://semver.org/)) release (e.g. `0.1.0`)
- `latest`: Build of the newest commit

### OS

Currently only `linux` is supported.

### Architecture

- `amd64`: Intel / AMD 64-bit
- `386`: Intel / AMD legacy 32-bit
- `arm64-v8`: ARMv8 64-bit
- `arm-v7`: ARMv7 (e.g. Raspberry Pi)

Unfortunately Docker plugin images don't support multiple architectures per tag.

## Network modes

`net-dhcp` supports three network modes, selected with `-o mode=<mode>` at network creation time.

| Mode | `-o mode=` | Host setup required | Container interface | Host↔container traffic | UDP broadcast/multicast |
|---|---|---|---|---|---|
| Bridge (default) | `bridge` | Create a bridge, enslave NIC to it | `<bridge-name>0` | Yes | May be filtered by `br_netfilter`/iptables |
| macvlan | `macvlan` | None | `eth0` | No (see note) | Works natively |
| ipvlan L2 | `ipvlan` | None | `eth0` | No (see note) | Works natively |

**Choosing a mode:**
- Use **bridge** mode if your containers need to communicate with the Docker host, or if you need fine-grained iptables control.
- Use **macvlan** mode if your workload relies on UDP broadcast or multicast (e.g. game servers, mDNS, discovery protocols). Each container gets a unique MAC address, so DHCP reservations by MAC work normally.
- Use **ipvlan** mode only if your NIC or upstream switch prevents MAC spoofing (making macvlan impractical). Note that all containers share the host NIC's MAC address, which means DHCP servers that key leases to MAC addresses will issue all containers the same IP — this mode requires manual MAC-based DHCP client IDs to be configured at the OS level and is not recommended for most use cases.

> **macvlan/ipvlan host isolation:** A well-known Linux limitation means that the Docker host cannot directly reach containers on a macvlan or ipvlan network (and vice versa) because the host NIC and its sub-interfaces are L2-isolated from each other. If you need host↔container reachability, use bridge mode or add a dedicated macvlan interface on the host side and assign it an IP.

## Network creation

### Bridge mode (default)

Bridge mode requires a pre-configured bridge interface on the host with your physical NIC enslaved to it. How you set
this up will depend on your system, but the following (manual) instructions should work on most Linux distros:

```
# Create the bridge
$ sudo ip link add my-bridge type bridge
$ sudo ip link set my-bridge up

# Assuming 'eth0' is connected to your LAN (where the DHCP server is)
$ sudo ip link set eth0 up
# Attach your network card to the bridge
$ sudo ip link set eth0 master my-bridge

# If your firewall's policy for forwarding is to drop packets, you'll need to add an ACCEPT rule
$ sudo iptables -A FORWARD -i my-bridge -j ACCEPT

# Get an IP for the host (will go out to the DHCP server since eth0 is attached to the bridge)
# Replace this step with whatever network configuration you were using for eth0
$ sudo dhcpcd my-bridge
```

Once the bridge is ready, create the network:

```
$ docker network create -d ghcr.io/devplayer0/docker-net-dhcp:release-linux-amd64 --ipam-driver null -o bridge=my-bridge my-dhcp-net
<some network id>
$

# With IPv6 enabled
# Although `docker network create` has a `--ipv6` flag, it doesn't work with the null IPAM driver
$ docker network create -d ghcr.io/devplayer0/docker-net-dhcp:release-linux-amd64 --ipam-driver null -o bridge=my-bridge -o ipv6=true my-dhcp-net
<some network id>
$
```

### macvlan mode

macvlan mode requires no bridge setup — point `-o bridge=` directly at the physical NIC that is connected to your LAN.
The plugin creates a macvlan sub-interface on that NIC for each container, giving each one a unique MAC address and a
DHCP lease from your network's DHCP server. Traffic bypasses the Linux bridge layer entirely, so UDP broadcasts and
multicast work without any iptables configuration.

```
$ docker network create -d ghcr.io/devplayer0/docker-net-dhcp:release-linux-amd64 --ipam-driver null \
    -o bridge=eth0 \
    -o mode=macvlan \
    my-dhcp-macvlan
<some network id>
$

# With IPv6 enabled
$ docker network create -d ghcr.io/devplayer0/docker-net-dhcp:release-linux-amd64 --ipam-driver null \
    -o bridge=eth0 \
    -o mode=macvlan \
    -o ipv6=true \
    my-dhcp-macvlan
<some network id>
$
```

Containers on a macvlan network appear to the rest of the LAN as independent hosts with their own MAC addresses,
so DHCP reservations by MAC address work as expected.

### ipvlan mode

ipvlan mode is configured the same way as macvlan, but uses L2 ipvlan instead. All containers share the host NIC's MAC
address. Use this only when MAC spoofing is not permitted by the NIC or upstream switch.

```
$ docker network create -d ghcr.io/devplayer0/docker-net-dhcp:release-linux-amd64 --ipam-driver null \
    -o bridge=eth0 \
    -o mode=ipvlan \
    my-dhcp-ipvlan
<some network id>
$
```

_Note: The `null` IPAM driver **must** be used in all modes, or else Docker will try to allocate IP addresses from its
choice of subnet — this can cause IP conflicts since the interface is connected to your local network!_

## Container creation

Once you've set up a network, you can create some containers:

```
$ docker run --rm -ti --network my-dhcp-net alpine
/ # ip address show
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
159: my-bridge0@if160: <BROADCAST,MULTICAST,UP,LOWER_UP,M-DOWN> mtu 1500 qdisc noqueue state UP qlen 1000
    link/ether 86:41:68:f8:85:b9 brd ff:ff:ff:ff:ff:ff
    inet 10.255.0.246/24 brd 10.255.0.255 scope global test0
       valid_lft forever preferred_lft forever
/ # ip route show
default via 10.255.0.123 dev my-bridge0
10.255.0.0/24 dev my-bridge0 scope link  src 10.255.0.246
/ #
```

In macvlan/ipvlan mode the container interface is named `eth0` instead of `<bridge>0`.

Or, in a Docker Compose file:

```yaml
version: '3'
services:
  app:
    hostname: my-http
    image: nginx
    mac_address: 86:41:68:f8:85:b9
    networks:
      - dhcp
networks:
  dhcp:
    external:
      name: my-dhcp-net
```

The above Compose file assumes your network has already been created with `docker network create`. **This is the
recommended way to use `docker-net-dhcp`**, since it allows the network to be shared among multiple compose projects and
other containers. However, you can also create the network as part of the Compose definition. In this case Docker
Compose will manage the network itself (for example deleting it when `docker-compose down` is run).

```yaml
version: '3'
services:
  app:
    image: nginx
    hostname: my-server
    networks:
      - dhcp
networks:
  dhcp:
    driver: ghcr.io/devplayer0/docker-net-dhcp:release-linux-amd64
    driver_opts:
      bridge: my-bridge
      mode: bridge        # optional; defaults to 'bridge'. Use 'macvlan' or 'ipvlan' for direct NIC attachment
      ipv6: 'true'
      ignore_conflicts: 'false'
      skip_routes: 'false'
    ipam:
      driver: 'null'
```

Note:
 - It will take a bit longer than usual for the container to start, as a DHCP lease needs to be obtained before creating it
 - Once created, a persistent DHCP client will renew the DHCP lease (and then update the default gateway in the
   container) when necessary - **this client runs separately from the container**
 - Use `--mac-address` to specify a MAC address if you've configured reserved IP addresses on your DHCP server, or if
   you want a container to re-use an old lease. Not supported in ipvlan mode (all containers share the parent NIC's MAC).
 - Add `--hostname my-host` to have the DHCP transmit this name as the host for the container. This is useful if your
   DHCP server is configured to update DNS records from DHCP leases.
 - If the `docker run` command times out waiting for a lease, you can try increasing the initial timeout value by
   passing `-o lease_timeout=60s` when creating the network (e.g. to increase to 60 seconds)
 - By default, a bridge can only be used for a single DHCP network (bridge mode only). There is additionally a check to
   see if a bridge is used by any other Docker networks. To disable this check (it's also possible this check might
   mistakenly detect a conflict), pass `-o ignore_conflicts=true` when creating the network.
 - `docker-net-dhcp` will try to copy static routes from the host bridge to the container (bridge mode only). To disable
   this behaviour, pass `-o skip_routes=true` when creating the network.

## Debugging

To read the plugin's log, do `cat /var/lib/docker/plugins/*/rootfs/var/log/net-dhcp.log` (as `root`). You can also use
`docker plugin set ghcr.io/devplayer0/docker-net-dhcp:release-linux-amd64 LOG_LEVEL=trace` to increase log verbosity.

# Implementation

## Bridge mode

The same mechanism used by Docker's `bridge` driver is used to wire up networking to containers: a bridge on the host
acts as a switch, and `veth` pairs connect each container's network namespace to it.

- While Docker creates and manages its own bridges (and routes and filters traffic), `net-dhcp` uses an existing bridge
  on the host, bridged with the desired local network.
- Instead of allocating IP addresses from a static pool stored on the Docker host, `net-dhcp` relies on an external DHCP
  server to provide IP addresses.

### Flow

1. Container creation request is made
2. A `veth` pair is created and the host end is connected to the bridge (at this point both interfaces are still in the
host namespace)
3. A DHCP client (BusyBox `udhcpc`) is started on the container end (still in the host namespace) — initial IP address
is provided to Docker by the plugin
4. Docker moves the container end of the `veth` pair into the container's network namespace and sets the IP address — at
this point `udhcpc` must be stopped
5. `net-dhcp` starts `udhcpc` on the container end of the `veth` pair in the container's **network namespace** (but
still in the plugin **PID namespace** — this means that the container can't see the DHCP client)
6. `udhcpc` continues to run, renewing the lease when required, until the container shuts down

## macvlan / ipvlan mode

Instead of a veth pair, a macvlan (or ipvlan L2) sub-interface is created directly on the specified parent NIC. The
sub-interface is created in the host namespace, `udhcpc` acquires an initial lease on it, and then Docker moves it
into the container's network namespace. Traffic travels directly between the sub-interface and the physical NIC without
passing through a Linux bridge, so no `br_netfilter` or iptables rules are applied to it.

Each container in macvlan mode gets a unique MAC address assigned by the kernel (or a user-specified one via
`--mac-address`), so the DHCP server sees each container as a distinct host on the network.

### Flow

1. Container creation request is made
2. A macvlan (or ipvlan) sub-interface is created on the parent NIC in the host namespace
3. A DHCP client (BusyBox `udhcpc`) is started on that interface (still in the host namespace) — initial IP address
is provided to Docker by the plugin
4. Docker moves the sub-interface into the container's network namespace and sets the IP address
5. `net-dhcp` locates the interface in the container namespace by its kernel interface index (which is stable across
namespace moves) and starts a persistent `udhcpc` there (in the plugin's **PID namespace** — invisible to the container)
6. `udhcpc` continues to run, renewing the lease when required, until the container shuts down. When the container
exits its network namespace is destroyed, which automatically removes the sub-interface.

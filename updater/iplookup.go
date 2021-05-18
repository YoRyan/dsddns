package updater

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"
)

const staleTime = 10 * time.Minute

var ipServices []ipService = []ipService{
	icanhazipService{},
	ipifyService{},
	wtfismyipService{}}

type ipSource struct {
	rtype RecordType
	iname string
}

type IPLookup struct {
	cache     map[ipSource](net.IP)
	retrieved map[ipSource](time.Time)
}

func NewIPLookup() IPLookup {
	var lookup IPLookup
	lookup.cache = make(map[ipSource](net.IP))
	lookup.retrieved = make(map[ipSource](time.Time))
	return lookup
}

type dialContext func(context.Context, string, string) (net.Conn, error)

func (l IPLookup) WebFacingIP(ctx context.Context, rtype RecordType, intname string) net.IP {
	key := ipSource{rtype, intname}
	since := l.retrieved[key]
	if time.Now().After(since.Add(staleTime)) {
		// Read all source addresses from the selected interface. If we fail to
		// find any addresses, fall back to automatic selection.
		var addrs []net.IP
		if intf, _ := net.InterfaceByName(intname); intf != nil {
			iaddrs := sourceAddresses(rtype, intf)
			if len(iaddrs) > 0 {
				addrs = iaddrs
			} else {
				addrs = nil
			}
		} else {
			addrs = nil
		}

		// Shuffle our list of IP address services.
		shuffled := make([]ipService, len(ipServices))
		copy(shuffled, ipServices)
		rand.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})

		// Check each source address for each service.
		for _, service := range shuffled {
			var dcs []dialContext
			if addrs == nil {
				dial := net.Dialer{}
				dcs = []dialContext{dial.DialContext}
			} else {
				dcs = make([]dialContext, 0)
				for _, addr := range addrs {
					dcs = append(dcs, dialContextFromAddr(addr))
				}
			}
			for _, dc := range dcs {
				var (
					ip  net.IP
					err error
				)
				switch rtype {
				case ARecord:
					ip, err = service.IPv4Addr(ctx, dc)
				case AAAARecord:
					ip, err = service.IPv6Addr(ctx, dc)
				}
				if err != nil {
					continue
				}
				l.cache[key] = ip
				return ip
			}
		}
	}
	return l.cache[key]
}

func sourceAddresses(rtype RecordType, intf *net.Interface) []net.IP {
	addrs := make([]net.IP, 0)
	iaddrs, err := intf.Addrs()
	if err != nil {
		return addrs
	}
	for _, addr := range iaddrs {
		switch v := addr.(type) {
		case *net.IPNet:
			var oktype bool
			switch rtype {
			case ARecord:
				oktype = isIPv4(v.IP)
			case AAAARecord:
				oktype = isIPv6(v.IP)
			default:
				oktype = false
			}
			if oktype && v.IP.IsGlobalUnicast() {
				addrs = append(addrs, v.IP)
			}
		}
	}
	return addrs
}

func dialContextFromAddr(addr net.IP) dialContext {
	return func(ctx context.Context, network, dialaddr string) (net.Conn, error) {
		var (
			tuple string
			laddr net.Addr
			err   error
		)
		if isIPv4(addr) {
			tuple = addr.String() + ":0"
		} else if isIPv6(addr) {
			tuple = "[" + addr.String() + "]:0"
		}
		switch network {
		case "tcp", "tcp4", "tcp6":
			laddr, err = net.ResolveTCPAddr(network, tuple)
		case "udp", "udp4", "udp6":
			laddr, err = net.ResolveUDPAddr(network, tuple)
		default:
			return nil, errors.New("unknown network")
		}
		if err != nil {
			return nil, err
		}
		dial := net.Dialer{
			LocalAddr: laddr,
		}
		return dial.DialContext(ctx, network, dialaddr)
	}
}

func isIPv4(ip net.IP) bool {
	return len(ip) == 4
}

func isIPv6(ip net.IP) bool {
	return len(ip) == 16
}

type ipService interface {
	IPv4Addr(context.Context, dialContext) (net.IP, error)
	IPv6Addr(context.Context, dialContext) (net.IP, error)
}

type icanhazipService struct{}

func (_ icanhazipService) IPv4Addr(ctx context.Context, dc dialContext) (net.IP, error) {
	return retrieve(ctx, "https://v4.icanhazip.com", dc)
}

func (_ icanhazipService) IPv6Addr(ctx context.Context, dc dialContext) (net.IP, error) {
	return retrieve(ctx, "https://v6.icanhazip.com", dc)
}

type ipifyService struct{}

func (_ ipifyService) IPv4Addr(ctx context.Context, dc dialContext) (net.IP, error) {
	return retrieve(ctx, "https://api.ipify.org", dc)
}

func (_ ipifyService) IPv6Addr(ctx context.Context, dc dialContext) (net.IP, error) {
	return retrieve(ctx, "https://api6.ipify.org", dc)
}

type wtfismyipService struct{}

func (_ wtfismyipService) IPv4Addr(ctx context.Context, dc dialContext) (net.IP, error) {
	return retrieve(ctx, "https://ipv4.wtfismyip.com/text", dc)
}

func (_ wtfismyipService) IPv6Addr(ctx context.Context, dc dialContext) (net.IP, error) {
	return retrieve(ctx, "https://ipv6.wtfismyip.com/text", dc)
}

func retrieve(ctx context.Context, url string, dc dialContext) (net.IP, error) {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: dc,
		},
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("bad response code")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ip := strings.TrimSpace(string(body))
	return net.ParseIP(ip), nil
}

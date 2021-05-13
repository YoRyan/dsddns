package updater

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const staleTime = 10 * time.Minute

var ipServices []ipService = []ipService{
	icanhazipService{}}

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

func (l IPLookup) WebFacingIP(ctx context.Context, rtype RecordType, intname string) net.IP {
	key := ipSource{rtype, intname}
	since := l.retrieved[key]
	if time.Now().After(since.Add(staleTime)) {
		for _, service := range ipServices {
			var (
				ip  net.IP
				err error
			)
			switch rtype {
			case ARecord:
				ip, err = service.IPv4Addr(ctx, intname)
			case AAAARecord:
				ip, err = service.IPv6Addr(ctx, intname)
			}
			if err != nil {
				continue
			}
			l.cache[key] = ip
			return ip
		}
	}
	return l.cache[key]
}

type ipService interface {
	IPv4Addr(context.Context, string) (net.IP, error)
	IPv6Addr(context.Context, string) (net.IP, error)
}

type icanhazipService struct{}

func (_ icanhazipService) IPv4Addr(ctx context.Context, intname string) (net.IP, error) {
	return retrieve(ctx, "https://v4.icanhazip.com", intname)
}

func (_ icanhazipService) IPv6Addr(ctx context.Context, intname string) (net.IP, error) {
	return retrieve(ctx, "https://v6.icanhazip.com", intname)
}

func retrieve(ctx context.Context, url, intname string) (net.IP, error) {
	// TODO use the appropriate interface here
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
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

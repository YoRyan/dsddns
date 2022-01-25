package updater

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	duckCooldown = 15 * time.Minute
)

// DuckService implements the Duck DNS protocol.
type DuckService struct {
	conf *duckServiceConf
}

type duckServiceConf struct {
	Subname string
	Token   string
}

// Submit sends the provided IP address to the dynamic DNS service. In case of failure, it returns a retry delay and the error.
func (s *DuckService) Submit(ctx context.Context, rtype RecordType, ip net.IP) (retryAfter time.Duration, err error) {
	qs := url.Values{}
	qs.Add("domains", s.conf.Subname)
	qs.Add("token", s.conf.Token)
	qs.Add("ip", ip.String())
	requrl := "https://www.duckdns.org/update?" + qs.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", requrl, nil)
	if err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		retryAfter = duckCooldown
		err = errors.New("bad response code")
		return
	}
	return
}

// Identifier returns a human readable name for this service given its endpoint.
func (s *DuckService) Identifier() string {
	return s.conf.Subname + ".duckdns.org"
}

// SupportsRecord determines whether this service supports the provided DNS record type.
func (s *DuckService) SupportsRecord(rtype RecordType) bool {
	switch rtype {
	case ARecord:
		return true
	case AAAARecord:
		return true
	default:
		return false
	}
}

// UnmarshalYAML constructs a service from a YAML configuration.
func (s *DuckService) UnmarshalYAML(value *yaml.Node) error {
	s.conf = &duckServiceConf{}
	if err := value.Decode(s.conf); err != nil {
		return err
	}
	return nil
}

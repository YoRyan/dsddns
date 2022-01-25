package updater

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"time"

	"gopkg.in/yaml.v3"
)

// No-IP dynamic DNS protocol; used by many providers
// see https://www.noip.com/integrate/request

const (
	noIPCooldown = 30 * time.Minute
	noIPForever  = 10 * time.Hour * 24 * 365
)

// NoIPService implements the No-IP protocol. It requires an endpoint.
type NoIPService struct {
	DefinedEndpoint string
	conf            *noIPServiceConf
}

type noIPServiceConf struct {
	Username string
	Password string
	Hostname string
	Endpoint string
}

// Submit sends the provided IP address to the dynamic DNS service. In case of failure, it returns a retry delay and the error.
func (s *NoIPService) Submit(ctx context.Context, rtype RecordType, ip net.IP) (retryAfter time.Duration, err error) {
	qs := url.Values{}
	qs.Add("hostname", s.conf.Hostname)
	qs.Add("myip", ip.String())
	requrl := s.conf.Endpoint + "?" + qs.Encode()

	req, err := http.NewRequestWithContext(ctx, "POST", requrl, nil)
	if err != nil {
		return
	}
	platform := runtime.GOOS + "-" + runtime.GOARCH + "-" + runtime.Version()
	req.SetBasicAuth(s.conf.Username, s.conf.Password)
	req.Header.Set("User-Agent", "DsDDNS/"+platform+" ryan@youngryan.com")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return noIPError(string(body))
	}
	return
}

func noIPError(response string) (retryAfter time.Duration, err error) {
	var text string
	const notAgain = "Will not attempt further updates."
	switch response {
	case "nohost":
		retryAfter = noIPForever
		text = "Hostname supplied does not exist under specified account. " + notAgain
	case "badauth":
		retryAfter = noIPForever
		text = "Invalid username password combination. " + notAgain
	case "badagent":
		retryAfter = noIPForever
		text = "Client disabled. " + notAgain
	case "abuse":
		retryAfter = noIPForever
		text = "Username is blocked due to abuse. " + notAgain
	case "911":
	case "":
		retryAfter = noIPCooldown
		text = "Temporary outage."
	default:
		retryAfter = noIPForever
		text = "Fatal error: " + response + ". " + notAgain
	}
	err = errors.New(text)
	return
}

// Identifier returns a human readable name for this service given its endpoint.
func (s *NoIPService) Identifier() string {
	return s.conf.Hostname
}

// SupportsRecord determines whether this service supports the provided DNS record type.
func (s *NoIPService) SupportsRecord(rtype RecordType) bool {
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
func (s *NoIPService) UnmarshalYAML(value *yaml.Node) error {
	s.conf = &noIPServiceConf{}
	if err := value.Decode(s.conf); err != nil {
		return err
	}
	if s.DefinedEndpoint != "" {
		s.conf.Endpoint = s.DefinedEndpoint
	}
	return nil
}

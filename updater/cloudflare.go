package updater

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"gopkg.in/yaml.v3"
)

const (
	cloudflareCooldown = 15 * time.Minute
)

// CloudflareService implements the Cloudflare DNS protocol.
type CloudflareService struct {
	conf *cloudflareServiceConf
	api  *cloudflare.API
}

type cloudflareServiceConf struct {
	APIKey   string `yaml:"api_key"`
	APIEmail string `yaml:"api_email"`
	APIToken string `yaml:"api_token"`
	ZoneID   string `yaml:"zone_id"`
	RecordID string `yaml:"record_id"`
	Name     string
	TTL      int
}

// Submit sends the provided IP address to the dynamic DNS service. In case of failure, it returns a retry delay and the error.
func (s *CloudflareService) Submit(ctx context.Context, rtype RecordType, ip net.IP) (retryAfter time.Duration, err error) {
	var ttl int
	if s.conf.TTL <= 0 {
		ttl = 1
	} else {
		ttl = s.conf.TTL
	}
	record := cloudflare.DNSRecord{
		Type:    RecordTypeString(rtype),
		Name:    s.conf.Name,
		Content: ip.String(),
		TTL:     ttl,
	}
	err = s.api.UpdateDNSRecord(ctx, s.conf.ZoneID, s.conf.RecordID, record)
	var cfErr cloudflare.APIRequestError
	if errors.As(err, &cfErr) {
		retryAfter = cloudflareCooldown
	}
	return
}

// Identifier returns a human readable name for this service given its endpoint.
func (s *CloudflareService) Identifier() string {
	return s.conf.Name
}

// SupportsRecord determines whether this service supports the provided DNS record type.
func (s *CloudflareService) SupportsRecord(rtype RecordType) bool {
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
func (s *CloudflareService) UnmarshalYAML(value *yaml.Node) error {
	s.conf = &cloudflareServiceConf{}
	err := value.Decode(s.conf)
	if err != nil {
		return err
	}
	if s.conf.APIKey != "" && s.conf.APIEmail != "" {
		s.api, err = cloudflare.New(s.conf.APIKey, s.conf.APIEmail)
		if err != nil {
			return err
		}
	} else if s.conf.APIToken != "" {
		s.api, err = cloudflare.NewWithAPIToken(s.conf.APIToken)
		if err != nil {
			return err
		}
	}
	return nil
}

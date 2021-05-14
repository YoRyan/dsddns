package updater

import (
	"context"
	"net"

	"github.com/cloudflare/cloudflare-go"
	"gopkg.in/yaml.v3"
)

type CloudflareService struct {
	conf *cloudflareServiceConf
	api  *cloudflare.API
}

type cloudflareServiceConf struct {
	ApiKey   string `yaml:"api_key"`
	ApiEmail string `yaml:"api_email"`
	ApiToken string `yaml:"api_token"`
	ZoneID   string `yaml:"zone_id"`
	RecordID string `yaml:"record_id"`
	Name     string
	TTL      int
}

func (s *CloudflareService) Submit(ctx context.Context, rtype RecordType, ip net.IP) error {
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
	if err := s.api.UpdateDNSRecord(ctx, s.conf.ZoneID, s.conf.RecordID, record); err != nil {
		return err
	}
	return nil
}

func (s *CloudflareService) Domain() string {
	return s.conf.Name
}

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

func (s *CloudflareService) UnmarshalYAML(value *yaml.Node) error {
	s.conf = &cloudflareServiceConf{}
	err := value.Decode(s.conf)
	if err != nil {
		return err
	}
	if s.conf.ApiKey != "" && s.conf.ApiEmail != "" {
		s.api, err = cloudflare.New(s.conf.ApiKey, s.conf.ApiEmail)
		if err != nil {
			return err
		}
	} else if s.conf.ApiToken != "" {
		s.api, err = cloudflare.NewWithAPIToken(s.conf.ApiToken)
		if err != nil {
			return err
		}
	}
	return nil
}

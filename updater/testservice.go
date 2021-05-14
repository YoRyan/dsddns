package updater

import (
	"context"
	"net"
	"time"

	"gopkg.in/yaml.v3"
)

type TestService struct{}

func (s *TestService) Submit(_ context.Context, _ RecordType, ip net.IP) (time.Duration, error) {
	return 0, nil
}

func (s *TestService) Identifier() string {
	return "test.example.com"
}

func (s *TestService) SupportsRecord(rtype RecordType) bool {
	return true
}

func (s *TestService) UnmarshalYAML(value *yaml.Node) error {
	return nil
}

// Package updater pushes IP address updates to dynamic DNS services.
package updater

import (
	"context"
	"errors"
	"log"
	"net"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type RecordType int

const (
	ARecord RecordType = iota
	AAAARecord
)

// RecordTypeString returns the string equivalent to a RecordType value.
func RecordTypeString(rtype RecordType) string {
	switch rtype {
	case ARecord:
		return "A"
	case AAAARecord:
		return "AAAA"
	default:
		return ""
	}
}

// An Updater manages a single DNS record.
type Updater struct {
	Type       RecordType
	Interface  string
	Service    RecordService
	IPOffset   net.IP
	IPMaskBits int
	tryAfter   time.Time
	submitted  net.IP
	lookup     IPLookup
	yaml.Unmarshaler
}

func (u *Updater) UnmarshalYAML(value *yaml.Node) error {
	var aux struct {
		Service    string
		Type       string
		Interface  string
		IPSLAAC    string `yaml:"ip_slaac"`
		IPOffset   string `yaml:"ip_offset"`
		IPMaskBits int    `yaml:"ip_mask_bits"`
	}
	if err := value.Decode(&aux); err != nil {
		return err
	}

	switch strings.ToLower(aux.Service) {
	case "cloudflare":
		u.Service = &CloudflareService{}
	case "duck":
		u.Service = &DuckService{}
	case "google":
		u.Service = &GoogleService{}
	default:
		return errors.New("unknown service")
	}
	if err := value.Decode(u.Service); err != nil {
		return err
	}

	switch strings.ToLower(aux.Type) {
	case "a":
		u.Type = ARecord
	case "aaaa":
		u.Type = AAAARecord
	default:
		return errors.New("invalid record type")
	}

	if !u.Service.SupportsRecord(u.Type) {
		return errors.New("service does not support this record type")
	}

	u.Interface = aux.Interface

	if ip := net.ParseIP(aux.IPOffset); ip != nil {
		u.IPOffset = ip
		u.IPMaskBits = aux.IPMaskBits
	}
	if aux.IPSLAAC != "" {
		if mac, err := net.ParseMAC(aux.IPSLAAC); err == nil {
			u.IPOffset = SlaacBits(mac)
			u.IPMaskBits = 64
		}
	}

	return nil
}

// Update attempts to refresh the record if necessary. It should be called every
// few minutes.
func (u *Updater) Update(ctx context.Context, logger *log.Logger) {
	rawip := u.lookup.WebFacingIP(ctx, u.Type, u.Interface)
	if rawip == nil {
		return
	}

	ip := AddIP(MaskIP(rawip, u.IPMaskBits), u.IPOffset)
	if !ip.Equal(u.submitted) && time.Now().After(u.tryAfter) {
		id := u.Service.Identifier()
		logger.Println(id, RecordTypeString(u.Type), "➤", ip.String())

		if retryAfter, err := u.Service.Submit(ctx, u.Type, ip); err != nil {
			logger.Println(id, "✗", err)
			logger.Println(id, "next attempt in", retryAfter.String())
			u.tryAfter = time.Now().Add(retryAfter)
		} else {
			u.submitted = ip
		}
	}
}

// DryRun performs an IP address lookup, but does not refresh the record.
func (u *Updater) DryRun(ctx context.Context, logger *log.Logger) {
	rawip := u.lookup.WebFacingIP(ctx, u.Type, u.Interface)
	if rawip == nil {
		log.Println("failed to look up IP address")
		return
	}

	ip := AddIP(MaskIP(rawip, u.IPMaskBits), u.IPOffset)
	logger.Println(u.Service.Identifier(), RecordTypeString(u.Type), "➤", ip.String())
}

// SlaacBits returns an IPv6 address with the lower 64 bits derived from the
// provided MAC address using the EUI-64 derivation.
func SlaacBits(mac net.HardwareAddr) net.IP {
	return []byte{
		0, 0, 0, 0,
		0, 0, 0, 0,
		mac[0] ^ 2, mac[1], mac[2], 0xff,
		0xfe, mac[3], mac[4], mac[5],
	}
}

// MaskIP zeroes out the specified number of lower bits in the provided IP
// address.
func MaskIP(ip net.IP, mask int) net.IP {
	return ip.Mask(net.CIDRMask(len(ip)*8-mask, len(ip)*8))
}

// AddIP adds together the hexadecimal contents of the provided IP addresses.
func AddIP(a net.IP, b net.IP) net.IP {
	var sz int
	if len(a) > len(b) {
		sz = len(a)
	} else {
		sz = len(b)
	}
	sum := make([]byte, sz)
	rem := 0
	for i, p := sz-1, 0; i >= 0; i, p = i-1, p+1 {
		b := int(fromRight(a, p)) + int(fromRight(b, p)) + rem
		rem = b / 256
		sum[i] = byte(b - b/256*256)
	}
	return net.IP(sum)
}

func fromRight(ip net.IP, place int) byte {
	if place >= len(ip) {
		return 0
	} else {
		return ip[len(ip)-place-1]
	}
}

// Updaters represents a slice of updaters defined by a YAML configuration. All
// updaters share the same IP address lookup cache.
type Updaters []*Updater

func (u *Updaters) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.SequenceNode {
		return errors.New("expected a YAML sequence")
	}

	lookup := NewIPLookup()
	for _, node := range value.Content {
		var updater Updater
		if err := node.Decode(&updater); err != nil {
			return err
		}
		updater.lookup = lookup
		*u = append(*u, &updater)
	}
	return nil
}

// Update processes all of the updaters in this slice.
func (u *Updaters) Update(ctx context.Context, logger *log.Logger) {
	for _, updater := range *u {
		updater.Update(ctx, logger)
	}
}

// DryRun tests all of the updaters in this slice.
func (u *Updaters) DryRun(ctx context.Context, logger *log.Logger) {
	for _, updater := range *u {
		updater.DryRun(ctx, logger)
	}
}

// A RecordService manages transactions concerning a particular record with a
// dynamic DNS service.
type RecordService interface {
	// Submit a new record value.
	Submit(context.Context, RecordType, net.IP) (retryAfter time.Duration, err error)

	// Retrieve a human-readable name for this record.
	Identifier() string

	// Determine support for a record type.
	SupportsRecord(RecordType) bool

	yaml.Unmarshaler
}

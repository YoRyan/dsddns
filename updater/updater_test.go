package updater

import (
	"net"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSlaacBits(t *testing.T) {
	got := SlaacBits([]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66})
	slaac := net.IP([]byte{
		0, 0, 0, 0, 0, 0, 0, 0,
		0x13, 0x22, 0x33, 0xff, 0xfe, 0x44, 0x55, 0x66,
	})
	if !got.Equal(slaac) {
		t.Errorf("SLAAC bits = %s; want %s", got.String(), slaac.String())
	}
}

func TestMaskIP(t *testing.T) {
	got := MaskIP([]byte{192, 168, 12, 34}, 0)
	masked := net.IP([]byte{192, 168, 12, 34})
	if !got.Equal(masked) {
		t.Errorf("Masked IP = %s; want %s", got.String(), masked.String())
	}

	got = MaskIP([]byte{192, 168, 12, 34}, 8)
	masked = net.IP([]byte{192, 168, 12, 0})
	if !got.Equal(masked) {
		t.Errorf("Masked IP = %s; want %s", got.String(), masked.String())
	}

	got = MaskIP([]byte{192, 168, 86, 34}, 12)
	masked = net.IP([]byte{192, 168, 80, 0})
	if !got.Equal(masked) {
		t.Errorf("Masked IP = %s; want %s", got.String(), masked.String())
	}
}

func TestAddIP(t *testing.T) {
	got := AddIP([]byte{0, 0, 0, 0}, []byte{1, 2, 3, 4})
	added := net.IP([]byte{1, 2, 3, 4})
	if !got.Equal(added) {
		t.Errorf("Summed IP = %s; want %s", got.String(), added.String())
	}

	got = AddIP([]byte{1, 2, 3, 4}, []byte{5, 6, 7, 8})
	added = net.IP([]byte{6, 8, 10, 12})
	if !got.Equal(added) {
		t.Errorf("Summed IP = %s; want %s", got.String(), added.String())
	}

	got = AddIP([]byte{0, 255, 156, 1}, []byte{0, 1, 100, 2})
	added = net.IP([]byte{1, 1, 0, 3})
	if !got.Equal(added) {
		t.Errorf("Summed IP = %s; want %s", got.String(), added.String())
	}
}

func TestUnmarshalUpdaters(t *testing.T) {
	data := []byte(`
- service: "test"
  type: "AAAA"
  interface: "eth0"
  ipoffset: "::1"
  ipmaskbits: 64`)
	var got Updaters
	err := yaml.Unmarshal(data, &got)
	if err != nil {
		t.Error(err)
	}
	if len(got) != 1 {
		t.Errorf("Number of updaters = %d; want 1", len(got))
	}
	if got[0].Type != AAAARecord {
		t.Error("Record type should be AAAA")
	}
	if got[0].Interface != "eth0" {
		t.Errorf("Interface = %s; want eth0", got[0].Interface)
	}
	if !got[0].IPOffset.Equal(net.ParseIP("::1")) {
		t.Errorf("Offset IP = %s; want ::1", got[0].IPOffset.String())
	}
}

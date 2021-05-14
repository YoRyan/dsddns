package updater

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"gopkg.in/yaml.v3"
)

// Google's Dynamic DNS API
// see https://support.google.com/domains/answer/6147083?hl=en

const (
	googleCooldown = 15 * time.Minute
)

type GoogleService struct {
	conf *googleServiceConf
}

type googleServiceConf struct {
	Username string
	Password string
	Hostname string
}

func (s *GoogleService) Submit(ctx context.Context, rtype RecordType, ip net.IP) (retryAfter time.Duration, err error) {
	qs := url.Values{}
	qs.Add("hostname", s.conf.Hostname)
	qs.Add("myip", ip.String())
	requrl := "https://domains.google.com/nic/update?" + qs.Encode()

	req, err := http.NewRequestWithContext(ctx, "POST", requrl, nil)
	if err != nil {
		return
	}
	req.SetBasicAuth(s.conf.Username, s.conf.Password)
	req.Header.Set("User-Agent", "Chrome/41.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		retryAfter = googleCooldown
		body, _ := io.ReadAll(resp.Body)
		err = googleError(string(body))
		return
	}
	return
}

func googleError(response string) error {
	var text string
	switch response {
	case "nohost":
		text = "The hostname does not exist, or does not have Dynamic DNS enabled."
	case "badauth":
		text = "The username / password combination is not valid for the specified host."
	case "notfqdn":
		text = "The supplied hostname is not a valid fully-qualified domain name."
	case "badagent":
		text = "Your Dynamic DNS client is making bad requests. Ensure the user agent is set in the request."
	case "abuse":
		text = "Dynamic DNS access for the hostname has been blocked due to failure to interpret previous responses correctly."
	default:
		text = "unknown error"
	}
	return errors.New(text)
}

func (s *GoogleService) Identifier() string {
	return s.conf.Hostname
}

func (s *GoogleService) SupportsRecord(rtype RecordType) bool {
	switch rtype {
	case ARecord:
		return true
	case AAAARecord:
		return true
	default:
		return false
	}
}

func (s *GoogleService) UnmarshalYAML(value *yaml.Node) error {
	s.conf = &googleServiceConf{}
	if err := value.Decode(s.conf); err != nil {
		return err
	}
	return nil
}

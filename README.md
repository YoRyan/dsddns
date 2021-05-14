# dsddns

DsDDNS is the Dual-Stack Dynamic DNS client. A dynamic DNS client keeps your DNS records in sync with the IP addresses associated with your home Internet connection, which are subject to reassignment by your ISP.

DsDDNS is the world's first dynamic DNS client built from the ground-up for IPv6. In comparison to other dynamic DNS clients available on the Internet, including some others written in Go, DsDDNS features:

- First-class support for IPv6, including the ability to update multiple hostnames within a dynamically allocated prefix.
- Support for determining the lower bits of the address either manually or with SLAAC.
- Support for managing multiple domains across multiple accounts and services.
- A single binary and a single configuration file that can be deployed anywhere.
- A YAML configuration format that supports the DRY ("don't repeat yourself") principle.

Currently, DsDDNS can manage A and AAAA records for the following services:

- [Cloudflare](https://www.cloudflare.com/dns/)
- [Duck DNS](https://www.duckdns.org/)
- [Google Domains](https://domains.google/)

## Installation

```
$ go get -u github.com/YoRyan/dsddns
```

## Configuration

The configuration file is in YAML format. Its path should be supplied to as the sole argument to DsDDNS:

```
dsddns /etc/dsddns.conf
```

This file contains a list of records to manage under the `records` key. Here is a simple example configuration:

```yaml
records:
  - type: A
    service: cloudflare,
    api_token: XXXXXXXXXXXXXXXXXX_XXXXXXXXXXXXXXXXXXXXX
    record_id: XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
    zone_id: XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
    name: ipv4.youngryan.com
  - type: AAAA
    service: cloudflare,
    api_token: XXXXXXXXXXXXXXXXXX_XXXXXXXXXXXXXXXXXXXXX
    record_id: XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
    zone_id: XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
    name: ipv6.youngryan.com
```

### Common fields

Some keys apply to all kinds of records, regardless of service. The following keys *must* be specified:

| Key | Type | Value |
| --- | --- | --- |
| type | string | Specifies the type of DNS record. Can be `A` (IPv4) or `AAAA` (IPv6). |
| service | string | <p>Specifies the dynamic DNS service that manages this record. Must be one of the following values:</p><ul><li>`cloudflare`</li><li>`duck`</li><li>`google`</li></ul> |

The following keys are optional:

| Key | Type | Value |
| --- | --- | --- |
| interface | string | Selects the source network interface to use when requesting the current IP address. The interface should be specified by its name, such as `eth0`. If it is not specified, the operating system selects the interface. |
| ip_mask_bits | number | Zeroes out the specified number of lower bits from the IP address. The value `64` can be used to zero out the interface identifier portion (right half) of an IPv6 address.
| ip_offset | string | Sets the lower bits of the IP address once they have been masked with `ip_mask_bits`. The value should be an "offset" IP address, such as `::1`, which will be added to the masked address.
| ip_slaac | string | Sets the lower 64 bits of the IP address using the provided MAC address, such as `11:22:33:44:55:66`. The EUI-64 method is used, matching the addresses generated by SLAAC. This setting overrides `ip_mask_bits` and `ip_offset`.

### Per-service fields

Other keys only apply to records managed by specific services. Some of them are mandatory.

<details>
<summary>Cloudflare</summary>

Cloudflare API requests can be authenticated using your account's global API key or by issuing individual API tokens. Specify a global API key or an API token, but not both.

The following keys are mandatory for Cloudflare-managed records:

| Key | Type | Value |
| --- | --- | --- |
| api_key | string | If using your global API key, provide it here. |
| api_email | string | If using your global API key, provide your Cloudflare login here. |
| api_token | string | If using an API token, provide it here. |
| name | string | Specify the full domain managed by this record, including its suffix. |
| zone_id | string | Specify the identifier of your domain's DNS zone. You can obtain this with the [List Zones](https://api.cloudflare.com/#zone-list-zones) API call. |
| record_id | string | Specify the identifier of your DNS record. You can obtain this with the [List DNS Records](https://api.cloudflare.com/#dns-records-for-a-zone-list-dns-records) API call. |

The following keys are optional:

| Key | Type | Value |
| --- | --- | --- |
| ttl | number | Sets the TTL for this record's updates. If it is not specified, the value 1 (automatic) is used. |
</details>

<details>
<summary>Duck DNS</summary>

The following keys are mandatory for Duck DNS-managed records:

| Key | Type | Value |
| --- | --- | --- |
| subname | string | The domain managed by this record. Should not include the ".duckdns.org" suffix. |
| token | string | The API token for this dynamic DNS client. |
</details>

<details>
<summary>Google Domains</summary>

To [use a dynamic DNS client](https://support.google.com/domains/answer/6147083?hl=en) with Google Domains, you have to set up a synthetic record for the hostname you want to manage and then generate a username/password combination for the client.

The following keys are mandatory for Google-managed records:

| Key | Type | Value |
| --- | --- | --- |
| username | string | The username generated for this dynamic DNS client. |
| password | string | The password generated for this client. |
| hostname | string | The FQDN for this record. |
</details>

### Avoiding repetition with merge keys

Because the configuration file uses YAML, you can use YAML's anchor, alias, and [merge key](https://yaml.org/type/merge.html) features to consolidate information that repeats itself.

Here is an example of a configuration file that uses merge keys to centralize authentication information:

```yaml
my_merges:
  - &CfYoungryanCom {
      service: cloudflare,
      api_token: XXXXXXXXXXXXXXXXXX_XXXXXXXXXXXXXXXXXXXXX,
      zone_id: XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
    }
records:
  - << : *CfYoungryanCom
    type: AAAA
    record_id: XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
    name: radio-free.youngryan.com
    ip_mask_bits: 64
    ip_offset: ::1
  - << : *CfYoungryanCom
    type: AAAA
    record_id: XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
    name: arvin.radio-free.youngryan.com
    ip_slaac: c8:5b:76:xx:xx:xx
  - << : *CfYoungryanCom
    type: AAAA
    record_id: XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
    name: shafter.radio-free.youngryan.com
    ip_slaac: 10:60:4b:xx:xx:xx
```
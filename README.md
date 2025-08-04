# GeoIP2

Provides middleware for resolving a users IP address against multiple Maxmind Geo IP databases with self update support

## Build

```sh
xcaddy  \
  --with github.com/relvacode/caddy-geoip2
```

## Caddyfile example

```
{
  order geoip2 first

  # accountId, licenseKey and updateUrl and updateFrequency are only required for automatic updates
  geoip2 {
    account_id         "{env.GEO_ACCOUNT_ID}"
    license_key        "{env.GEO_API_KEY}"
    database_directory "/tmp/"
    edition_id         GeoLite2-City
    edition_id         GeoLite2-ASN
    update_url         "https://updates.maxmind.com"
    update_frequency   604800   # in seconds
  }
}

localhost {
  geoip2

  # Add country and state code to the header
  header geoip-country "{geoip2.country_code}"

  @geofilter expression `{geoip2.country_code} in ["CN", "IR", "RU"]`
  
  error @geofilter "Blocked by geographic location" 403
}

```

## Variables

### Country

Supported with the `GeoLite2-City` and `GeoLite2-Country` editions

- `geoip2.country_code`
- `geoip2.country_name`
- `geoip2.country_eu`
- `geoip2.continent_code`
- `geoip2.continent_name`

### City

Supported with the `GeoLite2-City` edition

- `geoip2.city_name`
- `geoip2.postal_code`
- `geoip2.location_latitude`
- `geoip2.location_longitude`
- `geoip2.location_timezone`
- `geoip2.location_accuracy_radius`

### ASN

Supported with the `GeoLite2-ASN` edition

- `geoip2.asn_network`
- `geoip2.asn_organisation`
- `geoip2.asn_system_number`

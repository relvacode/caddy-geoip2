package geoip2

type GeoIP2Record struct {
	Country struct {
		Locales           []string          `json:"locales"`
		Confidence        uint16            `maxminddb:"confidence"`
		ISOCode           string            `maxminddb:"iso_code"`
		IsInEuropeanUnion bool              `maxminddb:"is_in_european_union"`
		Names             map[string]string `maxminddb:"names"`
		GeoNameID         uint64            `maxminddb:"geoname_id"`
	} `maxminddb:"country"`

	Continent struct {
		Locales   []string          `json:"locales"`
		Code      string            `maxminddb:"code"`
		GeoNameID uint              `maxminddb:"geoname_id"`
		Names     map[string]string `maxminddb:"names"`
	} `maxminddb:"continent"`

	City struct {
		Names      map[string]string `maxminddb:"names"`
		Confidence uint16            `maxminddb:"confidence"`
		GeoNameID  uint64            `maxminddb:"geoname_id"`
		Locales    []string          `json:"locales"`
	} `maxminddb:"city"`

	Location struct {
		AccuracyRadius    uint16  `maxminddb:"accuracy_radius"`
		AverageIncome     uint16  `maxminddb:"average_income"`
		Latitude          float64 `maxminddb:"latitude"`
		Longitude         float64 `maxminddb:"longitude"`
		MetroCode         uint    `maxminddb:"metro_code"`
		PopulationDensity uint    `maxminddb:"population_density"`
		TimeZone          string  `maxminddb:"time_zone"`
	} `maxminddb:"location"`

	Postal struct {
		Code       string `maxminddb:"code"`
		Confidence uint16 `maxminddb:"confidence"`
	} `maxminddb:"postal"`

	RegisteredCountry struct {
		GeoNameID         uint              `maxminddb:"geoname_id"`
		IsInEuropeanUnion bool              `maxminddb:"is_in_european_union"`
		IsoCode           string            `maxminddb:"iso_code"`
		Names             map[string]string `maxminddb:"names"`
	} `maxminddb:"registered_country"`

	RepresentedCountry struct {
		Locales           []string          `json:"locales"`
		Confidence        uint16            `maxminddb:"confidence"`
		GeoNameID         uint              `maxminddb:"geoname_id"`
		IsInEuropeanUnion bool              `maxminddb:"is_in_european_union"`
		IsoCode           string            `maxminddb:"iso_code"`
		Names             map[string]string `maxminddb:"names"`
		Type              string            `maxminddb:"type"`
	} `maxminddb:"represented_country"`

	Subdivisions []struct {
		Locales    []string          `json:"locales"`
		Confidence uint16            `maxminddb:"confidence"`
		GeoNameID  uint              `maxminddb:"geoname_id"`
		IsoCode    string            `maxminddb:"iso_code"`
		Names      map[string]string `maxminddb:"names"`
	} `maxminddb:"subdivisions"`

	Traits struct {
		IsAnonymousProxy    bool `maxminddb:"is_anonymous_proxy"`
		IsAnonymousVpn      bool `maxminddb:"is_anonymous_vpn"`
		IsSatelliteProvider bool `maxminddb:"is_satellite_provider"`

		AutonomousSystemNumber       uint64 `maxminddb:"autonomous_system_number"`
		AutonomousSystemOrganization string `maxminddb:"autonomous_system_organization"`
		ConnectionType               string `maxminddb:"connection_type"`
		Domain                       string `maxminddb:"domain"`

		IsHostingProvider  bool    `maxminddb:"is_hosting_provider"`
		IsLegitimateProxy  bool    `maxminddb:"is_legitimate_proxy"`
		IsPublicProxy      bool    `maxminddb:"is_public_proxy"`
		IsResidentialProxy bool    `maxminddb:"is_residential_proxy"`
		IsTorExitNode      bool    `maxminddb:"is_tor_exit_node"`
		Isp                string  `maxminddb:"isp"`
		MobileCountryCode  string  `maxminddb:"mobile_country_code"`
		MobileNetworkCode  string  `maxminddb:"mobile_network_code"`
		Network            string  `maxminddb:"network"`
		Organization       string  `maxminddb:"organization"`
		UserType           string  `maxminddb:"user_type"`
		UserCount          int32   `maxminddb:"userCount"`
		StaticIpScore      float64 `maxminddb:"static_ip_score"`
	} `maxminddb:"traits"`
}

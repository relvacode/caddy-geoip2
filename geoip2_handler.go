package geoip2

import (
	"fmt"
	"go.uber.org/zap"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

type Handler struct {
	state *GeoIp2
	ctx   caddy.Context
}

func init() {
	caddy.RegisterModule(new(Handler))
	httpcaddyfile.RegisterHandlerDirective("geoip2", parseCaddyfile)
}

func (*Handler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.geoip2",
		New: func() caddy.Module { return new(Handler) },
	}
}

func (m *Handler) ClientIP(r *http.Request) (netip.Addr, error) {
	// if handshake is not finished, we infer 0-RTT that has
	// not verified remote IP; could be spoofed, so we throw
	// HTTP 425 status to tell the client to try again after
	// the handshake is complete
	if r.TLS != nil && !r.TLS.HandshakeComplete {
		return netip.IPv4Unspecified(), caddyhttp.Error(http.StatusTooEarly, fmt.Errorf("TLS handshake not complete, remote IP cannot be verified"))
	}

	address := caddyhttp.GetVar(r.Context(), caddyhttp.ClientIPVarKey).(string)

	ipStr, _, err := net.SplitHostPort(address)
	if err != nil {
		ipStr = address // OK; probably didn't have a port
	}

	// Some IPv6-Addresses can contain zone identifiers at the end,
	// which are separated with "%"
	if strings.Contains(ipStr, "%") {
		split := strings.Split(ipStr, "%")
		ipStr = split[0]
	}

	ipAddr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return netip.IPv4Unspecified(), err
	}

	return ipAddr, nil
}

func (m *Handler) bindRequest(r *http.Request, repl *caddy.Replacer) {
	clientIP, _ := m.ClientIP(r)

	if clientIP.IsUnspecified() {
		caddy.Log().Named(ModuleName).Error("No client IP could be resolved from the request")
		return
	}

	record, err := m.state.Lookup(clientIP.AsSlice())
	if err != nil {
		caddy.Log().Named(ModuleName).Error("Failed to lookup geoip2 record", zap.String("ip", clientIP.String()), zap.Error(err))
		return
	}

	repl.Set("geoip2.ip_address", clientIP.String())

	//country
	repl.Set("geoip2.country_code", record.Country.ISOCode)

	for key, element := range record.Country.Names {
		repl.Set("geoip2.country_names_"+key, element)
		if key == "en" {
			repl.Set("geoip2.country_name", element)
		}
	}

	repl.Set("geoip2.country_eu", record.Country.IsInEuropeanUnion)
	repl.Set("geoip2.country_locales", record.Country.Locales)
	repl.Set("geoip2.country_confidence", record.Country.Confidence)
	repl.Set("geoip2.country_names", record.Country.Names)
	repl.Set("geoip2.country_geoname_id", record.Country.GeoNameID)

	//Continent
	repl.Set("geoip2.continent_code", record.Continent.Code)
	repl.Set("geoip2.continent_locales", record.Continent.Locales)
	repl.Set("geoip2.continent_names", record.Continent.Names)
	repl.Set("geoip2.continent_geoname_id", record.Continent.GeoNameID)

	for key, element := range record.Continent.Names {
		repl.Set("geoip2.continent_names_"+key, element)
		if key == "en" {
			repl.Set("geoip2.continent_name", element)
		}
	}

	//City
	repl.Set("geoip2.city_confidence", record.City.Confidence)
	repl.Set("geoip2.city_locales", record.City.Locales)
	repl.Set("geoip2.city_names", record.City.Names)
	repl.Set("geoip2.city_geoname_id", record.City.GeoNameID)
	// val, _ = record.City.Names["en"]
	// repl.Set("geoip2.city_name", val)

	for key, element := range record.City.Names {
		repl.Set("geoip2.city_names_"+key, element)
		if key == "en" {
			repl.Set("geoip2.city_name", element)
		}
	}

	//Location
	repl.Set("geoip2.location_latitude", record.Location.Latitude)
	repl.Set("geoip2.location_longitude", record.Location.Longitude)
	repl.Set("geoip2.location_time_zone", record.Location.TimeZone)
	repl.Set("geoip2.location_accuracy_radius", record.Location.AccuracyRadius)
	repl.Set("geoip2.location_average_income", record.Location.AverageIncome)
	repl.Set("geoip2.location_metro_code", record.Location.MetroCode)
	repl.Set("geoip2.location_population_density", record.Location.PopulationDensity)

	//Postal
	repl.Set("geoip2.postal_code", record.Postal.Code)
	repl.Set("geoip2.postal_confidence", record.Postal.Confidence)

	//RegisteredCountry
	repl.Set("geoip2.registeredcountry_geoname_id", record.RegisteredCountry.GeoNameID)
	repl.Set("geoip2.registeredcountry_is_in_european_union", record.RegisteredCountry.IsInEuropeanUnion)
	repl.Set("geoip2.registeredcountry_iso_code", record.RegisteredCountry.IsoCode)
	repl.Set("geoip2.registeredcountry_names", record.RegisteredCountry.Names)
	// val, _ = record.RegisteredCountry.Names["en"]
	// repl.Set("geoip2.registeredcountry_name", val)

	for key, element := range record.RegisteredCountry.Names {
		repl.Set("geoip2.registeredcountry_names_"+key, element)
		if key == "en" {
			repl.Set("geoip2.registeredcountry_name", element)
		}
	}

	//RepresentedCountry
	repl.Set("geoip2.representedcountry_geoname_id", record.RepresentedCountry.GeoNameID)
	repl.Set("geoip2.representedcountry_is_in_european_union", record.RepresentedCountry.IsInEuropeanUnion)
	repl.Set("geoip2.representedcountry_iso_code", record.RepresentedCountry.IsoCode)
	repl.Set("geoip2.representedcountry_names", record.RepresentedCountry.Names)
	repl.Set("geoip2.representedcountry_locales", record.RepresentedCountry.Locales)
	repl.Set("geoip2.representedcountry_confidence", record.RepresentedCountry.Confidence)
	repl.Set("geoip2.representedcountry_type", record.RepresentedCountry.Type)
	// val, _ = record.RepresentedCountry.Names["en"]
	// repl.Set("geoip2.representedcountry_name", val)

	for key, element := range record.RepresentedCountry.Names {
		repl.Set("geoip2.representedcountry_names_"+key, element)
		if key == "en" {
			repl.Set("geoip2.representedcountry_name", element)
		}
	}

	repl.Set("geoip2.subdivisions", record.Subdivisions)

	for index, subdivision := range record.Subdivisions {
		indexStr := strconv.Itoa(index + 1)
		repl.Set("geoip2.subdivisions_"+indexStr+"_confidence", subdivision.Confidence)
		repl.Set("geoip2.subdivisions_"+indexStr+"_geoname_id", subdivision.GeoNameID)
		repl.Set("geoip2.subdivisions_"+indexStr+"_iso_code", subdivision.IsoCode)
		repl.Set("geoip2.subdivisions_"+indexStr+"_locales", subdivision.Locales)
		repl.Set("geoip2.subdivisions_"+indexStr+"_names", subdivision.Names)
		for key, element := range subdivision.Locales {
			keyStr := strconv.Itoa(key)
			repl.Set("geoip2.subdivisions_"+indexStr+"_locales_"+keyStr, element)
		}
		for key, element := range subdivision.Names {
			repl.Set("geoip2.subdivisions_"+indexStr+"_names_"+key, element)
			if key == "en" {
				repl.Set("geoip2.subdivisions_"+indexStr+"_name", element)
			}
		}
	}

	//Traits
	repl.Set("geoip2.traits_is_anonymous_proxy", record.Traits.IsAnonymousProxy)
	repl.Set("geoip2.traits_is_anonymous_vpn", record.Traits.IsAnonymousVpn)
	repl.Set("geoip2.traits_is_satellite_provider", record.Traits.IsSatelliteProvider)
	repl.Set("geoip2.traits_autonomous_system_number", record.Traits.AutonomousSystemNumber)
	repl.Set("geoip2.traits_autonomous_system_organization", record.Traits.AutonomousSystemOrganization)

	//Traits
	repl.Set("geoip2.traits_connection_type", record.Traits.ConnectionType)
	repl.Set("geoip2.traits_domain", record.Traits.Domain)
	repl.Set("geoip2.traits_is_hosting_provider", record.Traits.IsHostingProvider)
	repl.Set("geoip2.traits_is_legitimate_proxy", record.Traits.IsLegitimateProxy)
	repl.Set("geoip2.traits_is_public_proxy", record.Traits.IsPublicProxy)
	repl.Set("geoip2.traits_is_residential_proxy", record.Traits.IsResidentialProxy)
	repl.Set("geoip2.traits_is_tor_exit_node", record.Traits.IsTorExitNode)
	repl.Set("geoip2.traits_isp", record.Traits.Isp)
	repl.Set("geoip2.traits_mobile_country_code", record.Traits.MobileCountryCode)
	repl.Set("geoip2.traits_mobile_network_code", record.Traits.MobileNetworkCode)
	repl.Set("geoip2.traits_network", record.Traits.Network)
	repl.Set("geoip2.traits_organization", record.Traits.Organization)
	repl.Set("geoip2.traits_user_type", record.Traits.UserType)
	repl.Set("geoip2.traits_userCount", record.Traits.UserCount)
	repl.Set("geoip2.traits_static_ip_score", record.Traits.StaticIpScore)
}

func (m *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	repl := r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)
	//init some variables with default value ""
	repl.Set("geoip2.ip_address", "")
	repl.Set("geoip2.country_code", "")
	repl.Set("geoip2.country_name", "")
	repl.Set("geoip2.country_eu", "")
	repl.Set("geoip2.country_locales", "")
	repl.Set("geoip2.country_confidence", "")
	repl.Set("geoip2.country_names", "")

	repl.Set("geoip2.country_names_0", "")
	repl.Set("geoip2.country_names_1", "")
	repl.Set("geoip2.country_geoname_id", "")
	repl.Set("geoip2.continent_code", "")
	repl.Set("geoip2.continent_locales", "")
	repl.Set("geoip2.continent_names", "")

	repl.Set("geoip2.continent_names_0", "")
	repl.Set("geoip2.continent_names_1", "")

	repl.Set("geoip2.continent_geoname_id", "")
	repl.Set("geoip2.continent_name", "")
	repl.Set("geoip2.city_confidence", "")
	repl.Set("geoip2.city_locales", "")
	repl.Set("geoip2.city_names", "")
	repl.Set("geoip2.city_names_0", "")
	repl.Set("geoip2.city_names_1", "")
	repl.Set("geoip2.city_geoname_id", "")
	// repl.Set("geoip2.city_name", val)
	repl.Set("geoip2.city_name", "")
	repl.Set("geoip2.location_latitude", "")
	repl.Set("geoip2.location_longitude", "")
	repl.Set("geoip2.location_time_zone", "")
	repl.Set("geoip2.location_accuracy_radius", "")
	repl.Set("geoip2.location_average_income", "")
	repl.Set("geoip2.location_metro_code", "")
	repl.Set("geoip2.location_population_density", "")
	repl.Set("geoip2.postal_code", "")
	repl.Set("geoip2.postal_confidence", "")
	repl.Set("geoip2.registeredcountry_geoname_id", "")
	repl.Set("geoip2.registeredcountry_is_in_european_union", "")
	repl.Set("geoip2.registeredcountry_iso_code", "")
	repl.Set("geoip2.registeredcountry_names", "")
	repl.Set("geoip2.registeredcountry_names_0", "")
	repl.Set("geoip2.registeredcountry_names_1", "")

	repl.Set("geoip2.registeredcountry_name", "")
	repl.Set("geoip2.representedcountry_geoname_id", "")
	repl.Set("geoip2.representedcountry_is_in_european_union", "")
	repl.Set("geoip2.representedcountry_iso_code", "")
	repl.Set("geoip2.representedcountry_names", "")
	repl.Set("geoip2.representedcountry_locales", "")
	repl.Set("geoip2.representedcountry_confidence", "")
	repl.Set("geoip2.representedcountry_type", "")
	repl.Set("geoip2.representedcountry_name", "")
	repl.Set("geoip2.representedcountry_names_0", "")
	repl.Set("geoip2.representedcountry_names_1", "")
	repl.Set("geoip2.subdivisions", "")
	repl.Set("geoip2.traits_is_anonymous_proxy", "")
	repl.Set("geoip2.traits_is_anonymous_vpn", "")
	repl.Set("geoip2.traits_is_satellite_provider", "")
	repl.Set("geoip2.traits_autonomous_system_number", "")
	repl.Set("geoip2.traits_autonomous_system_organization", "")
	repl.Set("geoip2.traits_connection_type", "")
	repl.Set("geoip2.traits_domain", "")
	repl.Set("geoip2.traits_is_hosting_provider", "")
	repl.Set("geoip2.traits_is_legitimate_proxy", "")
	repl.Set("geoip2.traits_is_public_proxy", "")
	repl.Set("geoip2.traits_is_residential_proxy", "")
	repl.Set("geoip2.traits_is_tor_exit_node", "")
	repl.Set("geoip2.traits_isp", "")
	repl.Set("geoip2.traits_mobile_country_code", "")
	repl.Set("geoip2.traits_mobile_network_code", "")
	repl.Set("geoip2.traits_network", "")
	repl.Set("geoip2.traits_organization", "")
	repl.Set("geoip2.traits_user_type", "")
	repl.Set("geoip2.traits_userCount", "")
	repl.Set("geoip2.traits_static_ip_score", "")

	repl.Set("geoip2.subdivisions_1_confidence", "")
	repl.Set("geoip2.subdivisions_1_geoname_id", "")
	repl.Set("geoip2.subdivisions_1_iso_code", "")
	repl.Set("geoip2.subdivisions_1_locales", "")
	repl.Set("geoip2.subdivisions_1_locales_en", "")
	repl.Set("geoip2.subdivisions_1_names", "")
	repl.Set("geoip2.subdivisions_1_names_0", "")
	repl.Set("geoip2.subdivisions_1_names_1", "")
	repl.Set("geoip2.subdivisions_1_name", "")

	repl.Set("geoip2.subdivisions_2_confidence", "")
	repl.Set("geoip2.subdivisions_2_geoname_id", "")
	repl.Set("geoip2.subdivisions_2_iso_code", "")
	repl.Set("geoip2.subdivisions_2_locales", "")
	repl.Set("geoip2.subdivisions_2_locales_en", "")
	repl.Set("geoip2.subdivisions_2_names", "")
	repl.Set("geoip2.subdivisions_2_names_0", "")
	repl.Set("geoip2.subdivisions_2_names_1", "")
	repl.Set("geoip2.subdivisions_2_name", "")

	m.bindRequest(r, repl)

	return next.ServeHTTP(w, r)
}

// for http handler
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var m Handler
	err := m.UnmarshalCaddyfile(h.Dispenser)
	return &m, err

}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
func (m *Handler) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	return nil
}

func (m *Handler) Provision(ctx caddy.Context) error {
	caddy.Log().Named("http.handlers.geoip2").Info(fmt.Sprintf("Provision"))
	app, err := ctx.App(ModuleName)
	if err != nil {
		return fmt.Errorf("getting geoip2 app: %v", err)
	}
	m.state = app.(*GeoIp2)
	m.ctx = ctx
	return nil
}
func (m *Handler) Validate() error {
	caddy.Log().Named("http.handlers.geoip2").Info(fmt.Sprintf("Validate"))
	return nil
}

// Interface guards
var (
	_ caddy.Module                = (*Handler)(nil)
	_ caddy.Provisioner           = (*Handler)(nil)
	_ caddy.Validator             = (*Handler)(nil)
	_ caddyhttp.MiddlewareHandler = (*Handler)(nil)
	_ caddyfile.Unmarshaler       = (*Handler)(nil)
)

package geoip2

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
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

func (m *Handler) lookupCountry(ip netip.Addr, repl *caddy.Replacer) {
	for _, db := range m.state.databases {
		rec, err := db.Country(ip)
		if err != nil {
			continue
		}

		if rec.HasData() {
			repl.Set("geoip2.country_code", rec.Country.ISOCode)
			repl.Set("geoip2.country_name", rec.Country.Names.English)
			repl.Set("geoip2.country_eu", rec.Country.IsInEuropeanUnion)

			repl.Set("geoip2.continent_code", rec.Continent.Code)
			repl.Set("geoip2.content_name", rec.Continent.Names.English)
		}

		break
	}
}

func (m *Handler) lookupCity(ip netip.Addr, repl *caddy.Replacer) {
	for _, db := range m.state.databases {
		rec, err := db.City(ip)
		if err != nil {
			continue
		}

		if rec.HasData() {
			repl.Set("geoip2.city_name", rec.City.Names.English)
			repl.Set("geoip2.postal_code", rec.Postal.Code)

			if rec.Location.HasData() {
				repl.Set("geoip2.location_latitude", rec.Location.Latitude)
				repl.Set("geoip2.location_longitude", rec.Location.Longitude)
				repl.Set("geoip2.location_timezone", rec.Location.TimeZone)
				repl.Set("geoip2.location_accuracy_radius", rec.Location.AccuracyRadius)
			}
		}

		break
	}
}

func (m *Handler) lookupASN(ip netip.Addr, repl *caddy.Replacer) {
	for _, db := range m.state.databases {
		rec, err := db.ASN(ip)
		if err != nil {
			continue
		}

		if rec.HasData() {
			repl.Set("geoip2.asn_network", rec.Network.String())
			repl.Set("geoip2.asn_organisation", rec.AutonomousSystemOrganization)
			repl.Set("geoip2.asn_system_number", rec.AutonomousSystemNumber)
		}

		break
	}
}

func (m *Handler) bind(r *http.Request, repl *caddy.Replacer) {
	clientIP, _ := m.ClientIP(r)

	if clientIP.IsUnspecified() {
		caddy.Log().Named(ModuleName).Error("No client IP could be resolved from the request")
		return
	}

	m.lookupCity(clientIP, repl)
	m.lookupCountry(clientIP, repl)
	m.lookupASN(clientIP, repl)
}

func (m *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	m.bind(r, r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer))
	return next.ServeHTTP(w, r)
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var m Handler
	err := m.UnmarshalCaddyfile(h.Dispenser)
	return &m, err
}

func (m *Handler) UnmarshalCaddyfile(_ *caddyfile.Dispenser) error {
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

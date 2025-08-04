package geoip2

import (
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/maxmind/geoipupdate/v4/pkg/geoipupdate"
	"strconv"
	"time"
)

const ModuleName = "geoip2"

type GeoIp2 struct {
	databases []*Database

	// Your MaxMind account ID. This was formerly known as UserId.
	AccountID string `json:"account_id,omitempty"`
	// The directory to store the database files. Defaults to DATADIR
	DatabaseDirectory string `json:"database_directory,omitempty"`
	// Your case-sensitive MaxMind license key.
	LicenseKey string `json:"license_key,omitempty"`
	// Enter the edition IDs of the databases you would like to update.
	// Should be GeoLite2-Ciy, GeoLite2-ASN
	EditionID []string `json:"edition_id,omitempty"`
	//update url to use. Defaults to https://updates.maxmind.com
	UpdateUrl string `json:"update_url,omitempty"`
	// The Frequency in seconds to run update. Default to 0, only update On Start
	UpdateFrequency int `json:"update_frequency,omitempty"`
}

func init() {
	caddy.RegisterModule(new(GeoIp2))
	httpcaddyfile.RegisterGlobalOption(ModuleName, parseGeoip2)
}

func (*GeoIp2) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "geoip2",
		New: func() caddy.Module { return new(GeoIp2) },
	}
}

func parseGeoip2(d *caddyfile.Dispenser, _ any) (any, error) {
	state := GeoIp2{}
	err := state.UnmarshalCaddyfile(d)
	return httpcaddyfile.App{
		Name:  "geoip2",
		Value: caddyconfig.JSON(&state, nil),
	}, err
}

func (g *GeoIp2) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		var value string
		key := d.Val()
		if !d.Args(&value) {
			continue
		}
		switch key {
		case "account_id":
			g.AccountID = value
			break
		case "database_directory":
			g.DatabaseDirectory = value
			break
		case "license_key":
			g.LicenseKey = value
			break
		case "edition_id":
			g.EditionID = append(g.EditionID, value)
			break
		case "update_url":
			g.UpdateUrl = value
			break
		case "update_frequency":
			UpdateFrequency, err := strconv.Atoi(value)
			if err == nil {
				g.UpdateFrequency = UpdateFrequency
			}
			break
		}
	}
	caddy.Log().Named("geoip2").Info(fmt.Sprintf("setup Config %v", g))

	return nil
}

func (g *GeoIp2) Start() error {
	return nil
}

func (g *GeoIp2) Stop() error {
	return nil
}

func (g *GeoIp2) Provision(_ caddy.Context) error {
	caddy.Log().Named("geoip2").Info(fmt.Sprintf("Provision"))

	var repl = caddy.NewReplacer()

	if g.UpdateUrl == "" {
		g.UpdateUrl = "https://updates.maxmind.com"
	}
	if g.UpdateFrequency == 0 {
		g.UpdateFrequency = 604800 // 7 days
	}
	if g.DatabaseDirectory == "" {
		g.DatabaseDirectory = "/tmp/"
	}
	if len(g.EditionID) == 0 {
		g.EditionID = []string{"GeoLite2-City", "GeoLite2-ASN"}
	}

	// Initialize updater config if both account ID and license key is set
	var config *geoipupdate.Config
	if g.AccountID != "" && g.LicenseKey != "" {
		accountId, err := strconv.Atoi(repl.ReplaceKnown(g.AccountID, ""))
		if err != nil {
			return fmt.Errorf("failed to parse account id: %w", err)
		}

		config = &geoipupdate.Config{
			AccountID:  accountId,
			LicenseKey: repl.ReplaceKnown(g.LicenseKey, ""),
			EditionIDs: g.EditionID,
			URL:        g.UpdateUrl,
		}
	}

	for _, edition := range g.EditionID {
		db, err := NewDatabase(config, edition, g.DatabaseDirectory, time.Second*time.Duration(g.UpdateFrequency))
		if err != nil {
			return fmt.Errorf("failed to initialize database for GeoIP edition %s: %w", edition, err)
		}

		g.databases = append(g.databases, db)
	}

	return nil
}

func (g *GeoIp2) Destruct() error {
	for _, db := range g.databases {
		_ = db.Close()
	}

	return nil
}

var (
	_ caddyfile.Unmarshaler = (*GeoIp2)(nil)
	_ caddy.Module          = (*GeoIp2)(nil)
	_ caddy.Provisioner     = (*GeoIp2)(nil)
	_ caddy.Destructor      = (*GeoIp2)(nil)
	_ caddy.App             = (*GeoIp2)(nil)
)

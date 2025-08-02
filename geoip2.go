package geoip2

import (
	"context"
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/maxmind/geoipupdate/v4/pkg/geoipupdate"
	"github.com/maxmind/geoipupdate/v4/pkg/geoipupdate/database"
	"github.com/oschwald/maxminddb-golang"
	"net"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

const ModuleName = "geoip2"

type GeoIp2 struct {
	mx       sync.RWMutex
	db       *maxminddb.Reader
	config   geoipupdate.Config
	filePath string

	ctx    context.Context
	cancel context.CancelFunc
	exit   chan error

	// Your MaxMind account ID. This was formerly known as UserId.
	AccountID string `json:"account_id,omitempty"`
	// The directory to store the database files. Defaults to DATADIR
	DatabaseDirectory string `json:"database_directory,omitempty"`
	// Your case-sensitive MaxMind license key.
	LicenseKey string `json:"license_key,omitempty"`
	//Enter the edition IDs of the databases you would like to update.
	//Should be  GeoLite2-City
	EditionID string `json:"edition_id,omitempty"`
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

func (g *GeoIp2) Lookup(ip net.IP) (*GeoIP2Record, error) {
	g.mx.RLock()
	defer g.mx.RUnlock()

	var res GeoIP2Record
	err := g.db.Lookup(ip, &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (g *GeoIp2) Start() error {
	// Do first update (blocking)
	err := g.update()
	if err != nil {
		return err
	}

	// If update frequency, start a new goroutine until cancelled
	if g.UpdateFrequency > 0 {
		go func() {
			defer close(g.exit)

			var interval = time.Duration(g.UpdateFrequency) * time.Second
			caddy.Log().Named(ModuleName).Debug(fmt.Sprintf("updating geoip update frequency every %s", interval))

			var ticker = time.NewTicker(interval)
			defer ticker.Stop()

			for {
				select {
				case <-g.ctx.Done():
					return
				case <-ticker.C:
					err = g.update()
					if err != nil {
						g.exit <- err
					}
				}
			}

		}()
	} else {
		// No routine to start
		close(g.exit)
	}

	return nil
}

func (g *GeoIp2) Stop() error {
	// Stop any running routines
	g.cancel()
	return <-g.exit
}

func (g *GeoIp2) Provision(ctx caddy.Context) error {
	caddy.Log().Named("geoip2").Info(fmt.Sprintf("Provision"))

	var repl = ctx.Value(caddy.ReplacerCtxKey).(*caddy.Replacer)

	if g.UpdateUrl == "" {
		g.UpdateUrl = "https://updates.maxmind.com"
	}

	if g.DatabaseDirectory == "" {
		g.DatabaseDirectory = "/tmp/"
	}
	if g.EditionID == "" {
		g.EditionID = "GeoLite2-City"
	}

	g.filePath = filepath.Join(g.DatabaseDirectory, g.EditionID+".mmdb")
	g.ctx, g.cancel = context.WithCancel(context.Background())
	g.exit = make(chan error, 1)

	accountId, err := strconv.Atoi(repl.ReplaceKnown(g.AccountID, ""))
	if err != nil {
		return fmt.Errorf("failed to parse account id: %w", err)
	}

	g.config = geoipupdate.Config{
		AccountID:         accountId,
		DatabaseDirectory: g.DatabaseDirectory,
		LicenseKey:        repl.ReplaceKnown(g.LicenseKey, ""),
		LockFile:          g.filePath + ".lock",
		EditionIDs:        []string{g.EditionID},
		URL:               g.UpdateUrl,
	}

	return nil
}

func (g *GeoIp2) Destruct() error {
	g.mx.Lock()
	defer g.mx.Unlock()

	if g.db != nil {
		_ = g.db.Close()
		g.db = nil
	}

	return nil
}

func (g *GeoIp2) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	g.mx.Lock()
	defer g.mx.Unlock()

	for d.Next() {
		var value string
		key := d.Val()
		if !d.Args(&value) {
			continue
		}
		switch key {
		case "accountId":
			g.AccountID = value
			break
		case "databaseDirectory":
			g.DatabaseDirectory = value
			break
		case "licenseKey":
			g.LicenseKey = value
			break
		case "editionID":
			g.EditionID = value
			break
		case "updateUrl":
			g.UpdateUrl = value
			break
		case "updateFrequency":
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

func (g *GeoIp2) update() error {
	g.mx.Lock()
	defer g.mx.Unlock()

	var (
		log = caddy.Log().Named("geoip2")

		client = geoipupdate.NewClient(&g.config)
		reader = database.NewHTTPDatabaseReader(client, &g.config)
	)

	// If we can update then do so now
	if g.config.AccountID > 0 && g.config.LicenseKey != "" {
		log.Info("Updating GeoIP database")

		w, err := database.NewLocalFileDatabaseWriter(g.filePath, g.config.LockFile, g.config.Verbose)
		if err != nil {
			return err
		}

		err = reader.Get(w, g.EditionID)
		if err != nil {
			return fmt.Errorf("updating database at %s: %w", g.filePath, err)
		}

		// Success, close the old database reference (if held)
		if g.db != nil {
			_ = g.db.Close()
			g.db = nil
		}

		// Commit the writer
		err = w.Commit()
		if err != nil {
			return fmt.Errorf("commiting updates to database at %s: %w", g.filePath, err)
		}
	}

	// Already open don't need to open again
	if g.db != nil {
		return nil
	}

	log.Debug(fmt.Sprintf("Opening GeoIP database at %s", g.filePath))
	var err error
	g.db, err = maxminddb.Open(g.filePath)
	if err != nil {
		return err
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

package geoip2

import (
	"context"
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/maxmind/geoipupdate/v4/pkg/geoipupdate"
	"github.com/maxmind/geoipupdate/v4/pkg/geoipupdate/database"
	"github.com/oschwald/geoip2-golang/v2"
	"go.uber.org/zap"
	"net/netip"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func update(config *geoipupdate.Config, edition, filePath string) error {
	var (
		client = geoipupdate.NewClient(config)
		reader = database.NewHTTPDatabaseReader(client, config)
	)

	w, err := database.NewLocalFileDatabaseWriter(filePath, filePath+".lock", config.Verbose)
	if err != nil {
		return err
	}

	err = reader.Get(w, edition)
	if err != nil {
		return fmt.Errorf("updating database at %s: %w", filePath, err)
	}

	return nil
}

// Database is a synchronous self-updating GeoIP2 database
type Database struct {
	mx sync.RWMutex
	db *geoip2.Reader

	log    *zap.Logger
	cancel context.CancelFunc
	err    chan error
}

func NewDatabase(config *geoipupdate.Config, edition string, dataDir string, updateEvery time.Duration) (*Database, error) {
	var ctx, cancel = context.WithCancel(context.Background())
	var filePath = filepath.Join(dataDir, edition+".mmdb")

	var db = &Database{
		log:    caddy.Log().Named(ModuleName).With(zap.String("edition", edition)),
		cancel: cancel,
		err:    make(chan error, 1),
	}

	// Check if the database exists
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) && config != nil {
		// No existing database but there is an update config, try loading it
		err = update(config, edition, filePath)
		if err != nil {
			err = fmt.Errorf("no existing database at %s and self update failed: %w", filePath, err)
		}
	}

	if err != nil {
		return nil, err
	}

	db.db, err = geoip2.Open(filePath)
	if err != nil {
		return nil, err
	}

	// If there is an update config and self update is enabled on updateEvery
	if config != nil && updateEvery > 0 {
		go db.startAutomaticUpdates(ctx, config, edition, filePath, updateEvery)
	} else {
		close(db.err)
	}

	return db, nil
}

func (db *Database) selfUpdater(config *geoipupdate.Config, edition, filePath string) func() error {
	return func() error {
		db.mx.Lock()
		defer db.mx.Unlock()

		err := update(config, edition, filePath)
		if err != nil {
			return err
		}

		r, err := geoip2.Open(filePath)
		if err != nil {
			return err
		}

		_ = db.db.Close()
		db.db = r

		return nil
	}
}

func (db *Database) startAutomaticUpdates(ctx context.Context, config *geoipupdate.Config, edition, filePath string, updateEvery time.Duration) {
	var ticker = time.NewTicker(updateEvery)
	defer ticker.Stop()

	db.log.Debug(fmt.Sprintf("Next update in %s", updateEvery))

	defer close(db.err)
	var updater = db.selfUpdater(config, edition, filePath)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			db.log.Debug("Updating database")
			err := updater()
			if err != nil {
				// Only log errors from updating (best effort)
				db.log.Warn("failed to update db", zap.Error(err))
			}
		}
	}
}

func (db *Database) Close() error {
	db.cancel()
	err := <-db.err

	db.mx.Lock()
	defer db.mx.Unlock()

	_ = db.db.Close()

	return err
}

func (db *Database) ASN(ip netip.Addr) (*geoip2.ASN, error) {
	db.mx.RLock()
	defer db.mx.RUnlock()

	return db.db.ASN(ip)
}

func (db *Database) City(ip netip.Addr) (*geoip2.City, error) {
	db.mx.RLock()
	defer db.mx.RUnlock()

	return db.db.City(ip)
}

func (db *Database) Country(ip netip.Addr) (*geoip2.Country, error) {
	db.mx.RLock()
	defer db.mx.RUnlock()

	return db.db.Country(ip)
}

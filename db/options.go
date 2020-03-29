package db

import (
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger/options"
	ds "github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger"
	core "github.com/textileio/go-threads/core/db"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/jsonpatcher"
)

const (
	defaultDatastorePath = "eventstore"
)

// Option takes a Config and modifies it.
type Option func(*Config) error

// Config has configuration parameters for a db.
type Config struct {
	RepoPath    string
	Datastore   ds.TxnDatastore
	EventCodec  core.EventCodec
	Debug       bool
	LowMem      bool
	Collections []CollectionConfig
	Credentials thread.Credentials
}

func newDefaultEventCodec() core.EventCodec {
	return jsonpatcher.New()
}

func newDefaultDatastore(repoPath string, lowMem bool) (ds.TxnDatastore, error) {
	path := filepath.Join(repoPath, defaultDatastorePath)
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return nil, err
	}
	opts := badger.DefaultOptions
	if lowMem {
		opts.TableLoadingMode = options.FileIO
	}
	return badger.NewDatastore(path, &opts)
}

// WithLowMem specifies whether or not to use low memory settings.
func WithLowMem(low bool) Option {
	return func(sc *Config) error {
		sc.LowMem = low
		return nil
	}
}

// WithRepoPath sets the repo path.
func WithRepoPath(path string) Option {
	return func(sc *Config) error {
		sc.RepoPath = path
		return nil
	}
}

// WithDebug indicate to output debug information.
func WithDebug(enable bool) Option {
	return func(sc *Config) error {
		sc.Debug = enable
		return nil
	}
}

// WithEventCodec configure to use ec as the EventCodec
// for transforming actions in events, and viceversa.
func WithEventCodec(ec core.EventCodec) Option {
	return func(sc *Config) error {
		sc.EventCodec = ec
		return nil
	}
}

// WithCredentials specifies credentials for cerating a new db.
func WithCredentials(cr thread.Credentials) Option {
	return func(sc *Config) error {
		sc.Credentials = cr
		return nil
	}
}

// WithCollections is used to specify collections that
// will be created.
func WithCollections(cs ...CollectionConfig) Option {
	return func(sc *Config) error {
		sc.Collections = cs
		return nil
	}
}

// TxnOptions defines options for a transaction.
type TxnOptions struct {
	Credentials thread.Credentials
}

// TxnOption specifies a transaction option.
type TxnOption func(*TxnOptions)

// WithTxnCredentials specifies credentials for the transaction.
func WithTxnCredentials(cr thread.Credentials) TxnOption {
	return func(args *TxnOptions) {
		args.Credentials = cr
	}
}

// ManagerOptions defines options for a db manager.
type ManagerOptions struct {
	Collections []CollectionConfig
	Credentials thread.Credentials
}

// ManagerOption specifies a manager option.
type ManagerOption func(*ManagerOptions)

// WithManagerCollections is used to specify collections that
// will be created in a managed db.
func WithManagerCollections(cs ...CollectionConfig) ManagerOption {
	return func(args *ManagerOptions) {
		args.Collections = cs
	}
}

// WithManagerCredentials specifies credentials for interacting with a managed db.
func WithManagerCredentials(cr thread.Credentials) ManagerOption {
	return func(args *ManagerOptions) {
		args.Credentials = cr
	}
}

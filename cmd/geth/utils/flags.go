package utils

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/triedb/hashdb"
	"github.com/ethereum/go-ethereum/trie/triedb/pathdb"
	"github.com/urfave/cli/v2"
)

var (
	Backend = &cli.StringFlag{
		Name:    "backend",
		Aliases: []string{"b"},
		Value:   "https://bsc-dataseed.binance.org/",
		Usage:   "Backend for get new blocks",
	}
	DbHost = &cli.StringFlag{
		Name:  "db-host",
		Value: "127.0.0.1",
		Usage: "Database host",
	}

	DbPort = &cli.StringFlag{
		Name:  "db-port",
		Value: "4321",
		Usage: "Database port",
	}

	StateSchemeFlag = &cli.StringFlag{
		Name:     "state.scheme",
		Usage:    "Scheme to use for storing ethereum state ('hash' or 'path')",
		Value:    rawdb.HashScheme,
		Category: flags.StateCategory,
	}

	SvcHost = &cli.StringFlag{
		Name:  "svc-host",
		Value: "0.0.0.0",
		Usage: "Service host",
	}

	SvcPort = &cli.IntFlag{
		Name:  "svc-port",
		Value: 8080,
		Usage: "Service port",
	}

	Engine = &cli.StringFlag{
		Name:  "engine",
		Value: "pebble",
		Usage: "Engine for leveldb/pebble",
	}
	SnapPath = &cli.StringFlag{
		Name:  "snap",
		Value: "/tmp/snap",
		Usage: "Snapshot path",
	}
)

// ParseStateScheme resolves scheme identifier from CLI flag. If the provided
// state scheme is not compatible with the one of persistent scheme, an error
// will be returned.
//
//   - none: use the scheme consistent with persistent state, or fallback
//     to hash-based scheme if state is empty.
//   - hash: use hash-based scheme or error out if not compatible with
//     persistent state scheme.
//   - path: use path-based scheme or error out if not compatible with
//     persistent state scheme.
func ParseStateScheme(ctx *cli.Context, disk ethdb.Database) (string, error) {
	// If state scheme is not specified, use the scheme consistent
	// with persistent state, or fallback to hash mode if database
	// is empty.
	stored := rawdb.ReadStateScheme(disk)
	if !ctx.IsSet(StateSchemeFlag.Name) {
		if stored == "" {
			// use default scheme for empty database, flip it when
			// path mode is chosen as default
			log.Info("State schema set to default", "scheme", "hash")
			return rawdb.HashScheme, nil
		}
		log.Info("State scheme set to already existing", "scheme", stored)
		return stored, nil // reuse scheme of persistent scheme
	}
	// If state scheme is specified, ensure it's compatible with
	// persistent state.
	scheme := ctx.String(StateSchemeFlag.Name)
	if stored == "" || scheme == stored {
		log.Info("State scheme set by user", "scheme", scheme)
		return scheme, nil
	}
	return "", fmt.Errorf("incompatible state scheme, stored: %s, provided: %s", stored, scheme)
}

// MakeTrieDatabase constructs a trie database based on the configured scheme.
func MakeTrieDatabase(ctx *cli.Context, disk ethdb.Database, preimage bool, readOnly bool) *trie.Database {
	config := &trie.Config{
		Preimages: preimage,
	}
	scheme, err := ParseStateScheme(ctx, disk)
	if err != nil {
		Fatalf("%v", err)
	}
	if scheme == rawdb.HashScheme {
		// Read-only mode is not implemented in hash mode,
		// ignore the parameter silently. TODO(rjl493456442)
		// please config it if read mode is implemented.
		config.HashDB = hashdb.Defaults
		return trie.NewDatabase(disk, config)
	}
	if readOnly {
		config.PathDB = pathdb.ReadOnly
	} else {
		config.PathDB = pathdb.Defaults
	}
	return trie.NewDatabase(disk, config)
}

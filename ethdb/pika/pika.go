package pika

import (
	"context"
	"unsafe"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/go-redis/redis/v8"
)

var _ ethdb.Database = (*Database)(nil)

func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func s2b(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}

type Database struct {
	ctx context.Context
	db  *redis.Client
}

func New(addr string) (ethdb.Database, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		DB:           0, // use default DB
		PoolSize:     20,
		MinIdleConns: 20,
	})
	return &Database{
		ctx: context.Background(),
		db:  rdb,
	}, nil
}

// Has retrieves if a key is present in the key-value data store.
func (d *Database) Has(key []byte) (bool, error) {
	if d.db.Exists(d.ctx, b2s(key)).Val() == 0 {
		return false, nil
	} else {
		return true, nil
	}
}

// Get retrieves the given key if it's present in the key-value data store.
func (d *Database) Get(key []byte) ([]byte, error) {
	return d.db.Get(d.ctx, b2s(key)).Bytes()
}

// HasAncient returns an indicator whether the specified data exists in the
// ancient store.
func (d *Database) HasAncient(kind string, number uint64) (bool, error) {
	return false, nil
}

// Ancient retrieves an ancient binary blob from the append-only immutable files.
func (d *Database) Ancient(kind string, number uint64) ([]byte, error) {
	return nil, nil
}

// AncientRange retrieves multiple items in sequence, starting from the index 'start'.
// It will return
//   - at most 'count' items,
//   - if maxBytes is specified: at least 1 item (even if exceeding the maxByteSize),
//     but will otherwise return as many items as fit into maxByteSize.
//   - if maxBytes is not specified, 'count' items will be returned if they are present
func (d *Database) AncientRange(kind string, start uint64, count uint64, maxBytes uint64) ([][]byte, error) {
	return nil, nil
}

// Ancients returns the ancient item numbers in the ancient store.
func (d *Database) Ancients() (uint64, error) {
	return 0, nil
}

// Tail returns the number of first stored item in the freezer.
// This number can also be interpreted as the total deleted item numbers.
func (d *Database) Tail() (uint64, error) {
	return 0, nil
}

// AncientSize returns the ancient size of the specified category.
func (d *Database) AncientSize(kind string) (uint64, error) {
	return 0, nil
}

// ItemAmountInAncient returns the actual length of current ancientDB.
func (d *Database) ItemAmountInAncient() (uint64, error) {
	return 0, nil
}

// AncientOffSet returns the offset of current ancientDB.
func (d *Database) AncientOffSet() uint64 {
	return 0
}

// ReadAncients runs the given read operation while ensuring that no writes take place
// on the underlying freezer.
func (d *Database) ReadAncients(fn func(ethdb.AncientReaderOp) error) (err error) {
	return nil
}

// Put inserts the given value into the key-value data store.
func (d *Database) Put(key []byte, value []byte) error {
	return d.db.Set(d.ctx, b2s(key), value, 0).Err()
}

// Delete removes the key from the key-value data store.
func (d *Database) Delete(key []byte) error {
	return d.db.Del(d.ctx, b2s(key)).Err()
}

// ModifyAncients runs a write operation on the ancient store.
// If the function returns an error, any changes to the underlying store are reverted.
// The integer return value is the total size of the written data.
func (d *Database) ModifyAncients(_ func(ethdb.AncientWriteOp) error) (int64, error) {
	return 0, nil
}

// TruncateHead discards all but the first n ancient data from the ancient store.
// After the truncation, the latest item can be accessed it item_n-1(start from 0).
func (d *Database) TruncateHead(n uint64) (uint64, error) {
	return 0, nil
}

// TruncateTail discards the first n ancient data from the ancient store. The already
// deleted items are ignored. After the truncation, the earliest item can be accessed
// is item_n(start from 0). The deleted items may not be removed from the ancient store
// immediately, but only when the accumulated deleted data reach the threshold then
// will be removed all together.
func (d *Database) TruncateTail(n uint64) (uint64, error) {
	return 0, nil
}

// Sync flushes all in-memory ancient store data to disk.
func (d *Database) Sync() error {
	return nil
}

// MigrateTable processes and migrates entries of a given table to a new format.
// The second argument is a function that takes a raw entry and returns it
// in the newest format.
func (d *Database) MigrateTable(_ string, _ func([]byte) ([]byte, error)) error {
	return nil
}

func (d *Database) DiffStore() ethdb.KeyValueStore {
	return nil
}

func (d *Database) SetDiffStore(diff ethdb.KeyValueStore) {
	panic("not implemented") // TODO: Implement
}

// NewBatch creates a write-only database that buffers changes to its host db
// until a final write is called.
func (d *Database) NewBatch() ethdb.Batch {
	return &batch{
		db:     d,
		writes: make([]keyvalue, 0),
		size:   0,
	}
}

// NewBatchWithSize creates a write-only database batch with pre-allocated buffer.
func (d *Database) NewBatchWithSize(size int) ethdb.Batch {
	return &batch{
		db:     d,
		writes: make([]keyvalue, 0),
		size:   0,
	}
}

// NewIterator creates a binary-alphabetical iterator over a subset
// of database content with a particular key prefix, starting at a particular
// initial key (or after, if it does not exist).
//
// Note: This method assumes that the prefix is NOT part of the start, so there's
// no need for the caller to prepend the prefix to the start
func (d *Database) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	return NewIterator(d, prefix, start)
}

// Stat returns a particular internal stat of the database.
func (d *Database) Stat(property string) (string, error) {
	panic("not implemented") // TODO: Implement
}

// AncientDatadir returns the path of root ancient directory. Empty string
// will be returned if ancient store is not enabled at all. The returned
// path can be used to construct the path of other freezers.
func (d *Database) AncientDatadir() (string, error) {
	panic("not implemented") // TODO: Implement
}

// Compact flattens the underlying data store for the given key range. In essence,
// deleted and overwritten versions are discarded, and the data is rearranged to
// reduce the cost of operations needed to access them.
//
// A nil start is treated as a key before all keys in the data store; a nil limit
// is treated as a key after all keys in the data store. If both is nil then it
// will compact entire data store.
func (d *Database) Compact(start []byte, limit []byte) error {
	return nil
}

// NewSnapshot creates a database snapshot based on the current state.
// The created snapshot will not be affected by all following mutations
// happened on the database.
// Note don't forget to release the snapshot once it's used up, otherwise
// the stale data will never be cleaned up by the underlying compactor.
func (d *Database) NewSnapshot() (ethdb.Snapshot, error) {
	return d, nil
}

func (d *Database) Close() error {
	log.Info("close database")
	return d.db.Close()
}

func (d *Database) Release() {}

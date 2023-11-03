package chainkv

import (
	"github.com/Ankr-Shanghai/chainkv/client"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

type Database struct {
	client client.Client
	log    log.Logger
}

func NewChainKV(host, port string, size int) (*Database, error) {
	opt := &client.Option{
		Host: host,
		Port: port,
		Size: size,
	}
	client, err := client.NewClient(opt)
	if err != nil {
		return nil, err
	}
	log := log.New("chainkv", "chainkv")
	return &Database{
		client: client,
		log:    log,
	}, nil
}

// Close stops the metrics collection, flushes any pending data to disk and closes
// all io accesses to the underlying key-value store.
func (d *Database) Close() error {
	return d.client.Close()
}

// Has retrieves if a key is present in the key-value store.
func (d *Database) Has(key []byte) (bool, error) {
	return d.client.Has(key)
}

// Get retrieves the given key if it's present in the key-value store.
func (d *Database) Get(key []byte) ([]byte, error) {
	return d.client.Get(key)
}

// Put inserts the given value into the key-value store.
func (d *Database) Put(key []byte, value []byte) error {
	return d.client.Put(key, value)
}

// Delete removes the key from the key-value store.
func (d *Database) Delete(key []byte) error {
	return nil
}

// NewBatch creates a write-only key-value store that buffers changes to its host
// database until a final write is called.
func (d *Database) NewBatch() ethdb.Batch {
	b, err := d.client.NewBatch()
	if err != nil {
		d.log.Error("NewBatch error", "err", err)
		return nil
	}
	return &batch{
		batch: b,
	}
}

// NewBatchWithSize creates a write-only database batch with pre-allocated buffer.
// It's not supported by pebble, but pebble has better memory allocation strategy
// which turns out a lot faster than leveldb. It's performant enough to construct
// batch object without any pre-allocated space.
func (d *Database) NewBatchWithSize(_ int) ethdb.Batch {
	b, err := d.client.NewBatch()
	if err != nil {
		d.log.Error("NewBatch error", "err", err)
		return nil
	}
	return &batch{
		batch: b,
	}
}

// snapshot wraps a pebble snapshot for implementing the Snapshot interface.
type snapshot struct {
	snap *client.Snap
}

// NewSnapshot creates a database snapshot based on the current state.
// The created snapshot will not be affected by all following mutations
// happened on the database.
// Note don't forget to release the snapshot once it's used up, otherwise
// the stale data will never be cleaned up by the underlying compactor.
func (d *Database) NewSnapshot() (ethdb.Snapshot, error) {
	snap, err := d.client.NewSnap()
	if err != nil {
		return nil, err
	}

	return &snapshot{
		snap: snap,
	}, nil
}

// Has retrieves if a key is present in the snapshot backing by a key-value
// data store.
func (snap *snapshot) Has(key []byte) (bool, error) {
	return snap.snap.Has(key)
}

// Get retrieves the given key if it's present in the snapshot backing by
// key-value data store.
func (snap *snapshot) Get(key []byte) ([]byte, error) {
	return snap.snap.Get(key)
}

// Release releases associated resources. Release should always succeed and can
// be called multiple times without causing error.
func (snap *snapshot) Release() {
	snap.snap.Release()
}

// Stat returns a particular internal stat of the database.
func (d *Database) Stat(property string) (string, error) {
	return "", nil
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

// Path returns the path to the database directory.
func (d *Database) Path() string {
	return ""
}

// batch is a write-only batch that commits changes to its host database
// when Write is called. A batch cannot be used concurrently.
type batch struct {
	batch *client.Batch
}

// Put inserts the given value into the batch for later committing.
func (b *batch) Put(key, value []byte) error {
	return b.batch.Put(key, value)
}

// Delete inserts the a key removal into the batch for later committing.
func (b *batch) Delete(key []byte) error {
	return b.batch.Delete(key)
}

// ValueSize retrieves the amount of data queued up for writing.
func (b *batch) ValueSize() int {
	return b.batch.ValueSize()
}

// Write flushes any accumulated data to disk.
func (b *batch) Write() error {
	return b.batch.Write()
}

// Reset resets the batch for reuse.
func (b *batch) Reset() {
	b.batch.Reset()
}

// Replay replays the batch contents.
func (b *batch) Replay(w ethdb.KeyValueWriter) error {
	for _, kv := range b.batch.Writes {
		if kv.Delete {
			err := w.Delete(kv.Key)
			if err != nil {
				return err
			}
			continue
		}
		if err := w.Put(kv.Key, kv.Value); err != nil {
			return err
		}
	}
	return nil
}

// pebbleIterator is a wrapper of underlying iterator in storage engine.
// The purpose of this structure is to implement the missing APIs.
type pebbleIterator struct {
	iter *client.Iterator
}

// NewIterator creates a binary-alphabetical iterator over a subset
// of database content with a particular key prefix, starting at a particular
// initial key (or after, if it does not exist).
func (d *Database) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	iterator, err := d.client.NewIter(prefix, start)
	if err != nil {
		d.log.Error("NewIterator error", "err", err)
		return nil
	}
	return &pebbleIterator{
		iter: iterator,
	}
}

// Next moves the iterator to the next key/value pair. It returns whether the
// iterator is exhausted.
func (iter *pebbleIterator) Next() bool {
	return iter.iter.Next()
}

// Error returns any accumulated error. Exhausting all the key/value pairs
// is not considered to be an error.
func (iter *pebbleIterator) Error() error {
	return iter.iter.Error()
}

// Key returns the key of the current key/value pair, or nil if done. The caller
// should not modify the contents of the returned slice, and its contents may
// change on the next call to Next.
func (iter *pebbleIterator) Key() []byte {
	return iter.iter.Key()
}

// Value returns the value of the current key/value pair, or nil if done. The
// caller should not modify the contents of the returned slice, and its contents
// may change on the next call to Next.
func (iter *pebbleIterator) Value() []byte {
	return iter.iter.Value()
}

// Release releases associated resources. Release should always succeed and can
// be called multiple times without causing error.
func (iter *pebbleIterator) Release() {
	iter.iter.Close()
}

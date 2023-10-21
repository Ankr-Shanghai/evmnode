package pika

import (
	"context"
	"fmt"
	"os"
)

type iterator struct {
	db    *Database
	index int
	keys  []string
}

func NewIterator(d *Database, prefix, start []byte) *iterator {
	st := fmt.Sprintf("%s%s*", prefix, start)

	keys := make([]string, 0)
	var (
		err    error
		cursor uint64
		ks     []string
		ctx    = context.Background()
	)

	for {
		ks, cursor, err = d.db.Scan(ctx, cursor, st, 32).Result()
		if err != nil {
			println(err)
			os.Exit(-1)
		}
		keys = append(keys, ks...)
		if 0 == cursor {
			break
		}
	}

	return &iterator{
		db:    d,
		index: -1,
		keys:  keys,
	}
}

// Next moves the iterator to the next key/value pair. It returns whether the
// iterator is exhausted.
func (it *iterator) Next() bool {
	// Short circuit if iterator is already exhausted in the forward direction.
	if it.index >= len(it.keys) {
		return false
	}
	it.index += 1
	return it.index < len(it.keys)
}

// Error returns any accumulated error. Exhausting all the key/value pairs
// is not considered to be an error. A memory iterator cannot encounter errors.
func (it *iterator) Error() error {
	return nil
}

// Key returns the key of the current key/value pair, or nil if done. The caller
// should not modify the contents of the returned slice, and its contents may
// change on the next call to Next.
func (it *iterator) Key() []byte {
	// Short circuit if iterator is not in a valid position
	if it.index < 0 || it.index >= len(it.keys) {
		return nil
	}
	return []byte(it.keys[it.index])
}

// Value returns the value of the current key/value pair, or nil if done. The
// caller should not modify the contents of the returned slice, and its contents
// may change on the next call to Next.
func (it *iterator) Value() []byte {
	// Short circuit if iterator is not in a valid position
	if it.index < 0 || it.index >= len(it.keys) {
		return nil
	}
	val, _ := it.db.Get(s2b(it.keys[it.index]))
	return val
}

// Release releases associated resources. Release should always succeed and can
// be called multiple times without causing error.
func (it *iterator) Release() {
	it.index, it.keys = -1, nil
}

package rawdb

import (
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/chainkv"
	"github.com/ethereum/go-ethereum/ethdb/pika"
)

func NewPikaDatabase(addr string) (ethdb.Database, error) {
	return pika.New(addr)
}

func NewChainKVDatabase(host, port string, size int) (ethdb.Database, error) {
	kvdb, err := chainkv.NewChainKV(host, port, size)
	if err != nil {
		return nil, err
	}
	return NewDatabase(kvdb), nil
}

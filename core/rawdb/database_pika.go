package rawdb

import (
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/pika"
)

func NewPikaDatabase(addr string) (ethdb.Database, error) {
	return pika.New(addr)
}

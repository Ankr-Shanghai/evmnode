package pika

import "errors"

var (
	errDBClosed = errors.New("database closed")

	errDbNotFound = errors.New("not found")

	errSnapshotReleased = errors.New("snapshot released")
)

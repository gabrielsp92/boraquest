// Package idgen generates entity identifiers.
package idgen

import "github.com/google/uuid"

// UUIDGenerator creates random (v4) UUIDs.
type UUIDGenerator struct{}

// NewID returns a new UUID string.
func (UUIDGenerator) NewID() string {
	return uuid.NewString()
}

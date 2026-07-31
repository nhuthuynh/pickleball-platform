// Package idgen provides a shared port.IDGenerator implementation for every
// bounded context, so contexts don't each pull in their own UUID dependency.
package idgen

import "github.com/google/uuid"

// UUID implements every context's port.IDGenerator using random (v4) UUIDs.
type UUID struct{}

func (UUID) NewID() string {
	return uuid.NewString()
}

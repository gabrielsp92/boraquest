// Package health holds the domain model for service health reporting.
package health

import (
	"errors"
	"time"
)

// Status is the health state of the service.
type Status string

const (
	StatusUp   Status = "up"
	StatusDown Status = "down"
)

// ErrInvalidStatus is returned when a Status is not one of the known values.
var ErrInvalidStatus = errors.New("health: invalid status")

// Validate reports whether s is a known status.
func (s Status) Validate() error {
	switch s {
	case StatusUp, StatusDown:
		return nil
	default:
		return ErrInvalidStatus
	}
}

// Report is a point-in-time snapshot of the service health.
type Report struct {
	Status    Status
	Version   string
	CheckedAt time.Time
}

// NewReport builds a validated Report.
func NewReport(status Status, version string, checkedAt time.Time) (Report, error) {
	if err := status.Validate(); err != nil {
		return Report{}, err
	}
	return Report{Status: status, Version: version, CheckedAt: checkedAt}, nil
}

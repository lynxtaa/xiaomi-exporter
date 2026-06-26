// Package devicescanner defines the port for scanning sensor devices and
// reporting their decoded readings.
package devicescanner

import (
	"context"
	"time"
)

// Reading holds metrics decoded from a single advertisement packet.
// A nil field means that object was not present in the packet.
type Reading struct {
	Temperature *float64
	Humidity    *float64
}

// Device identifies the sensor a reading came from. It carries identity only;
// adapter secrets (e.g. decryption keys) stay inside the adapter.
type Device struct {
	Name string
	Mac  string
}

// ReadingFunc receives every decoded packet. The Reading carries whichever
// objects that packet held (often only temperature or only humidity).
type ReadingFunc func(d Device, r Reading)

// Scanner contains data for scanning devices via Bluetooth
type Scanner interface {
	Run(ctx context.Context, interval, window time.Duration, onReading ReadingFunc) error
}

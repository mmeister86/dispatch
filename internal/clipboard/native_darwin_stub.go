//go:build !darwin || !cgo

package clipboard

import "context"

func readNativeDarwinPNG(context.Context) ([]byte, error) {
	return nil, ErrUnavailable
}

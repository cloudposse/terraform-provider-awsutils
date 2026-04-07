package jsonutil

import (
	"encoding/json"
	"errors"
	"io"
)

var (
	errReaderNil = errors.New("'io.Reader' being decoded is nil")
	errOutNil    = errors.New("output parameter 'out' is nil")
)

// Decodes/Unmarshals the given io.Reader pointing to a JSON, into a desired object.
func DecodeJSONFromReader(r io.Reader, out interface{}) error {
	if r == nil {
		return errReaderNil
	}
	if out == nil {
		return errOutNil
	}

	dec := json.NewDecoder(r)

	// While decoding JSON values, interpret the integer values as `json.Number`s instead of `float64`.
	dec.UseNumber()

	// Since 'out' is an interface representing a pointer, pass it to the decoder without an '&'
	return dec.Decode(out)
}

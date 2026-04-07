package nullable

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	errIntAtLeast = errors.New("value below minimum")
	errIntBetween = errors.New("value out of range")
)

const (
	TypeNullableInt = schema.TypeString
)

type Int string

func (i Int) IsNull() bool {
	return i == ""
}

func (i Int) Value() (int64, bool, error) {
	if i.IsNull() {
		return 0, true, nil
	}

	value, err := strconv.ParseInt(string(i), 10, 64)
	if err != nil {
		return 0, false, err
	}
	return value, false, nil
}

// ValidateTypeStringNullableInt provides custom error messaging for TypeString ints
// Some arguments require an int value or unspecified, empty field.
func ValidateTypeStringNullableInt(v interface{}, k string) (ws []string, es []error) {
	value, ok := v.(string)
	if !ok {
		es = append(es, fmt.Errorf("%w: %s", errExpectedString, k))
		return ws, es
	}

	if value == "" {
		return ws, es
	}

	if _, err := strconv.ParseInt(value, 10, 64); err != nil {
		es = append(es, fmt.Errorf("%s: cannot parse '%s' as int: %w", k, value, err))
	}

	return ws, es
}

// ValidateTypeStringNullableIntAtLeast provides custom error messaging for TypeString ints
// Some arguments require an int value or unspecified, empty field.
func ValidateTypeStringNullableIntAtLeast(min int) schema.SchemaValidateFunc { //nolint:staticcheck // SA1019: experimental helper preserves legacy validate signature
	return func(i interface{}, k string) (ws []string, es []error) {
		value, ok := i.(string)
		if !ok {
			es = append(es, fmt.Errorf("%w: %s", errExpectedString, k))
			return ws, es
		}

		if value == "" {
			return ws, es
		}

		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			es = append(es, fmt.Errorf("%s: cannot parse '%s' as int: %w", k, value, err))
			return ws, es
		}

		if v < int64(min) {
			es = append(es, fmt.Errorf("%w: %s expected at least (%d), got %d", errIntAtLeast, k, min, v))
		}

		return ws, es
	}
}

// ValidateTypeStringNullableIntBetween provides custom error messaging for TypeString ints
// Some arguments require an int value or unspecified, empty field.
func ValidateTypeStringNullableIntBetween(min int, max int) schema.SchemaValidateFunc { //nolint:staticcheck // SA1019: experimental helper preserves legacy validate signature
	return func(i interface{}, k string) (ws []string, es []error) {
		value, ok := i.(string)
		if !ok {
			es = append(es, fmt.Errorf("%w: %s", errExpectedString, k))
			return ws, es
		}

		if value == "" {
			return ws, es
		}

		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			es = append(es, fmt.Errorf("%s: cannot parse '%s' as int: %w", k, value, err))
			return ws, es
		}

		if v < int64(min) || v > int64(max) {
			es = append(es, fmt.Errorf("%w: %s expected between (%d) and (%d), got %d", errIntBetween, k, min, max, v))
		}

		return ws, es
	}
}

package plutusencoder

import (
	"fmt"
	"reflect"
	"strconv"
)

// isOptionalField reports whether a struct field is marked optional via the
// `plutusOptional:"true"` tag. Optional fields may be absent from an input
// map and are left at their zero value. Tag values are parsed with
// strconv.ParseBool; an invalid value is an error rather than being silently
// treated as required.
func isOptionalField(field reflect.StructField) (bool, error) {
	tagVal := field.Tag.Get("plutusOptional")
	if tagVal == "" {
		return false, nil
	}
	optional, err := strconv.ParseBool(tagVal)
	if err != nil {
		return false, fmt.Errorf("invalid plutusOptional tag %q on field %s: %w", tagVal, field.Name, err)
	}
	return optional, nil
}

package analyzer

import (
	"go/types"
	"math/big"
	"strconv"
	"strings"
	"time"
)

func greaterValue(target kind, left, right string) bool {
	left, right = strings.TrimSpace(left), strings.TrimSpace(right)
	if target == kDuration {
		a, aErr := time.ParseDuration(left)
		b, bErr := time.ParseDuration(right)
		return aErr == nil && bErr == nil && a > b
	}
	a, aOK := new(big.Float).SetString(left)
	b, bOK := new(big.Float).SetString(right)
	return aOK && bOK && a.Cmp(b) > 0
}

func checkNumber(typ types.Type, text string) error {
	typ = types.Unalias(typ)
	if named, ok := typ.(*types.Named); ok {
		typ = named.Underlying()
	}
	basic, ok := typ.Underlying().(*types.Basic)
	if !ok {
		return errText("not a number")
	}
	bits := 64
	switch basic.Kind() {
	case types.Int8, types.Uint8:
		bits = 8
	case types.Int16, types.Uint16:
		bits = 16
	case types.Int32, types.Uint32, types.Float32:
		bits = 32
	}
	var err error
	if basic.Info()&types.IsUnsigned != 0 {
		_, err = strconv.ParseUint(text, 10, bits)
	} else if basic.Info()&types.IsInteger != 0 {
		_, err = strconv.ParseInt(text, 10, bits)
	} else if strings.ContainsAny(text, "xXpP_") {
		err = errText("not a decimal float")
	} else {
		var value float64
		value, err = strconv.ParseFloat(text, bits)
		if err == nil && (value != value || value > 1.7976931348623157e308 || value < -1.7976931348623157e308) {
			err = errText("non-finite number")
		}
	}
	if err != nil {
		return errText("invalid or overflowing number " + strconv.Quote(text))
	}
	return nil
}

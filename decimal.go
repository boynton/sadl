// Decimal is a big.Float equivalent that marshals to/from JSON.
package sadl

import (
	"fmt"
	"math/big"
)

//we establish this "reasonable" max precision
const DecimalPrecision = uint(250)

type Decimal struct {
	big.Float
}

// Encode as a JSON number. The JSON spec allows for arbitrary precision, so this is the correct thing to do.
func (d Decimal) MarshalJSON() ([]byte, error) {
	repr := d.Text('f', -1)
	return []byte(repr), nil
}

func (d *Decimal) UnmarshalJSON(b []byte) error {
	stringRepr := string(b)
	num, err := ParseDecimal(stringRepr)
	if err == nil {
		*d = *num
		return nil
	}
	return fmt.Errorf("Bad Decimal number: %s", string(b))
}

func ParseDecimal(text string) (*Decimal, error) {
	num, _, err := big.ParseFloat(text, 10, DecimalPrecision, big.ToNearestEven)
	if err != nil {
		return nil, fmt.Errorf("Bad Decimal number: %s", text)
	}
	return &Decimal{Float: *num}, nil
}

func NewDecimal(val float64) *Decimal {
	return &Decimal{Float: *big.NewFloat(val)}
}

func (d *Decimal) String() string {
	return d.Text('f', -1)
}

func (d *Decimal) AsInt32() int32 {
	n := d.AsInt64()
	return int32(n)
}

func (d *Decimal) AsInt64() int64 {
	i, _ := d.Int64()
	return i
}

func (d *Decimal) AsFloat64() float64 {
	f, _ := d.Float64()
	return f
}

func (d *Decimal) AsBigFloat() *big.Float {
	return &d.Float
}

func DecimalValue(val *Decimal, defval interface{}) *Decimal {
	if val != nil {
		return val
	}
	if defval != nil {
		switch n := defval.(type) {
		case *Decimal:
			return n
		case int64:
			d, _ := ParseDecimal(fmt.Sprint(n))
			return d
		case int32:
			return NewDecimal(float64(n))
		case int16:
			return NewDecimal(float64(n))
		case int8:
			return NewDecimal(float64(n))
		case float32:
			return NewDecimal(float64(n))
		case float64:
			return NewDecimal(n)
		}
	}
	return nil
}

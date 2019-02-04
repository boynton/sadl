package sadl

import (
	"encoding/json"
	"fmt"
	"math/big"
)

const DecimalPrecision = uint(250)

type Decimal struct {
	bf *big.Float
}

// Encode as a string. Encoding as a JSON number works fine, but the Unbmarshal doesn't. If we use string as the representation in JSON, it works fine.
// What a shame.
func (d *Decimal) MarshalJSON() ([]byte, error) {
	repr := d.bf.Text('f', -1)
	stringRepr := "\"" + repr + "\""
	return []byte(stringRepr), nil
}

func (d *Decimal) UnmarshalJSON(b []byte) error {
	var stringRepr string
	err := json.Unmarshal(b, &stringRepr)
	if err == nil {
		var num *Decimal
		num, err = ParseDecimal(stringRepr)
		if err == nil {
			d.bf = num.bf
			return nil
		}
	} else {
		var floatRepr float64
		err = json.Unmarshal(b, &floatRepr)
		if err == nil {
			num := NewDecimal(floatRepr)
			d.bf = num.bf
			return nil
		}
	}
	return fmt.Errorf("Bad Decimal number: %s", string(b))
}

func ParseDecimal(text string) (*Decimal, error) {
	num, _, err := big.ParseFloat(text, 10, DecimalPrecision, big.ToNearestEven)
	if err != nil {
		return nil, fmt.Errorf("Bad Decimal number: %s", text)
	}
	return &Decimal{bf: num}, nil
}

func NewDecimal(val float64) *Decimal {
	return &Decimal{bf: big.NewFloat(val)}
}

func (d *Decimal) String() string {
	if d == nil {
		return ""
	}
	return fmt.Sprint(d.bf)
}

func (d *Decimal) Int32() int32 {
	n := d.Int64()
	return int32(n)
}

func (d *Decimal) Int64() int64 {
	i, _ := d.bf.Int64()
	return i
}

func (d *Decimal) Float64() float64 {
	f, _ := d.bf.Float64()
	return f
}

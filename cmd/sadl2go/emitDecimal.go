package main

func (gen *GoGenerator) emitDecimalType() {
	if gen.err != nil {
		return
	}
	gen.addImport("encoding/json")
	gen.addImport("fmt")
	gen.addImport("math/big")
	gen.emit(decimalType)
}

const decimalType = `// Decimal is a big.Float equivalent, but marshals to JSON as strings to preserve precision.

const DecimalPrecision = uint(250)

type Decimal struct {
	big.Float
}

// Encode as a string. Encoding as a JSON number works fine, but the Unbmarshal doesn't. If we use string as the representation in JSON, it works fine.
// What a shame.
func (d *Decimal) MarshalJSON() ([]byte, error) {
	repr := d.Text('f', -1)
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
			*d = *num
			return nil
		}
	} else {
		var floatRepr float64
		err = json.Unmarshal(b, &floatRepr)
		if err == nil {
         *d = *NewDecimal(floatRepr)
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
   return &Decimal{Float:*num}, nil
}

func NewDecimal(val float64) *Decimal {
   return &Decimal{Float:*big.NewFloat(val)}
}

func (d *Decimal) String() string {
	return fmt.Sprint(d)
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

`

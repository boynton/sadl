package sadl

import(
	"encoding/json"
	"math/big"
)

const DecimalPrecision = uint(250)

type decimal struct {
	bf *big.Float
}

// Encode as a string. Encoding as a JSON number works fine, but the Unbmarshal doesn't. If we use string as the representation in JSON, it works fine.
// What a shame.
func (bd *decimal) MarshalJSON() ([]byte, error) {
	repr := bd.bf.Text('f',-1)
	stringRepr := "\"" + repr + "\""
	return []byte(stringRepr), nil
}

func (bd *decimal) UnmarshalJSON(b []byte) error {
	var stringRepr string
	err := json.Unmarshal(b, &stringRepr)
   if err == nil {
		num, err := parseDecimal(stringRepr)
		if err == nil {
			bd.bf = num.bf
		}
	}
	return err
}

func parseDecimal(text string) (*decimal, error) {
	num, _, err := big.ParseFloat(text, 10, DecimalPrecision, big.ToNearestEven)
	if err != nil {
		return nil, err
	}
	return &decimal{bf: num}, nil
}

func decimalToInt64(d *decimal) int64 {
	i, _ := d.bf.Int64()
	return i
}

func decimalToFloat64(d *decimal) float64 {
	f, _ := d.bf.Float64()
	return f
}


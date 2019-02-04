package sadl

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Quantity struct {
	Value *Decimal
	Unit  string
}

func (q *Quantity) String() string {
	return fmt.Sprintf("%v %s", q.Value, q.Unit)
}

func ParseQuantity(repr string) (*Quantity, error) {
	n := strings.Index(repr, " ")
	if n > 0 {
		value, err := ParseDecimal(repr[:n])
		if err == nil {
			unit := repr[n+1:]
			if len(unit) > 0 {
				return &Quantity{
					Value: value,
					Unit:  unit,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("Not a valid Quantity: %q", repr)
}

func NewQuantity(value float64, unit string) *Quantity {
	return &Quantity{
		Value: NewDecimal(value),
		Unit:  unit,
	}
}

func (q *Quantity) MarshalJSON() ([]byte, error) {
	return []byte("\"" + q.String() + "\""), nil
}

func (q *Quantity) UnmarshalJSON(b []byte) error {
	var repr string
	err := json.Unmarshal(b, &repr)
	if err == nil {
		var q2 *Quantity
		q2, err = ParseQuantity(repr)
		if err == nil {
			*q = *q2
			return nil
		}
	}
	return fmt.Errorf("Not a valid Quantity (%v)", err)
}

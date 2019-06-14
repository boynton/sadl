package sadl

import (
	"encoding/json"
	"fmt"
	"strings"
)

type UnitValue struct {
	Value *Decimal
	Unit  string
}

func (q *UnitValue) String() string {
	return fmt.Sprintf("%v %s", q.Value, q.Unit)
}

func ParseUnitValue(repr string) (*UnitValue, error) {
	n := strings.Index(repr, " ")
	if n > 0 {
		value, err := ParseDecimal(repr[:n])
		if err == nil {
			unit := repr[n+1:]
			if len(unit) > 0 {
				return &UnitValue{
					Value: value,
					Unit:  unit,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("Not a valid UnitValue: %q", repr)
}

func NewUnitValue(value float64, unit string) *UnitValue {
	return &UnitValue{
		Value: NewDecimal(value),
		Unit:  unit,
	}
}

func (q *UnitValue) MarshalJSON() ([]byte, error) {
	return []byte("\"" + q.String() + "\""), nil
}

func (q *UnitValue) UnmarshalJSON(b []byte) error {
	var repr string
	err := json.Unmarshal(b, &repr)
	if err == nil {
		var q2 *UnitValue
		q2, err = ParseUnitValue(repr)
		if err == nil {
			*q = *q2
			return nil
		}
	}
	return fmt.Errorf("Not a valid UnitValue (%v)", err)
}

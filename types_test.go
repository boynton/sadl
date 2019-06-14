package sadl

import (
	"encoding/json"
	"fmt"
	"testing"
)

func encode(obj interface{}) (string, error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func decode(jsonData string, expected interface{}) error {
	return json.Unmarshal([]byte(jsonData), &expected)
}

func TestTimestamp(test *testing.T) {
	jsonData := `"2019-02-03T22:48:19.043Z"`
	var ts Timestamp
	err := decode(jsonData, &ts)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(Pretty(ts))
	}
}

func TestBadTimestamp(test *testing.T) {
	jsonData := `"2019-02-03T22:48:19.Zz"`
	var ts Timestamp
	err := decode(jsonData, &ts)
	if err == nil {
		test.Errorf("Bad timestamp should have caused an error: %q", jsonData)
	}
}

func TestUUID(test *testing.T) {
	jsonData := `"1ce437b0-1dd2-11b2-bc26-003ee1be85f9"`
	var u1 UUID
	err := decode(jsonData, &u1)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(Pretty(u1))
	}
}

func TestBadUUID(test *testing.T) {
	jsonData := `{}`
	var u1 UUID
	err := decode(jsonData, &u1)
	if err == nil {
		test.Errorf("Bad UUID should have caused an error: %q", jsonData)
	}
}

func TestDecimal(test *testing.T) {
	jsonData := `"123.456"`
	var d *Decimal
	err := decode(jsonData, &d)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(Pretty(d))
	}
}

func TestGood2Decimal(test *testing.T) {
	jsonData := `123`
	var d *Decimal
	err := decode(jsonData, &d)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(Pretty(d))
	}
}

func TestBadDecimal(test *testing.T) {
	jsonData := `"123foo"`
	var d *Decimal
	err := decode(jsonData, &d)
	if err == nil {
		test.Errorf("Bad Decimal should have caused an error: %q", jsonData)
	}
}

func TestLargeDecimal(test *testing.T) {
	jsonData := `3.141592653589793238462643383279502884197169399375105819`
	pi, err := ParseDecimal(jsonData)
	if err != nil {
		test.Errorf("%v", err)
		return
	}
	fmt.Println("pi:", pi)

	encoded, err := encode(pi)
	if err != nil {
		test.Errorf("%v", err)
		return
	}
	if jsonData != encoded {
		test.Errorf("Decimal did not encode accurately, should be %s, but encoded to %s", jsonData, encoded)
	}
	fmt.Printf("Decimal encoding succeeded: Pi correctly encoded to %s\n", encoded)

	var d *Decimal
	err = decode(jsonData, &d)
	if err != nil {
		test.Errorf("%v", err)
		return
	}
	if pi.Cmp(d.AsBigFloat()) != 0 {
		fmt.Printf("Decimal decoding loss of precision: decoded %s to %s\n", jsonData, d.String())
		fmt.Println("[This is a known problem with golang's JSON decoder]")
	}
}

func TestUnitValue(test *testing.T) {
	val := 100.0
	unit := "USD"
	q1 := NewUnitValue(val, unit)
	jsonData := Pretty(q1)
	var q *UnitValue
	err := decode(jsonData, &q)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		if q == nil {
			test.Errorf("UnitValue JSON round trip resulting in no object")
		} else if q.Value == nil || q.Value.AsFloat64() != val {
			test.Errorf("UnitValue JSON round trip resulting in UnitValue.value: %v", q.Value)
		} else if q.Unit != unit {
			test.Errorf("UnitValue JSON round trip resulting in UnitValue.unit: %v", q.Unit)
		} else {
			fmt.Println("Valid UnitValue:", Pretty(q))
		}
	}
}

func TestBadUnitValue(test *testing.T) {
	jsonData := `100`
	var q *UnitValue
	err := decode(jsonData, &q)
	if err == nil {
		test.Errorf("Bad UnitValue should have caused an error: %q", jsonData)
	} else {
		fmt.Println(err)
	}

}

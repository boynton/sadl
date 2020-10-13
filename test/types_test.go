package test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/boynton/sadl"
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
	var ts sadl.Timestamp
	err := decode(jsonData, &ts)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(sadl.Pretty(ts))
	}
}

func TestBadTimestamp(test *testing.T) {
	jsonData := `"2019-02-03T22:48:19.Zz"`
	var ts sadl.Timestamp
	err := decode(jsonData, &ts)
	if err == nil {
		test.Errorf("Bad timestamp should have caused an error: %q", jsonData)
	}
}

func TestUUID(test *testing.T) {
	jsonData := `"1ce437b0-1dd2-11b2-bc26-003ee1be85f9"`
	var u1 sadl.UUID
	err := decode(jsonData, &u1)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(sadl.Pretty(u1))
	}
}

func TestBadUUID(test *testing.T) {
	jsonData := `{}`
	var u1 sadl.UUID
	err := decode(jsonData, &u1)
	if err == nil {
		test.Errorf("Bad UUID should have caused an error: %q", jsonData)
	}
}

func TestGood2Decimal(test *testing.T) {
	jsonData := `123`
	var d *sadl.Decimal
	err := decode(jsonData, &d)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(sadl.Pretty(d))
	}
}

func TestBadDecimal(test *testing.T) {
	jsonData := `"123foo"`
	var d *sadl.Decimal
	err := decode(jsonData, &d)
	if err == nil {
		test.Errorf("Bad Decimal should have caused an error: %q", jsonData)
	}
}

func TestLargeDecimal(test *testing.T) {
	jsonData := `3.141592653589793238462643383279502884197169399375105819`
	pi, err := sadl.ParseDecimal(jsonData)
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

	var d *sadl.Decimal
	err = decode(jsonData, &d)
	if err != nil {
		test.Errorf("%v", err)
		return
	}
	if pi.Cmp(d.AsBigFloat()) != 0 {
		fmt.Printf("Decimal decoding loss of precision: decoded %s to %s\n", jsonData, d.String())
	} else {
		fmt.Printf("Decimal decoding succeeded: Pi correctly decoded to %s\n", d.String())
	}
}

func TestUnitValue(test *testing.T) {
	val := 100.0
	unit := "USD"
	q1 := sadl.NewUnitValue(val, unit)
	jsonData := sadl.Pretty(q1)
	var q *sadl.UnitValue
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
			fmt.Println("Valid UnitValue:", sadl.Pretty(q))
		}
	}
}

func TestBadUnitValue(test *testing.T) {
	jsonData := `100`
	var q *sadl.UnitValue
	err := decode(jsonData, &q)
	if err == nil {
		test.Errorf("Bad UnitValue should have caused an error: %q", jsonData)
	}

}

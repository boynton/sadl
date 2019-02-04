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

func pretty(obj interface{}) string {
	j, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		return fmt.Sprintf("<JSON ERROR: %v>", err)
	}
	return string(j)
}

func TestTimestamp(test *testing.T) {
	jsonData := `"2019-02-03T22:48:19.043Z"`
	var ts Timestamp
	err := decode(jsonData, &ts)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(pretty(ts))
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
		fmt.Println(pretty(u1))
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
		fmt.Println(pretty(d))
	}
}

func TestGood2Decimal(test *testing.T) {
	jsonData := `123`
	var d *Decimal
	err := decode(jsonData, &d)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(pretty(d))
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

func TestQuantity(test *testing.T) {
	val := 100.0
	unit := "USD"
	q1 := NewQuantity(val, unit)
	jsonData := pretty(q1)
	var q *Quantity
	err := decode(jsonData, &q)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		if q == nil {
			test.Errorf("Quantity JSON round trip resulting in no object")
		} else if q.Value == nil || q.Value.Float64() != val {
			test.Errorf("Quantity JSON round trip resulting in quantity value: %v", q.Value)
		} else if q.Unit != unit {
			test.Errorf("Quantity JSON round trip resulting in quantity unit: %v", q.Unit)
		} else {
			fmt.Println("Valid Quantity:", pretty(q))
		}
	}
}

func TestBadQuantity(test *testing.T) {
	jsonData := `100`
	var q *Quantity
	err := decode(jsonData, &q)
	if err == nil {
		test.Errorf("Bad Quantity should have caused an error: %q", jsonData)
	} else {
		fmt.Println(err)
	}

}

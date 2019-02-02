package sadl

import (
//	"encoding/json"
	"fmt"
//	"io/ioutil"
//	"strings"
	"testing"
)

func xTestQuantity(test *testing.T) {
	Verbose = true
	v, err := ParseString(`type Money Quantity<Decimal,String>`)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(Pretty(v))
	}
}

func xTestArray(test *testing.T) {
	Verbose = true
	v, err := ParseString(`type Foo Array<String> (maxsize=2)`)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(Pretty(v))
	}
}

func xTestUnion(test *testing.T) {
	Verbose = true
	v, err := ParseString(`type Foo Union<Int32,String>`)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(Pretty(v))
	}
}

func TestStruct(test *testing.T) {
	Verbose = true
	v, err := ParseString(`
type Foo Struct {
   String s (pattern="y*")
   Decimal d (min=0, max=100)
   Array<Int32> nums (maxsize=100)
}
`)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(Pretty(v))
	}
}

func xTestBaseTypes(test *testing.T) {
	v, err := ParseString(
	`
type Int Int32
type Long Int64 (min=0, max=20)
type Float Float32
type Double Float64
type BigDecimal Decimal
type ByteArray Bytes (maxsize=4,minsize=2,x_one,x_two="Hey")
type NonEmptyString String (minsize=1)
type DateTime Timestamp
type Currency String(pattern="^[A-Z][A-Z][A-Z]$")
type Money Quantity<Decimal,Currency>
type ID UUID
type List Array
type StringList Array<String>
type StringMap Map<String,String>
type Set Map<String,Bool>
type JSONObject Struct
type Point Struct {
    Float64 x (required)
    Float64 y (required)
}
type WeightUnits Enum {
   in
   ft
   yd
   mi
   cm
   m
   km
}
type Distance Quantity<Decimal,WeightUnits>
/*type Number Union<Int8,Int16,Int32,Int64,Float32,Float64,Decimal>*/
`)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(Pretty(v))
	}
}

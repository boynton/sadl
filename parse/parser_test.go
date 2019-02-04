package parse

import (
	"fmt"
	"testing"
	//	"github.com/boynton/sadl"
)

func TestComments(test *testing.T) {
	//	Verbose = true
	v, err := parseString(`//one
//two
name foo

//three
type Foo String

//four
type Bar Int32
`)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		if len(v.Types) != 2 {
			test.Errorf("Did not parse correctly, expected 2 types in the schema")
			return
		}
		if v.Comment == "one two" {
			if v.Types[0].Comment == "three" && v.Types[1].Comment == "four" {
				// Looks ok
				fmt.Println(Pretty(v))
				return
			}
		}
		test.Errorf("Comments did not get attached to correct elements in schema")
	}
}

func TestQuantity(test *testing.T) {
	v, err := parseString(`type Money Quantity<Decimal,Stringx>`)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(Pretty(v))
	}
}

func TestArray(test *testing.T) {
	v, err := parseString(`type Foo Array<String> (maxsize=2)`)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(Pretty(v))
	}
}

func xTestUnion(test *testing.T) {
	v, err := parseString(`type Foo Union<Int32,String>`)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(Pretty(v))
	}
}

func TestStruct(test *testing.T) {
	v, err := parseString(`
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
	v, err := parseString(
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

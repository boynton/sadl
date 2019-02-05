package parse

import (
	"fmt"
	"testing"
	//	"github.com/boynton/sadl"
)

func testParse(test *testing.T, expectSuccess bool, src string) {
	v, err := parseString(src)
	if expectSuccess {
		if err != nil {
			test.Errorf("%v", err)
		}
	} else {
		if err == nil {
			test.Errorf("Expected failure, but it parsed: %v", v)
		}
	}
}

func xTestFieldStringConstraints(test *testing.T) {
}

func TestFieldDefaultValidates(test *testing.T) {
	testParse(test, true, `
type Test Struct {
   foo Timestamp (default="2019-02-05T01:24:30.998Z")
}
`)
	testParse(test, false, `
type Test Struct {
   foo Timestamp (default=23)
}
`)
	testParse(test, true, `
type Test Struct {
   foo String (default="one")
}
`)
	testParse(test, false, `
type Test Struct {
   foo String (default=["one"])
}
`)
	testParse(test, true, `
type Test Struct {
   foo String (minSize=1, default="one")
}
`)
	testParse(test, false, `
type Test Struct {
   foo String (minSize=5, default="one")
}
`)
	testParse(test, true, `
type Test Struct {
   foo String (maxSize=3, default="one")
}
`)
	testParse(test, false, `
type Test Struct {
   foo String (maxSize=2, default="one")
}
`)
	testParse(test, true, `
type Test Struct {
   foo String (values=["one", "two"], default="one")
}
`)
	testParse(test, false, `
type Test Struct {
   foo String (values=["one", "two"], default="three")
}
`)
	testParse(test, false, `
type Test Struct {
   foo String (pattern="^[a-z]*$", values=["one","two"],default="one")
}
`)
	testParse(test, true, `
type Test Struct {
   foo String (pattern="^[a-z]*$", default="one")
}
`)
	testParse(test, false, `
type Test Struct {
   foo String (pattern="^[a-z]*$", default="three21")
}
`)

}

func okTestFieldDefaultRequired(test *testing.T) {
	v, err := parseString(`
//Test comment.
type Test Struct {
   s String (required, default="blah")
}
`)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(Pretty(v))
	}
}

func xTestNestedStruct(test *testing.T) {
	v, err := parseString(`
//Test comment.
type Test Struct {
    //This field is a nested struct.
    mynestedstruct Struct { //Where does this comment go?
      //something comment
      something String //a field in the nested struct
      oranother Int32 //and another field
    } (x_foo="bar",default={"something": "Hey", "oranother": 23})
} //More Test comment.

`)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(Pretty(v))
	}
}

func xTestComments(test *testing.T) {
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

func xTestQuantity(test *testing.T) {
	v, err := parseString(`type Money Quantity<Decimal,Stringx>`)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(Pretty(v))
	}
}

func xTestArray(test *testing.T) {
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

func xTestStruct(test *testing.T) {
	v, err := parseString(`
type Foo Struct {
   s String (pattern="y*")
   d Decimal (min=0, max=100)
   nums Array<Int32> (maxsize=100)
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
    x Float64 (required)
    y Float64 (required)
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

func xTestFieldTypeNotDefined(test *testing.T) {
	v, err := parseString(`
type Foo Struct {
   b Bar
}
`)
	if err == nil {
		test.Errorf("Undefined field type should have caused an error: %v", Pretty(v))
	} else {
		fmt.Println("[Correctly detected error]", err)
	}
}

func xTestDupType(test *testing.T) {
	v, err := parseString(`
type Foo String
type Foo Struct {
   s String
}
`)
	if err == nil {
		test.Errorf("Duplicate type should have caused an error: %v", Pretty(v))
	} else {
		fmt.Println("[Correctly detected error]", err)
	}
}

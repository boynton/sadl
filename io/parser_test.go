package io

import (
	"fmt"
	"testing"

	"github.com/boynton/sadl/util"
)

func testParse(test *testing.T, expectSuccess bool, src string) {
	v, err := parseString(src, nil)
	if expectSuccess {
		if err != nil {
			test.Errorf("%v", err)
		}
	} else {
		if err == nil {
			test.Errorf("Expected failure, but this:\n%s\nparsed anyway: %v", src, util.Pretty(v))
		}
	}
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

func TestDecimalDefault(test *testing.T) {
	v, err := parseString(`
type Test Struct {
   foo Decimal (default=3.141592653589793238462643383279502884197169399375105819)
}
`, nil)
	if err != nil {
		test.Errorf("Cannot parse valid Decimal default value: %v", err)
	} else {
		fmt.Println(util.Pretty(v))
	}
}

func TestFieldDefaultRequired(test *testing.T) {
	v, err := parseString(`
//Test comment.
type Test Struct {
   s String (required, default="blah")
}
`, nil)
	if err == nil {
		test.Errorf("expected error providing a default value for a required field")
	} else {
		fmt.Println(util.Pretty(v))
	}
}

func TestNestedStruct(test *testing.T) {
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

`, nil)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(util.Pretty(v))
	}
}

func TestComments(test *testing.T) {
	//	Verbose = true
	v, err := parseString(`//one
//two
name foo

//three
type Foo String

//four
type Bar Int32
`, nil)
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
				fmt.Println(util.Pretty(v))
				return
			}
		}
		test.Errorf("Comments did not get attached to correct elements in schema")
	}
}

func TestParseUnitValue(test *testing.T) {
	v, err := parseString(`type Money UnitValue<Decimal,String>`, nil)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(util.Pretty(v))
	}
}

func TestArray(test *testing.T) {
	v, err := parseString(`type Foo Array<String> (maxsize=2)`, nil)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(util.Pretty(v))
	}
}

func TestUnion(test *testing.T) {
	v, err := parseString(`type Foo Union<Int32,String>`, nil)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(util.Pretty(v))
	}
}

func TestStruct(test *testing.T) {
	v, err := parseString(`
type Foo Struct {
   s String (pattern="y*")
   d Decimal (min=0, max=100)
   nums Array<Int32> (maxsize=100)
}
`, nil)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(util.Pretty(v))
	}
}

func TestBaseTypes(test *testing.T) {
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
type Money UnitValue<Decimal,Currency>
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
type Distance UnitValue<Decimal,WeightUnits>
/*type Number Union<Int8,Int16,Int32,Int64,Float32,Float64,Decimal>*/
`, nil)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		fmt.Println(util.Pretty(v))
	}
}

func TestFieldTypeNotDefined(test *testing.T) {
	v, err := parseString(`
type Foo Struct {
   b Bar
}
`, nil)
	if err == nil {
		test.Errorf("Undefined field type should have caused an error: %v", util.Pretty(v))
	} else {
		fmt.Println("[Correctly detected error]", err)
	}
}

func TestDupType(test *testing.T) {
	v, err := parseString(`
type Foo String
type Foo Struct {
   s String
}
`, nil)
	if err == nil {
		test.Errorf("Duplicate type should have caused an error: %v", util.Pretty(v))
	} else {
		fmt.Println("[Correctly detected error]", err)
	}
}

func TestActionCombos(test *testing.T) {
	header := `
type BarRequest Struct {
   name String
}
type BarResponse Struct {
   greeting String
}
type BadRequestError Struct {
   message String
}
type GenericError Struct {
   message String
}
`
	testParse(test, true, header+"action Bar()\n")
	testParse(test, true, header+"action Bar(BarRequest)\n")
	testParse(test, true, header+"action Bar() BarResponse\n")
	testParse(test, true, header+"action Bar(BarRequest) BarResponse\n")
	testParse(test, true, header+"action Bar(BarRequest) BarResponse except BadRequestError\n")
	testParse(test, true, header+"action Bar(BarRequest) BarResponse except BadRequestError, GenericError\n")
	testParse(test, true, header+"action Bar() except BadRequestError\n")
	testParse(test, true, header+"action Bar() except BadRequestError, GenericError\n")
	testParse(test, true, header+"action Bar() BarResponse except BadRequestError, GenericError\n")
	testParse(test, true, header+"action Bar(BarRequest) except BadRequestError\n")
	testParse(test, false, header+"action Bar(BarRequest) except\n")
	testParse(test, false, header+"action Bar\n")
	testParse(test, false, header+"action Bar(BarResponse\n")
	testParse(test, false, header+"action Bar(BarResponse) BarResponse BadRequestError\n")
}

func TestPathTemplateSyntax(test *testing.T) {
	v, err := parseString(`http GET "/one/{two}" { }`, nil)
	if err != nil {
		test.Errorf("Good path template caused an error (%v): %v", err, util.Pretty(v))
	}
	v, err = parseString(`http GET "/one/{two" { }`, nil)
	if err == nil {
		test.Errorf("Bad path template should have caused an error: %v", util.Pretty(v))
	}
	v, err = parseString(`http GET "/one/{two}}" { }`, nil)
	if err == nil {
		test.Errorf("Bad path template should have caused an error: %v", util.Pretty(v))
	}
	v, err = parseString(`http GET "/one{/two}" { }`, nil)
	if err == nil {
		test.Errorf("Bad path template should have caused an error: %v", util.Pretty(v))
	}
	v, err = parseString(`http GET "/one/{{two}" { }`, nil)
	if err == nil {
		test.Errorf("Bad path template should have caused an error: %v", util.Pretty(v))
	}
}

func TestSimpleExpect(test *testing.T) {
	v, err := parseString(`type Foo Struct {
  x String
}
http GET "/foo" {
  expect 200 {
    body Foo
  }
}
`, nil)
	if err != nil {
		test.Errorf("standard expect caused an error (%v): %v", err, util.Pretty(v))
	}
	v, err = parseString(`type Foo Struct {
  x String
}
http GET "/foo" {
  expect 200 Foo
}
`, nil)
	if err != nil {
		test.Errorf("simple expect caused an error (%v): %v", err, util.Pretty(v))
	}
}

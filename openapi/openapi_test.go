package openapi

import (
	"fmt"
	"testing"

	"github.com/boynton/sadl"
	"github.com/ghodss/yaml"
)

var emptyConfig = sadl.NewData()

func TestExport(test *testing.T) {
	src := `
name Test1
version "1.2.3"
type Foo Struct {
   descr String (required)
   count Int32
   b Int8
   w Int16
   l Int64
   f Float32
   dub Float64
   dec Decimal
   by Bytes
   q UnitValue
   ts Timestamp
   a Array<String>
   
}
type Foos Array<Foo>

`
	model, err := sadl.ParseSadlString(src, emptyConfig)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		//		model.Name = "test1"
		gen := NewGenerator(model, emptyConfig)
		oas, err := gen.ExportToOAS3()
		if err != nil {
			test.Errorf("%v", err)
		} else {
			fmt.Println(sadl.Pretty(oas))
		}
	}
}

func TestCrudl(test *testing.T) {
	testFile(test, "../examples/petstore.sadl")
}

func testFile(test *testing.T, path string) {
	model, err := sadl.ParseSadlFile(path, emptyConfig)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		gen := NewGenerator(model, emptyConfig)
		oas, err := gen.ExportToOAS3()
		if err != nil {
			test.Errorf("%v", err)
		} else {
			fmt.Println(ToYAML(oas))
		}
	}
}

func ToYAML(obj interface{}) string {
	j := sadl.Pretty(obj)
	b, err := yaml.JSONToYAML([]byte(j))
	if err != nil {
		return j
	}
	return string(b)
}

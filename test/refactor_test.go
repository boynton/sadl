package test

import (
	"testing"

	"github.com/boynton/sadl/io"
	"github.com/boynton/sadl/util"
)

func TestRefactorStruct(test *testing.T) {
	model, err := io.ParseSadlString(`type Foo Struct {
    values Enum {
       ONE,
       TWO
    }
}
`)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		err = model.ConvertInlineEnums()
		if err != nil {
			test.Errorf("%v", err)
		}
		if len(model.Types) != 2 {
			test.Errorf("Failed to refactor inline Enums: %s", util.Pretty(model))
		}
	}
}

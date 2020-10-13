package test

import (
	"testing"

	"github.com/boynton/sadl"
)

func TestRefactorStruct(test *testing.T) {
	model, err := sadl.ParseSadlString(`type Foo Struct {
    values Enum {
       ONE,
       TWO
    }
}
`, sadl.NewData())
	if err != nil {
		test.Errorf("%v", err)
	} else {
		err = model.ConvertInlineEnums()
		if err != nil {
			test.Errorf("%v", err)
		}
		if len(model.Types) != 2 {
			test.Errorf("Failed to refactor inline Enums: %s", sadl.Pretty(model))
		}
	}
}

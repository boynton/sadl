package sadl

import (
	"testing"
)

func TestRefactorStruct(test *testing.T) {
	model, err := parseString(`type Foo Struct {
    values Enum {
       ONE,
       TWO
    }
}
`, nil)
	if err != nil {
		test.Errorf("%v", err)
	} else {
		err = model.ConvertInlineEnums()
		if err != nil {
			test.Errorf("%v", err)
		}
		if len(model.Types) != 2 {
			test.Errorf("Failed to refactor inline Enums: %s", Pretty(model))
		}
	}
}

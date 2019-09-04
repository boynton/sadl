package sadl

import (
	"fmt"
	"strings"
)

//for every typedef and action parameter that has an inline enum def, create a toplevel enum def and refer to it instead.
//This reduces duplicate definitions.
//This produces an error if name conflicts cannot be resolved.
func (model *Model) ConvertInlineEnums() error {
	//	refs := make(map[string]*TypeSpec, 0)
	var tds []*TypeDef
	for _, td := range model.Types {
		switch td.Type {
		case "Struct":
			for _, fdef := range td.Fields {
				if fdef.Type == "Enum" {
					tname := capitalize(fdef.Name)
					ntd := &TypeDef{
						TypeSpec: fdef.TypeSpec,
						Name:     tname,
					}
					prev := model.FindType(tname)
					if prev != nil {
						if !model.EquivalentTypes(&prev.TypeSpec, &fdef.TypeSpec) {
							//Alternatively, could prefix the struct type name to the new type `td.Name + tname` to make it unique. Steill not foolproof.
							return fmt.Errorf("cannot refactor, duplicate type names for non-equivalent types: %s and %s\n", Pretty(prev), Pretty(fdef))
						}
					}
					var blank TypeSpec
					fdef.TypeSpec = blank
					fdef.Type = tname
					tds = append(tds, ntd)
				}
			}
		}
	}
	for _, td := range tds {
		model.Types = append(model.Types, td)
		model.typeIndex[td.Name] = td
	}
	return nil
}

func capitalize(s string) string {
	return strings.ToUpper(s[0:1]) + s[1:]
}

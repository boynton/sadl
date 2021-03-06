package graphql

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/boynton/sadl"
	gql_ast "github.com/graphql-go/graphql/language/ast"
	gql_parser "github.com/graphql-go/graphql/language/parser"
	gql_source "github.com/graphql-go/graphql/language/source"
)

func IsValidFile(path string) bool {
	if strings.HasSuffix(path, ".graphql") {
		return true
	}
	return false
}

func Import(paths []string, conf *sadl.Data) (*sadl.Model, error) {
	//todo: merge multiple files
	if len(paths) != 1 {
		return nil, fmt.Errorf("GraphQL file merging NYI")
	}
	path := paths[0]
	src, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	doc, err := gql_parser.Parse(gql_parser.ParseParams{
		Source: &gql_source.Source{
			Body: src,
			Name: "GraphQL",
		},
		Options: gql_parser.ParseOptions{
			NoLocation: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Cannot parse GraphQL: %v\n", err)
	}
	schema, err := gqlSchema(doc, conf)
	if err != nil {
		return nil, err
	}
	return sadl.NewModel(schema)
}

func gqlSchema(doc *gql_ast.Document, conf *sadl.Data) (*sadl.Schema, error) {
	name := conf.GetString("name")
	if name == "" {
		name = "generatedFromGraphQL"
	}
	namespace := conf.GetString("namespace")
	schema := &sadl.Schema{
		Name:      name,
		Namespace: namespace,
	}
	ignore := make(map[string]bool, 0)
	var err error
	td := &sadl.TypeDef{
		Name:    "ID",
		Comment: "The GraphQL 'ID' type",
		TypeSpec: sadl.TypeSpec{
			Type: "String",
		},
	}
	schema.Types = append(schema.Types, td)
	for _, def := range doc.Definitions {
		switch tdef := def.(type) {
		case *gql_ast.ObjectDefinition:
			if _, ok := ignore[tdef.Name.Value]; !ok {
				err = gqlStruct(schema, tdef)
			}
		case *gql_ast.SchemaDefinition:
			for _, opt := range tdef.OperationTypes {
				iname := (*gql_ast.Named)(opt.Type).Name.Value
				ignore[iname] = true
			}
		case *gql_ast.EnumDefinition:
			err = gqlEnum(schema, tdef)
		case *gql_ast.UnionDefinition:
			err = gqlUnion(schema, tdef)
		case *gql_ast.InterfaceDefinition:
			//ignore for now fmt.Println("fix me: interfaces")
		case *gql_ast.InputObjectDefinition:
			//ignore for now fmt.Println("fix me: input objects")
		case *gql_ast.ScalarDefinition:
			sname := tdef.Name.Value
			switch sname {
			case "Timestamp":
			case "UUID":
				//Allow the name through, a native SADL type
			default:
				err = fmt.Errorf("Unsupported custom scalar: %s\n", sadl.Pretty(def))
			}
		default:
			err = fmt.Errorf("Unsupported definition: %v\n", def.GetKind())
		}
		if err != nil {
			return nil, err
		}
	}
	return schema, nil
}

func typeName(t gql_ast.Type) string {
	switch tt := t.(type) {
	case *gql_ast.Named:
		return tt.Name.Value
	case *gql_ast.List:
		return "Array"
	default:
		panic("FixMe")
	}
}

func convertTypeName(n string) string {
	switch n {
	case "Int":
		return "Int32"
	case "Float":
		return "Float64"
	case "Boolean":
		return "Bool"
		//	case "ID":
		//		return "String" //anything else for this?!
	default:
		//Assume a reference to a user-defined type for now.
		return n
	}
}

func stringValue(sv *gql_ast.StringValue) string {
	if sv == nil {
		return ""
	}
	return sv.Value
}

func commentValue(descr string) string {
	//SADL comments do not contain unescaped newlines
	return strings.Replace(descr, "\n", " ", -1)
}

func gqlEnum(schema *sadl.Schema, def *gql_ast.EnumDefinition) error {
	td := &sadl.TypeDef{
		Name: def.Name.Value,
		TypeSpec: sadl.TypeSpec{
			Type: "Enum",
		},
	}
	if def.Description != nil {
		td.Comment = commentValue(def.Description.Value)
	}
	for _, symdef := range def.Values {
		el := &sadl.EnumElementDef{
			Symbol: symdef.Name.Value,
		}
		if symdef.Description != nil {
			el.Comment = commentValue(symdef.Description.Value)
		}
		td.Elements = append(td.Elements, el)
	}
	schema.Types = append(schema.Types, td)
	return nil
}

func gqlUnion(schema *sadl.Schema, def *gql_ast.UnionDefinition) error {
	td := &sadl.TypeDef{
		Name: def.Name.Value,
		TypeSpec: sadl.TypeSpec{
			Type: "Union",
		},
	}
	if def.Description != nil {
		td.Comment = commentValue(def.Description.Value)
	}
	for _, vardef := range def.Types {
		tn := vardef.Name.Value
		vd := &sadl.UnionVariantDef{}
		vd.Name = tn
		vd.Type = tn
		td.Variants = append(td.Variants, vd)
	}
	schema.Types = append(schema.Types, td)
	return nil
}

func gqlStruct(schema *sadl.Schema, structDef *gql_ast.ObjectDefinition) error {
	td := &sadl.TypeDef{
		Name:    structDef.Name.Value,
		Comment: commentValue(stringValue(structDef.Description)),
		TypeSpec: sadl.TypeSpec{
			Type: "Struct",
		},
	}
	for _, fnode := range structDef.Fields {
		f := (*gql_ast.FieldDefinition)(fnode)
		fd := &sadl.StructFieldDef{
			Name:    f.Name.Value,
			Comment: commentValue(stringValue(f.Description)),
		}
		switch t := (*gql_ast.FieldDefinition)(fnode).Type.(type) {
		case *gql_ast.Named:
			fd.Type = convertTypeName(t.Name.Value)
		case *gql_ast.List:
			fd.Type = "Array"
			switch it := t.Type.(type) {
			case *gql_ast.Named:
				fd.Items = convertTypeName(it.Name.Value)
			case *gql_ast.NonNull:
				switch it := it.Type.(type) {
				case *gql_ast.Named:
					fd.Items = convertTypeName(it.Name.Value)
				default:
					panic("inline list type not supported")
				}
			default:
				panic("list type not supported")
			}
		case *gql_ast.NonNull:
			fd.Required = true
			switch t := t.Type.(type) {
			case *gql_ast.Named:
				fd.Type = convertTypeName(t.Name.Value)
			case *gql_ast.List:
				fd.Type = "Array"
				switch it := t.Type.(type) {
				case *gql_ast.Named:
					fd.Items = convertTypeName(it.Name.Value)
				case *gql_ast.NonNull:
					fd.Items = convertTypeName(typeName(it.Type))
				default:
					panic("inline list type not supported")
				}
			default:
				fd.Type = convertTypeName(typeName(t))
			}
		}
		td.Fields = append(td.Fields, fd)
	}
	schema.Types = append(schema.Types, td)
	return nil
}

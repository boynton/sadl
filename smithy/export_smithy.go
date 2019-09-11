package smithy

import (
	"fmt"
	"strings"

	"github.com/boynton/sadl"
)

func FromSADL(schema *sadl.Model, ns string) (*Model, error) {
	model := &Model{
		Version: "0.4.0",
		Namespaces: make(map[string]*Namespace, 0),
	}
	theNamespace := &Namespace{
		Shapes: make(map[string]*Shape, 0),
	}
	model.Namespaces[ns] = theNamespace
	for _, td := range schema.Types {
		err := defineShapeFromTypeSpec(schema, theNamespace, &td.TypeSpec, td.Name, td.Comment, td.Annotations)
		if err != nil {
			return nil, err
		}
	}
	for _, hd := range schema.Http {
		expectedCode := 200
		if hd.Expected != nil {
			expectedCode = int(hd.Expected.Status)
		}
		path := hd.Path
		query := ""
		n := strings.Index(path, "?")
		if n >= 0 {
			//todo: could find required queryparams, include them into the path for matching purposes. Literal query matches in Smithy
			//don't seem so useful.
			query = path[n+1:]
			path = path[:n]
		}
		if query != "" {
			fmt.Println("what to do with the  query:", query)
		}
		name := capitalize(hd.Name)
		if name == "" {
			name = capitalize(strings.ToLower(hd.Method)) + "Something" //fix!
		}
		shape := Shape{
			Type: "operation",
		}
		shape.Http = &Http{
			Uri: path,
			Method: hd.Method,
			Code: expectedCode,
		}
		switch hd.Method {
		case "GET", "PUT", "DELETE":
//			shape.Idempotent = true
		}

		//if we have any inputs, define this
		shape.Input = name + "Input"

		//if we have any outputs, define this
		shape.Output = name + "Output"

		//if we have any exceptions, define them
		if len(hd.Exceptions) > 0 {
			for _, e := range hd.Exceptions {
				shape.Errors = append(shape.Errors, e.Type)
				//make sure e.Type has a @httpError error code trait on it...and a "server" or "client" error attribute, for that
				//matter.
			}
		}
		theNamespace.Shapes[name] = &shape
/*
//from the smithy docs:
@http(method: "PUT", uri: "/{bucketName}/{key}", code: 200)
operation PutObject(PutObjectInput)
structure PutObjectInput {
    // Sent in the URI label named "key".
    @required
    @httpLabel
    key: ObjectKey,

    // Sent in the URI label named "bucketName".
    @required
    @httpLabel
    bucketName: String,

    // Sent in the X-Foo header
    @httpHeader("X-Foo")
    foo: String,

    // Sent in the query string as paramName
    @httpQuery("paramName")
    someValue: String,

    // Sent in the body
    data: MyBlob,

    // Sent in the body
    additional: String,
}

*/		
	}
	return model, nil
}

func typeReference(ts *sadl.TypeSpec) string {
	switch ts.Type {
	case "Bool":
		return "Boolean"
	case "Int8":
		return "Byte"
	case "Int16":
		return "Short"
	case "Int32":
		return "Integer"
	case "Int64":
		return "Long"
	case "Float32":
		return "Float"
	case "Float64":
		return "Double"
	case "Decimal":
		return "BigDecimal"
	case "Timestamp":
		return "Timestamp"
	case "UUID":
		return "String" //!
	case "Bytes":
		return "Blob"
	case "String":
		return "String"
	case "Array":
		return "List"
	case "Map":
		return "Map"
//	case "Struct": /naked struct
//		return "?"
	default:
		return ts.Type
	}
}

func listTypeReference(schema *sadl.Model, ns *Namespace, prefix string, fd *sadl.StructFieldDef) string {
	ftype := capitalize(prefix) + capitalize(fd.Name)
	td := schema.FindType(ftype)
	if td != nil {
		fmt.Printf("Inline defs not allowed, synthesize %q to refer to: %s\n", ftype, sadl.Pretty(fd))
		panic("Already have one with that name!!!")
	}
	shape := Shape{
		Type: "list",
	}
	shape.Member = &Member{
		Target: fd.Items,
	}
	shape.Documentation = "[autogenerated for field '" + fd.Name + "' in struct '" + prefix + "']";
	ns.Shapes[ftype] = &shape
	return ftype
}

func capitalize(s string) string {
	return strings.ToUpper(s[0:1]) + s[1:]
}

func defineShapeFromTypeSpec(model *sadl.Model, ns *Namespace, ts *sadl.TypeSpec, name string, comment string, annos map[string]string) error {
		var shape Shape
		switch ts.Type {
		case "String":
			shape = shapeFromString(ts)
		case "Enum":
			shape = shapeFromEnum(ts)
		case "Struct":
			//schema *sadl.Model, ns *Namespace, tname string, ts *sadl.TypeSpec
			shape = shapeFromStruct(model, ns, name, ts)
		default:
			fmt.Println("So far:", sadl.Pretty(model))
			panic("handle this type:" + sadl.Pretty(ts))
		}
		if comment != "" {
			shape.Documentation = comment
		}
		if annos != nil {
			for k, v := range annos {
				switch k {
				case "x_sensitive":
					shape.Sensitive = true
				case "x_deprecated":
					dep := &Deprecated{}
					if v != "" {
						n := strings.Index(v, "|")
						if n >= 0 {
							dep.Since = v[:n]
							dep.Message = v[n+1:]
						} else {
							dep.Message = v
						}
					}
					shape.Deprecated = dep
				}
			}
		}
		ns.Shapes[name] = &shape
	return nil
}

func shapeFromStruct(schema *sadl.Model, ns *Namespace, tname string, ts *sadl.TypeSpec) Shape {
	shape := Shape{
		Type: "structure",
	}
	members := make(map[string]*Member, 0)
	for _, fd := range ts.Fields {
		ftype := typeReference(&fd.TypeSpec)
		switch ftype {
		case "List":
			ftype = listTypeReference(schema, ns, tname, fd)
		}
		member := &Member{
			Target: ftype,
		}
		members[fd.Name] = member
	}
	shape.Members = members
	return shape
}

func shapeFromString(ts *sadl.TypeSpec) Shape {
	shape := Shape{
		Type: "string",
	}
	min := int64(-1)
	max := int64(-1)
	if ts.MinSize != nil {
		min = *ts.MinSize
	}
	if ts.MaxSize != nil {
		max = *ts.MaxSize
	}
	shape.Length = length(min, max)
	if ts.Pattern != "" {
		shape.Pattern = ts.Pattern
	}
	return shape
}

func shapeFromEnum(ts *sadl.TypeSpec) Shape {
	shape := Shape{
		Type: "string",
	}
	//for sadl, enum values *are* the symbols, so the name must be set to match the key
	//note that this same form can work with values, where the name is optional but the key is the actual value
	items := make(map[string]*Item, 0)
	shape.Enum = items
	for _, el := range ts.Elements {
		item := &Item{
			Name: el.Symbol, //the programmatic name, might be different than the value itself in Smithy. In SADL, they are the same.
			Documentation: el.Comment,
		}
		items[el.Symbol] = item
		//el.Annotations -> if contains x_tags, then expand to item.Tags
	}
	return shape
}

func length(min int64, max int64) *Length {
	l := &Length{}
	if min < 0 && max < 0 {
		return nil
	}
	if min >= 0 {
		l.Min = &min
	}
	if max >= 0 {
		l.Max = &max
	}
	return l
}

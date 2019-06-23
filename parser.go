package sadl

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

//
// import "github.com/boynton/sadl"
// ...
// model, err := sadl.ParseFile("/some/path")
//
func ParseFile(path string, extensions ...Extension) (*Model, error) {
	return parseFile(path, extensions)
}

//
// import "github.com/boynton/sadl"
// ...
// model, err := sadl.ParseString("...")
//
func ParseString(src string, extensions ...Extension) (*Model, error) {
	return parseString(src, extensions)
}

//----------------

type Parser struct {
	path           string
	source         string
	scanner        *Scanner
	model          *Model
	schema         *Schema
	lastToken      *Token
	prevLastToken  *Token
	ungottenToken  *Token
	currentComment string
	extensions     map[string]Extension
}

type Extension interface {
	Name() string
	Result() interface{}
	Parse(p *Parser) error
	Validate(p *Parser) error
}

func parseFile(path string, extensions []Extension) (*Model, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	src := string(b)
	p := &Parser{
		scanner: NewScanner(strings.NewReader(src)),
		path:    path,
		source:  src,
	}
	return p.Parse(extensions)
}

func parseString(src string, extensions []Extension) (*Model, error) {
	p := &Parser{
		scanner: NewScanner(strings.NewReader(src)),
		source:  src,
	}
	return p.Parse(extensions)
}

func (p *Parser) CurrentComment() string {
	return p.currentComment
}

func (p *Parser) Model() *Model {
	return p.model
}

func (p *Parser) UngetToken() {
	Debug("UngetToken() -> ", p.lastToken)
	p.ungottenToken = p.lastToken
	p.lastToken = p.prevLastToken
}

func (p *Parser) GetToken() *Token {
	if p.ungottenToken != nil {
		p.lastToken = p.ungottenToken
		p.ungottenToken = nil
		Debug("GetToken() -> ", p.lastToken)
		return p.lastToken
	}
	p.prevLastToken = p.lastToken
	tok := p.scanner.Scan()
	for {
		if tok.Type == EOF {
			return nil //fixme
		} else if tok.Type != BLOCK_COMMENT {
			break
		}
		tok = p.scanner.Scan()
	}
	p.lastToken = &tok
	Debug("GetToken() -> ", p.lastToken)
	return p.lastToken
}

func (p *Parser) Source() string {
	source := p.source
	if p.path != "" && source == "" {
		data, err := ioutil.ReadFile(p.path)
		if err == nil {
			source = string(data)
		}
	}
	return source
}

func (p *Parser) Parse(extensions []Extension) (*Model, error) {
	for _, ext := range extensions {
		err := p.registerExtension(ext)
		if err != nil {
			return nil, err
		}
	}
	p.schema = &Schema{
		Types: make([]*TypeDef, 0),
	}
	if p.schema.Name == "" {
		p.schema.Name = BaseFileName(p.path)
	}
	comment := ""
	for {
		var err error
		tok := p.GetToken()
		if tok == nil {
			break
		}
		switch tok.Type {
		case SYMBOL:
			switch tok.Text {
			case "name":
				err = p.parseNameDirective(comment)
			case "version":
				err = p.parseVersionDirective(comment)
			case "type":
				err = p.parseTypeDirective(comment)
			case "example":
				err = p.parseExampleDirective(comment)
			case "base":
				err = p.parseBaseDirective(comment)
			case "rpc", "action":
				err = p.parseActionDirective(comment)
			case "http":
				err = p.parseHttpDirective(comment)
			default:
				if strings.HasPrefix(tok.Text, "x_") {
					p.schema.Comment = p.MergeComment(p.schema.Comment, comment)
					p.schema.Annotations, comment, err = p.parseExtendedOptionTopLevel(p.schema.Annotations, tok.Text)
					p.schema.Comment = p.MergeComment(p.schema.Comment, comment)
				} else {
					if p.extensions != nil {
						if handler, ok := p.extensions[tok.Text]; ok {
							p.currentComment = comment
							err = handler.Parse(p)
						}
					}
				}
			}
			comment = ""
		case LINE_COMMENT:
			comment = p.MergeComment(comment, tok.Text)
		case SEMICOLON:
			/* ignore */
		case NEWLINE:
			/* ignore */
		default:
			return nil, p.expectedDirectiveError()
		}
		if err != nil {
			return nil, err
		}
	}
	var err error
	p.model, err = NewModel(p.schema)
	p.schema = nil
	if err != nil {
		return nil, err
	}
	if extensions != nil {
		p.model.Extensions = make(map[string]interface{})
		for _, ext := range extensions {
			p.model.Extensions[ext.Name()] = ext.Result()
		}
	}
	return p.Validate()
}

func (p *Parser) parseNameDirective(comment string) error {
	p.schema.Comment = p.MergeComment(p.schema.Comment, comment)
	txt, err := p.expectText()
	if err == nil {
		p.schema.Name = txt
	}
	return err
}

func (p *Parser) parseVersionDirective(comment string) error {
	p.schema.Comment = p.MergeComment(p.schema.Comment, comment)
	tok := p.GetToken()
	if tok == nil {
		return p.EndOfFileError()
	}
	switch tok.Type {
	case NUMBER, SYMBOL, STRING:
		p.schema.Version = tok.Text
		return nil
	default:
		return p.Error("Bad version value: " + tok.Text)
	}
}

func (p *Parser) parseBaseDirective(comment string) error {
	p.schema.Comment = p.MergeComment(p.schema.Comment, comment)
	base, err := p.ExpectString()
	if err == nil {
		p.schema.Base = base
		if base != "" && !strings.HasPrefix(base, "/") {
			err = p.Error("Bad base path value: " + base)
		}
	}
	return err
}

func (p *Parser) parseActionDirective(comment string) error {
	name, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	err = p.expect(OPEN_PAREN)
	if err != nil {
		return err
	}
	tok := p.GetToken()
	if tok == nil {
		return p.EndOfFileError()
	}
	input := ""
	if tok.Type == SYMBOL {
		input = tok.Text
		tok = p.GetToken()
		if tok == nil {
			return p.EndOfFileError()
		}
	}
	if tok.Type != CLOSE_PAREN {
		return p.SyntaxError()
	}
	output := ""
	var options *Options
	var etypes []string
	tok = p.GetToken()
	if tok != nil {
		if tok.Type == SYMBOL {
			if tok.Text != "except" {
				output = tok.Text
				tok = p.GetToken()
			}
			if tok != nil && tok.Type == SYMBOL && tok.Text == "except" {
				for {
					etype := p.getIdentifier()
					if etype == "" {
						if len(etypes) == 0 {
							return p.SyntaxError()
						}
						break
					}
					etypes = append(etypes, etype)
				}
			} else {
				if tok != nil {
					if tok.Type == SYMBOL {
						return p.SyntaxError()
					}
					p.UngetToken()
				}
			}
		} else {
			p.UngetToken()
		}
	}
	options, err = p.ParseOptions("action", []string{})
	if err != nil {
		return err
	}
	comment, err = p.EndOfStatement(comment)
	action := &ActionDef{
		Name:        name,
		Input:       input,
		Output:      output,
		Exceptions:  etypes,
		Comment:     comment,
		Annotations: options.Annotations,
	}
	p.schema.Actions = append(p.schema.Actions, action)
	return nil
}

func (p *Parser) getIdentifier() string {
	tok := p.GetToken()
	if tok == nil {
		return ""
	}
	if tok.Type == COMMA {
		//ignore the comma, try again
		return p.getIdentifier()
	}
	if tok.Type == SYMBOL {
		return tok.Text
	}
	p.UngetToken()
	return ""
}

func (p *Parser) parseHttpDirective(comment string) error {
	sym, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	var method string
	up := strings.ToUpper(sym)
	switch up {
	case "POST", "GET", "PUT", "DELETE": //HEAD, OPTIONS
		method = up
	default:
		return p.Error(fmt.Sprintf("HTTP 'method' invalid: %s", sym))
	}
	pathTemplate, err := p.ExpectString()
	if err != nil {
		return err
	}
	options, err := p.ParseOptions("http", []string{"action"})
	if err != nil {
		return err
	}
	op := &HttpDef{
		Method:      method,
		Path:        pathTemplate,
		Name:        options.Action,
		Annotations: options.Annotations,
	}
	tok := p.GetToken()
	if tok == nil {
		return p.EndOfFileError()
	}
	if tok.Type == OPEN_BRACE {
		var done bool
		op.Comment = p.ParseTrailingComment(comment)
		comment = ""
		for {
			done, comment, err = p.IsBlockDone(comment)
			if done {
				break
			}
			err := p.parseHttpSpec(op, comment, true)
			comment = ""
			if err != nil {
				return err
			}
		}
		op.Comment, err = p.EndOfStatement(op.Comment)
		p.schema.Http = append(p.schema.Http, op)
	} else {
		return p.SyntaxError()
	}
	return nil
}

func (p *Parser) parseHttpSpec(op *HttpDef, comment string, top bool) error {
	pathTemplate := op.Path
	ename, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	if ename == "expect" {
		if !top {
			err = p.SyntaxError()
		} else {
			return p.parseHttpExpectedSpec(op, comment)
		}
	} else if ename == "except" {
		if !top {
			err = p.SyntaxError()
		} else {
			return p.parseHttpExceptionSpec(op, comment)
		}
	}
	if err != nil {
		return err
	}
	ts, _, comment, err := p.ParseTypeSpec(comment)
	if err != nil {
		return err
	}
	options, err := p.ParseOptions("HttpParam", []string{"header", "default", "required"})
	if err != nil {
		return err
	}
	comment, err = p.EndOfStatement(comment)
	spec := &HttpParamSpec{
		StructFieldDef: StructFieldDef{
			Name:        ename,
			Annotations: options.Annotations,
			Comment:     comment,
			TypeSpec:    *ts,
		},
	}
	if options.Default != "" {
		spec.Default = options.Default
	}
	if options.Header != "" {
		spec.Header = options.Header
	} else {
		if top {
			paramType, paramName := p.parameterSource(pathTemplate, ename)
			switch paramType {
			case "path":
				spec.Path = true
			case "query":
				spec.Query = paramName
			case "body":
			default:
			}
		}
	}
	if top {
		op.Inputs = append(op.Inputs, spec)
	} else {
		op.Expected.Outputs = append(op.Expected.Outputs, spec)
	}
	return nil
}

func (p *Parser) parseHttpExpectedSpec(op *HttpDef, comment string) error {
	if op.Expected != nil {
		return p.Error("Only a single 'expect' directive is allowed per HTTP action")
	}
	estatus, err := p.expectInt32()
	if err != nil {
		return err
	}
	options, err := p.ParseOptions("HttpResponse", []string{})
	if err != nil {
		return err
	}
	op.Expected = &HttpExpectedSpec{
		Status:      estatus,
		Annotations: options.Annotations,
	}
	tok := p.GetToken()
	if tok == nil {
		return p.EndOfFileError()
	}
	if tok.Type == OPEN_BRACE {
		op.Expected.Comment = p.ParseTrailingComment(comment)
		comment = ""
		for {
			done, comment, err := p.IsBlockDone(comment)
			if done {
				op.Expected.Comment = p.MergeComment(op.Expected.Comment, comment)
				break
			}
			err = p.parseHttpSpec(op, comment, false)
			if err != nil {
				return err
			}
		}
	} else {
		p.UngetToken()
	}
	op.Expected.Comment, err = p.EndOfStatement(op.Expected.Comment)
	return err
}

func (p *Parser) parseHttpExceptionSpec(op *HttpDef, comment string) error {
	estatus, err := p.expectInt32()
	if err != nil {
		return err
	}
	etype, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	options, err := p.ParseOptions("HttpResponse", []string{})
	if err != nil {
		return err
	}
	exc := &HttpExceptionSpec{
		Type:        etype,
		Status:      estatus,
		Annotations: options.Annotations,
	}
	exc.Comment, err = p.EndOfStatement(comment)
	if err != nil {
		return err
	}
	op.Exceptions = append(op.Exceptions, exc)
	return nil

}

func (p *Parser) parameterSource(pathTemplate, name string) (string, string) {
	path := pathTemplate
	query := ""
	n := strings.Index(path, "?")
	if n >= 0 {
		query = path[n+1:]
		path = path[:n]
	}
	match := "{" + name + "}" //fixme: wildcard for the end of the path
	if strings.Index(path, match) >= 0 {
		return "path", ""
	}
	for _, qparam := range strings.Split(query, "&") {
		kv := strings.Split(qparam, "=")
		if len(kv) > 1 && kv[1] == match {
			return "query", kv[0]
		}
	}
	return "", ""
}

func (p *Parser) IsBlockDone(comment string) (bool, string, error) {
	tok := p.GetToken()
	if tok == nil {
		return false, comment, p.EndOfFileError()
	}
	for {
		if tok.Type == CLOSE_BRACE {
			return true, comment, nil
		} else if tok.Type == LINE_COMMENT {
			comment = p.MergeComment(comment, tok.Text)
			tok = p.GetToken()
			if tok == nil {
				return false, comment, p.EndOfFileError()
			}
		} else if tok.Type == NEWLINE {
			tok = p.GetToken()
			if tok == nil {
				return false, comment, p.EndOfFileError()
			}
		} else {
			p.UngetToken()
			return false, comment, nil
		}
	}
}

func (p *Parser) parseExampleDirective(comment string) error {
	target, err := p.ExpectCompoundIdentifier()
	if err != nil {
		return err
	}
	val, err := p.parseLiteralValue()
	if err == nil {
		ex := &ExampleDef{
			Target:  target,
			Example: val,
		}
		p.schema.Examples = append(p.schema.Examples, ex)
	}
	return err
}

func (p *Parser) parseTypeDirective(comment string) error {
	typeName, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	superName, params, fields, elements, options, comment2, err := p.ParseTypeSpecElements() //note that this can return user-defined types
	if err != nil {
		return err
	}
	comment = p.MergeComment(comment, comment2)
	td := &TypeDef{
		TypeSpec: TypeSpec{
			Type: superName,
		},
		Name:    typeName,
		Comment: comment,
	}
	switch superName {
	case "Any":
		err = p.parseAnyDef(td)
	case "Bool":
		err = p.parseBoolDef(td)
	case "Int8", "Int16", "Int32", "Int64", "Float32", "Float64", "Decimal":
		err = p.parseNumberDef(td)
	case "Bytes":
		err = p.parseBytesDef(td)
	case "String":
		err = p.parseStringDef(td)
	case "Timestamp":
		err = p.parseTimestampDef(td)
	case "UUID":
		err = p.parseUUIDDef(td)
	case "UnitValue":
		err = p.parseUnitValueDef(td, params)
	case "Array":
		err = p.parseArrayDef(td, params)
	case "Map":
		err = p.parseMapDef(td, params)
	case "Struct":
		err = p.parseStructDef(td, fields)
		td.Annotations = options.Annotations
	case "Enum":
		td.Elements = elements
		td.Annotations = options.Annotations
	case "Union":
		err = p.parseUnionDef(td, params)
	default:
		err = p.Error(fmt.Sprintf("Super type must be a base type: %v", superName))
	}
	if err != nil {
		return err
	}
	p.schema.Types = append(p.schema.Types, td)
	return nil
}

func (p *Parser) ParseTypeSpec(comment string) (*TypeSpec, *Options, string, error) {
	tsType, tsParams, tsFields, tsElements, options, tsComment, err := p.ParseTypeSpecElements()
	if err != nil {
		return nil, nil, "", err
	}
	comment = p.MergeComment(comment, tsComment) //?
	var tsKeys, tsItems string
	var tsUnit, tsValue string
	switch tsType {
	case "Array":
		tsItems, err = p.arrayParams(tsParams)
	case "Map":
		tsKeys, tsItems, err = p.mapParams(tsParams)
	case "UnitValue":
		tsValue, tsUnit, err = p.unitValueParams(tsParams)
	default:
		if len(tsParams) != 0 {
			//unions!?
			err = p.SyntaxError()
		}
	}
	if err != nil {
		return nil, nil, "", err
	}
	ts := &TypeSpec{
		Type:     tsType,
		Items:    tsItems,
		Keys:     tsKeys,
		Value:    tsValue,
		Unit:     tsUnit,
		Fields:   tsFields,
		Elements: tsElements,
	}
	return ts, options, comment, nil
}

func (p *Parser) ParseTypeSpecElements() (string, []string, []*StructFieldDef, []*EnumElementDef, *Options, string, error) {
	options := &Options{}
	typeName, err := p.ExpectIdentifier()
	if err != nil {
		return "", nil, nil, nil, options, "", err
	}
	tok := p.GetToken()
	if tok == nil {
		return typeName, nil, nil, nil, options, "", nil
	}
	if tok.Type == OPEN_ANGLE {
		var params []string
		var expectedParams int
		switch typeName {
		case "Array":
			expectedParams = 1
		case "Map", "UnitValue":
			expectedParams = 2
		case "Union":
			expectedParams = -1
		default:
			return typeName, nil, nil, nil, options, "", p.SyntaxError()
		}
		for {
			tok = p.GetToken()
			if tok == nil {
				return typeName, nil, nil, nil, options, "", p.EndOfFileError()
			}
			if tok.Type != COMMA {
				if tok.Type == CLOSE_ANGLE {
					if expectedParams >= 0 && expectedParams != len(params) {
						return typeName, nil, nil, nil, options, "", p.SyntaxError()
					}
					return typeName, params, nil, nil, options, "", nil
				}
				if tok.Type != SYMBOL {
					return typeName, params, nil, nil, options, "", p.SyntaxError()
				}
				params = append(params, tok.Text)
			}
		}
	} else if typeName == "Struct" || typeName == "Enum" {
		if tok.Type != OPEN_BRACE {
			p.UngetToken()
			options, err = p.ParseOptions(typeName, []string{})
			if err != nil {
				return typeName, nil, nil, nil, options, "", err
			}
			tok = p.GetToken()
			if tok == nil {
				return typeName, nil, nil, nil, options, "", p.EndOfFileError()
			}
		}
		if tok.Type == OPEN_BRACE {
			comment := p.ParseTrailingComment("")
			switch typeName {
			case "Struct":
				var fields []*StructFieldDef
				tok := p.GetToken()
				if tok == nil {
					return typeName, nil, fields, nil, options, comment, p.SyntaxError()
				}
				if tok.Type != NEWLINE {
					p.UngetToken()
				}
				for {
					field, err := p.parseStructFieldDef()
					if err != nil {
						return typeName, nil, fields, nil, options, comment, err
					}
					if field == nil {
						break
					}
					fields = append(fields, field)
				}
				comment, err = p.EndOfStatement(comment)
				p.UngetToken() //the closing brace
				return typeName, nil, fields, nil, options, comment, nil
			case "Enum":
				var elements []*EnumElementDef
				tok := p.GetToken()
				if tok == nil {
					return typeName, nil, nil, nil, options, comment, p.SyntaxError()
				}
				if tok.Type != NEWLINE {
					p.UngetToken()
				}
				for {
					element, err := p.parseEnumElementDef()
					if err != nil {
						return typeName, nil, nil, nil, options, comment, err
					}
					if element == nil {
						break
					}
					elements = append(elements, element)
				}
				comment, err = p.EndOfStatement(comment)
				p.UngetToken() //the closing brace
				return typeName, nil, nil, elements, options, comment, nil
			}
			return typeName, nil, nil, nil, options, comment, p.SyntaxError()
		}
	}
	p.UngetToken()
	return typeName, nil, nil, nil, options, "", nil
}

func (p *Parser) parseAnyDef(td *TypeDef) error {
	err := p.parseTypeOptions(td)
	if err == nil {
		td.Comment, err = p.EndOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseBoolDef(td *TypeDef) error {
	err := p.parseTypeOptions(td)
	if err == nil {
		td.Comment, err = p.EndOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseNumberDef(td *TypeDef) error {
	err := p.parseTypeOptions(td, "min", "max")
	if err == nil {
		td.Comment, err = p.EndOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseBytesDef(td *TypeDef) error {
	err := p.parseTypeOptions(td, "minsize", "maxsize")
	if err == nil {
		td.Comment, err = p.EndOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseStringDef(td *TypeDef) error {
	err := p.parseTypeOptions(td, "minsize", "maxsize", "pattern", "values", "reference")
	if err == nil {
		td.Comment, err = p.EndOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseTimestampDef(td *TypeDef) error {
	err := p.parseTypeOptions(td)
	if err == nil {
		td.Comment, err = p.EndOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseUUIDDef(td *TypeDef) error {
	err := p.parseTypeOptions(td, "reference")
	if err == nil {
		td.Comment, err = p.EndOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseUnitValueDef(td *TypeDef, params []string) error {
	err := p.parseTypeOptions(td)
	if err == nil {
		td.Value, td.Unit, err = p.unitValueParams(params)
		if err == nil {
			td.Comment, err = p.EndOfStatement(td.Comment)
		}
	}
	return err
}

func (p *Parser) parseArrayDef(td *TypeDef, params []string) error {
	var err error
	td.Items, err = p.arrayParams(params)
	if err == nil {
		err = p.parseTypeOptions(td, "minsize", "maxsize")
		if err == nil {
			td.Comment, err = p.EndOfStatement(td.Comment)
		}
	}
	return err
}

func (p *Parser) parseMapDef(td *TypeDef, params []string) error {
	var err error
	td.Keys, td.Items, err = p.mapParams(params)
	if err == nil {
		err = p.parseTypeOptions(td, "minsize", "maxsize")
		if err == nil {
			td.Comment, err = p.EndOfStatement(td.Comment)
		}
	}
	return err
}

func (p *Parser) parseStructDef(td *TypeDef, fields []*StructFieldDef) error {
	err := p.parseTypeOptions(td)
	if err == nil {
		td.Fields = fields
	}
	return err
}

func (p *Parser) parseEnumDef(td *TypeDef, elements []*EnumElementDef) error {
	td.Elements = elements
	return nil
}

func (p *Parser) parseUnionDef(td *TypeDef, params []string) error {
	err := p.parseTypeOptions(td)
	if err == nil {
		td.Variants = params
		td.Comment, err = p.EndOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) Error(msg string) error {
	Debug("*** error, last token:", p.lastToken)
	return fmt.Errorf("*** %s\n", formattedAnnotation(p.path, p.Source(), "", msg, p.lastToken, RED, 5))
}

func (p *Parser) SyntaxError() error {
	return p.Error("Syntax error")
}

func (p *Parser) EndOfFileError() error {
	return p.Error("Unexpected end of file")
}

func (p *Parser) assertIdentifier(tok *Token) (string, error) {
	if tok == nil {
		return "", p.EndOfFileError()
	}
	if tok.Type == SYMBOL {
		return tok.Text, nil
	}
	return tok.Text, p.Error(fmt.Sprintf("Expected symbol, found %v", tok.Type))
}

func (p *Parser) ExpectIdentifier() (string, error) {
	tok := p.GetToken()
	return p.assertIdentifier(tok)
}

func (p *Parser) ExpectCompoundIdentifier() (string, error) {
	tok := p.GetToken()
	s, err := p.assertIdentifier(tok)
	if err != nil {
		return s, err
	}
	tok = p.GetToken()
	if tok == nil {
		return s, nil
	}
	if tok.Type != DOT {
		p.UngetToken()
		return s, nil
	}
	ss, err := p.ExpectCompoundIdentifier()
	if err != nil {
		return "", err
	}
	return s + "." + ss, nil
}

func (p *Parser) expectEqualsIdentifier() (string, error) {
	err := p.expect(EQUALS)
	if err != nil {
		return "", err
	}
	return p.ExpectIdentifier()
}

func (p *Parser) assertString(tok *Token) (string, error) {
	if tok == nil {
		return "", p.EndOfFileError()
	}
	if tok.Type == STRING {
		return tok.Text, nil
	}
	return tok.Text, p.Error(fmt.Sprintf("Expected string, found %v", tok.Type))
}

func (p *Parser) ExpectString() (string, error) {
	tok := p.GetToken()
	return p.assertString(tok)
}

func (p *Parser) expectEqualsString() (string, error) {
	err := p.expect(EQUALS)
	if err != nil {
		return "", err
	}
	return p.ExpectString()
}

func (p *Parser) expectText() (string, error) {
	tok := p.GetToken()
	if tok == nil {
		return "", fmt.Errorf("Unexpected end of file")
	}
	if tok.IsText() {
		return tok.Text, nil
	}
	return "", fmt.Errorf("Expected symbol or string, found %v", tok.Type)
}

func (p *Parser) expectInt32() (int32, error) {
	tok := p.GetToken()
	if tok == nil {
		return 0, p.EndOfFileError()
	}
	if tok.IsNumeric() {
		l, err := strconv.ParseInt(tok.Text, 10, 32)
		return int32(l), err
	}
	return 0, p.Error(fmt.Sprintf("Expected number, found %v", tok.Type))
}

func (p *Parser) expectEqualsInt32() (*int32, error) {
	var val int32
	err := p.expect(EQUALS)
	if err != nil {
		return nil, err
	}
	val, err = p.expectInt32()
	if err != nil {
		return nil, err
	}
	return &val, nil
}

func (p *Parser) expectInt64() (int64, error) {
	tok := p.GetToken()
	if tok == nil {
		return 0, p.EndOfFileError()
	}
	if tok.IsNumeric() {
		l, err := strconv.ParseInt(tok.Text, 10, 64)
		return int64(l), err
	}
	return 0, p.Error(fmt.Sprintf("Expected number, found %v", tok.Type))
}

func (p *Parser) expectEqualsInt64() (*int64, error) {
	var val int64
	err := p.expect(EQUALS)
	if err != nil {
		return nil, err
	}
	val, err = p.expectInt64()
	if err != nil {
		return nil, err
	}
	return &val, nil
}

func (p *Parser) expectNumber() (*Decimal, error) {
	tok := p.GetToken()
	if tok == nil {
		return nil, p.EndOfFileError()
	}
	if tok.IsNumeric() {
		return ParseDecimal(tok.Text)
	}
	return nil, p.Error(fmt.Sprintf("Expected number, found %v", tok.Type))
}

func (p *Parser) expectEqualsNumber() (*Decimal, error) {
	err := p.expect(EQUALS)
	if err != nil {
		return nil, err
	}
	return p.expectNumber()
}

func (p *Parser) expect(toktype TokenType) error {
	tok := p.GetToken()
	if tok == nil {
		return p.EndOfFileError()
	}
	if tok.Type == toktype {
		return nil
	}
	return p.Error(fmt.Sprintf("Expected %v, found %v", toktype, tok.Type))
}

func containsOption(options []string, option string) bool {
	if options != nil {
		for _, opt := range options {
			if opt == option {
				return true
			}
		}
	}
	return false
}

func (p *Parser) parseTypeOptions(td *TypeDef, acceptable ...string) error {
	options, err := p.ParseOptions(td.Type, acceptable)
	if err == nil {
		td.Pattern = options.Pattern
		td.Values = options.Values
		td.MinSize = options.MinSize
		td.MaxSize = options.MaxSize
		td.Min = options.Min
		td.Max = options.Max
		td.Reference = options.Reference
		td.Annotations = options.Annotations
	}
	return err
}

type Options struct {
	Required    bool
	Default     interface{}
	Pattern     string
	Values      []string
	MinSize     *int64
	MaxSize     *int64
	Min         *Decimal
	Max         *Decimal
	Action      string
	Header      string
	Reference   string
	Annotations map[string]string
}

func (p *Parser) ParseOptions(typeName string, acceptable []string) (*Options, error) {
	options := &Options{}
	var err error
	tok := p.GetToken()
	if tok == nil {
		return options, nil
	}
	if tok.Type == OPEN_PAREN {
		for {
			tok := p.GetToken()
			if tok == nil {
				return nil, p.SyntaxError()
			}
			if tok.Type == CLOSE_PAREN {
				return options, nil
			}
			if tok.Type == SYMBOL {
				match := strings.ToLower(tok.Text)
				if strings.HasPrefix(match, "x_") {
					options.Annotations, err = p.parseExtendedOption(options.Annotations, tok.Text)
				} else if containsOption(acceptable, match) {
					switch match {
					case "min":
						options.Min, err = p.expectEqualsNumber()
					case "max":
						options.Max, err = p.expectEqualsNumber()
					case "minsize":
						options.MinSize, err = p.expectEqualsInt64()
					case "maxsize":
						options.MaxSize, err = p.expectEqualsInt64()
					case "pattern":
						options.Pattern, err = p.expectEqualsString()
					case "values":
						options.Values, err = p.expectEqualsStringArray()
					case "required":
						options.Required = true
					case "default":
						options.Default, err = p.parseEqualsLiteral()
					case "action":
						options.Action, err = p.expectEqualsIdentifier()
					case "reference":
						options.Reference, err = p.expectEqualsIdentifier()
					case "header":
						options.Header, err = p.expectEqualsString()
					default:
						err = p.Error("Unrecognized option: " + tok.Text)
					}
				} else {
					err = p.Error(fmt.Sprintf("Unrecognized option for %s: %s", typeName, tok.Text))
				}
				if err != nil {
					return nil, err
				}
			} else if tok.Type == COMMA {
				//ignore
			} else {
				return nil, p.SyntaxError()
			}
		}
	} else {
		p.UngetToken()
		return options, nil
	}
}

//parse the next string. And also a line comment, and the end of line, if present. Anything else is an error
func (p *Parser) parseStringToEndOfLine() (string, string, error) {
	val := ""
	comment := ""
	tok := p.GetToken()
	if tok == nil {
		return val, comment, nil
	}
	if tok.Type == EQUALS {
		//ignore it except error if at end of file
		tok = p.GetToken()
		if tok == nil {
			return "", "", p.EndOfFileError()
		}
	}
	if tok.Type == STRING {
		val = tok.Text
		tok = p.GetToken()
	}
	if tok == nil {
		return val, comment, nil
	}
	if tok.Type == LINE_COMMENT {
		comment = tok.Text
		tok = p.GetToken()
	}
	if tok == nil {
		return val, comment, nil
	}
	if tok.Type != NEWLINE {
		return "", "", p.SyntaxError()
	}
	return val, comment, nil
}

func (p *Parser) parseExtendedOptionTopLevel(annos map[string]string, anno string) (map[string]string, string, error) {
	val, comment, err := p.parseStringToEndOfLine()
	if annos == nil {
		annos = make(map[string]string, 0)
	}
	annos[anno] = val
	return annos, comment, err
}

func (p *Parser) parseExtendedOption(annos map[string]string, anno string) (map[string]string, error) {
	var err error
	var val string
	tok := p.GetToken()
	if tok != nil {
		if tok.Type == EQUALS {
			val, err = p.ExpectString()
		} else {
			p.UngetToken()
		}
	} else {
		err = p.EndOfFileError()
	}
	if err != nil {
		return annos, err
	}
	if annos == nil {
		annos = make(map[string]string, 0)
	}
	annos[anno] = val
	return annos, err
}

func (p *Parser) parseBytesOptions(typedef *TypeDef) error {
	tok := p.GetToken()
	if tok == nil {
		return p.SyntaxError()
	}
	expected := ""
	if tok.Type == OPEN_PAREN {
		for {
			tok := p.GetToken()
			if tok == nil {
				return p.SyntaxError()
			}
			if tok.Type == CLOSE_PAREN {
				return nil
			}
			if tok.Type == SYMBOL {
				switch tok.Text {
				case "minsize", "maxsize":
					expected = tok.Text
				default:
					return p.Error("invalid bytes option: " + tok.Text)
				}
			} else if tok.Type == EQUALS {
				if expected == "" {
					return p.SyntaxError()
				}
			} else if tok.Type == NUMBER {
				if expected == "" {
					return p.SyntaxError()
				}
				val, err := ParseDecimal(tok.Text)
				if err != nil {
					return err
				}
				if expected == "minsize" {
					i := val.AsInt64()
					typedef.MinSize = &i
				} else if expected == "maxsize" {
					i := val.AsInt64()
					typedef.MinSize = &i
				} else {
					return p.Error("bytes option must have numeric value")
				}
				expected = ""
			}
		}
	} else {
		p.UngetToken()
		return nil
	}
}

func (p *Parser) expectEqualsStringArray() ([]string, error) {
	var values []string
	tok := p.GetToken()
	if tok == nil {
		return nil, p.EndOfFileError()
	}
	if tok.Type != EQUALS {
		return nil, p.SyntaxError()
	}

	tok = p.GetToken()
	if tok == nil {
		return nil, p.EndOfFileError()
	}
	if tok.Type != OPEN_BRACKET {
		return nil, p.SyntaxError()
	}
	for {
		tok = p.GetToken()
		if tok == nil {
			return nil, p.EndOfFileError()
		}
		if tok.Type == CLOSE_BRACKET {
			break
		}
		if tok.Type == STRING {
			values = append(values, tok.Text)
		} else if tok.Type == COMMA || tok.Type == NEWLINE {
			//ignore
		} else {
			return nil, p.SyntaxError()
		}
	}
	return values, nil
}

func (p *Parser) parseEnumElementDef() (*EnumElementDef, error) {
	comment := ""
	sym := ""
	var err error
	for {
		tok := p.GetToken()
		if tok == nil {
			return nil, p.EndOfFileError()
		}
		if tok.Type == CLOSE_BRACE {
			return nil, nil
		} else if tok.Type == LINE_COMMENT {
			comment = p.MergeComment(comment, tok.Text)
		} else if tok.Type == SEMICOLON || tok.Type == NEWLINE || tok.Type == COMMA {
			//ignore
		} else {
			sym, err = p.assertIdentifier(tok)
			if err != nil {
				return nil, err
			}
			break
		}
	}
	options, err := p.ParseOptions("Enum", []string{})
	if err != nil {
		return nil, err
	}
	comment = p.ParseTrailingComment(comment)
	return &EnumElementDef{
		Symbol:      sym,
		Comment:     comment,
		Annotations: options.Annotations,
	}, nil
}

func (p *Parser) expectNewline() error {
	tok := p.GetToken()
	if tok == nil {
		return p.SyntaxError()
	}
	if tok.Type != NEWLINE {
		p.UngetToken()
		return p.SyntaxError()
	}
	return nil
}

func (p *Parser) parseStructFieldDef() (*StructFieldDef, error) {
	var comment string
	tok := p.GetToken()
	if tok == nil {
		return nil, p.EndOfFileError()
	}
	for {
		if tok.Type == CLOSE_BRACE {
			return nil, nil
		} else if tok.Type == LINE_COMMENT {
			comment = p.MergeComment("", tok.Text)
			tok = p.GetToken()
			if tok == nil {
				return nil, p.EndOfFileError()
			}
		} else if tok.Type == NEWLINE {
			tok = p.GetToken()
			if tok == nil {
				return nil, p.EndOfFileError()
			}
		} else {
			p.UngetToken()
			break
		}
	}
	fname, err := p.ExpectIdentifier()
	if err != nil {
		return nil, err
	}
	ts, foptions, fcomment, err := p.ParseTypeSpec(comment)
	if err != nil {
		return nil, err
	}
	comment = p.MergeComment(comment, fcomment)
	field := &StructFieldDef{
		Name:     fname,
		Comment:  comment,
		TypeSpec: *ts,
	}
	err = p.parseStructFieldOptions(field)
	if err != nil {
		return nil, err
	}
	if foptions != nil {
		if foptions.Annotations != nil && len(foptions.Annotations) > 0 {
			if field.Annotations == nil {
				field.Annotations = make(map[string]string, 0)
			}
			for k, v := range foptions.Annotations {
				field.Annotations[k] = v
			}
		}
	}
	field.Comment, err = p.EndOfStatement(field.Comment)
	return field, nil
}

func (p *Parser) parseStructFieldOptions(field *StructFieldDef) error {
	var acceptable []string
	switch field.Type {
	case "String":
		acceptable = []string{"pattern", "values", "minsize", "maxsize", "reference"}
	case "UUID":
		acceptable = []string{"reference"}
	case "Int8", "Int16", "Int32", "Int64", "Float32", "Float64", "Decimal":
		acceptable = []string{"min", "max"}
	case "Bytes", "Array", "Map":
		acceptable = []string{"minsize", "maxsize"}
	}
	acceptable = append(acceptable, "required")
	acceptable = append(acceptable, "default")
	options, err := p.ParseOptions(field.Type, acceptable)
	if err == nil {
		field.Required = options.Required
		field.Default = options.Default
		field.Pattern = options.Pattern
		field.Values = options.Values
		field.MinSize = options.MinSize
		field.MaxSize = options.MaxSize
		field.Min = options.Min
		field.Max = options.Max
		field.Annotations = options.Annotations
		field.Reference = options.Reference
	}
	return err
}

func (p *Parser) parseEqualsLiteral() (interface{}, error) {
	err := p.expect(EQUALS)
	if err != nil {
		return 0, err
	}
	return p.parseLiteralValue()
}

func (p *Parser) parseLiteralValue() (interface{}, error) {
	tok := p.GetToken()
	if tok == nil {
		return nil, p.SyntaxError()
	}
	return p.parseLiteral(tok)
}

func (p *Parser) parseLiteral(tok *Token) (interface{}, error) {
	switch tok.Type {
	case SYMBOL:
		return p.parseLiteralSymbol(tok)
	case STRING:
		return p.parseLiteralString(tok)
	case NUMBER:
		return p.parseLiteralNumber(tok)
	case OPEN_BRACKET:
		return p.parseLiteralArray()
	case OPEN_BRACE:
		return p.parseLiteralObject()
	default:
		return nil, p.SyntaxError()
	}
}

func (p *Parser) parseLiteralSymbol(tok *Token) (interface{}, error) {
	switch tok.Text {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null":
		return nil, nil
	default:
		return nil, fmt.Errorf("Not a valid symbol: %s", tok.Text)
	}
}
func (p *Parser) parseLiteralString(tok *Token) (*string, error) {
	s := "\"" + tok.Text + "\""
	q, err := strconv.Unquote(s)
	if err != nil {
		return nil, p.Error("Improperly escaped string: " + tok.Text)
	}
	return &q, nil
}

func (p *Parser) parseLiteralNumber(tok *Token) (interface{}, error) {
	num, err := ParseDecimal(tok.Text)
	if err != nil {
		return nil, p.Error(fmt.Sprintf("Not a valid number: %s", tok.Text))
	}
	return num, nil
}

func (p *Parser) parseLiteralArray() (interface{}, error) {
	var ary []interface{}
	for {
		tok := p.GetToken()
		if tok == nil {
			return nil, p.EndOfFileError()
		}
		if tok.Type == CLOSE_BRACKET {
			return ary, nil
		}
		if tok.Type != COMMA {
			obj, err := p.parseLiteral(tok)
			if err != nil {
				return nil, err
			}
			ary = append(ary, obj)
		}
	}
}

func (p *Parser) parseLiteralObject() (interface{}, error) {
	//either a map or a struct, i.e. a JSON object
	obj := make(map[string]interface{}, 0)
	for {
		tok := p.GetToken()
		if tok == nil {
			return nil, p.EndOfFileError()
		}
		if tok.Type == CLOSE_BRACE {
			return obj, nil
		}
		if tok.Type == STRING {
			pkey, err := p.parseLiteralString(tok)
			if err != nil {
				return nil, err
			}
			err = p.expect(COLON)
			if err != nil {
				return nil, err
			}
			val, err := p.parseLiteralValue()
			if err != nil {
				return nil, err
			}
			obj[*pkey] = val
		} else if tok.Type == SYMBOL {
			return nil, p.Error("Expected String key for JSON object, found symbol '" + tok.Text + "'")
		} else {
			//fmt.Println("ignoring this token:", tok)
		}
	}
}

func (p *Parser) arrayParams(params []string) (string, error) {
	var items string
	switch len(params) {
	case 0:
		items = "Any"
	case 1:
		items = params[0]
	default:
		return "", p.SyntaxError()
	}
	return items, nil
}

func (p *Parser) parseCollectionOptions(typedef *TypeDef) error {
	tok := p.GetToken()
	if tok == nil {
		return p.SyntaxError()
	}
	if tok.Type == OPEN_PAREN {
		for {
			tok := p.GetToken()
			if tok == nil {
				return p.SyntaxError()
			}
			if tok.Type == CLOSE_PAREN {
				return nil
			}
			if tok.Type == SYMBOL {
				switch tok.Text {
				case "minsize":
					num, err := p.expectEqualsInt64()
					if err != nil {
						return err
					}
					typedef.MinSize = num
				case "maxsize":
					num, err := p.expectEqualsInt64()
					if err != nil {
						return err
					}
					typedef.MaxSize = num
				}
			} else {
				return p.SyntaxError()
			}
		}
	} else {
		p.UngetToken()
		return nil
	}
}

func (p *Parser) mapParams(params []string) (string, string, error) {
	var keys string
	var items string
	switch len(params) {
	case 0:
		keys = "String"
		items = "Any"
	case 2:
		keys = params[0]
		items = params[1]
	default:
		return "", "", p.SyntaxError()
	}
	return keys, items, nil
}

func (p *Parser) unitValueParams(params []string) (string, string, error) {
	var value string
	var unit string
	var err error
	switch len(params) {
	case 0:
		value = "Decimal"
		unit = "String"
	case 2:
		value = params[0]
		unit = params[1]
	default:
		err = p.SyntaxError()
	}
	return value, unit, err
}

func (p *Parser) EndOfStatement(comment string) (string, error) {
	for {
		tok := p.GetToken()
		if tok == nil {
			return comment, nil
		}
		if tok.Type == SEMICOLON {
			//ignore it
		} else if tok.Type == LINE_COMMENT {
			comment = p.MergeComment(comment, tok.Text)
		} else if tok.Type == NEWLINE {
			return comment, nil
		} else {
			return comment, p.SyntaxError()
		}
	}
}

func (p *Parser) parseLeadingComment(comment string) string {
	for {
		tok := p.GetToken()
		if tok == nil {
			return comment
		}
		if tok.Type == LINE_COMMENT {
			comment = p.MergeComment(comment, tok.Text)
		} else {
			p.UngetToken()
			return comment
		}
	}
}

func (p *Parser) ParseTrailingComment(comment string) string {
	tok := p.GetToken()
	if tok != nil {
		if tok.Type == LINE_COMMENT {
			comment = p.MergeComment(comment, tok.Text)
		} else {
			p.UngetToken()
		}
	}
	return comment
}

func (p *Parser) MergeComment(comment1 string, comment2 string) string {
	if comment1 == "" {
		return strings.TrimSpace(comment2)
	}
	if comment2 == "" {
		return strings.TrimSpace(comment1)
	}
	return strings.TrimSpace(comment1) + " " + strings.TrimSpace(comment2)
}

func (p *Parser) Validate() (*Model, error) {
	var err error
	for _, td := range p.model.Types {
		switch td.Type {
		case "Struct":
			err = p.validateStruct(td)
		case "Array":
			err = p.validateArray(td)
		case "Map":
			err = p.validateMap(td)
		case "UnitValue":
			err = p.validateUnitValue(td)
		case "String":
			err = p.validateStringDef(td)
		case "UUID":
			err = p.validateReference(td)
		}
		if err != nil {
			return nil, err
		}
	}
	for _, ex := range p.model.Examples {
		err = p.validateExample(ex)
		if err != nil {
			return nil, err
		}
	}
	for _, action := range p.model.Actions {
		err = p.validateAction(action)
		if err != nil {
			return nil, err
		}
	}
	for _, hdef := range p.model.Http {
		err = p.validateHttp(hdef)
		if err != nil {
			return nil, err
		}
	}
	for _, ext := range p.extensions {
		err = ext.Validate(p)
		if err != nil {
			return nil, err
		}
	}
	return p.model, err
}

func (p *Parser) validateExample(ex *ExampleDef) error {
	lst := strings.Split(ex.Target, ".")
	theType := lst[0]
	t := p.model.FindType(theType)
	if t == nil {
		return fmt.Errorf("Undefined type '%s' in example: %s", theType, Pretty(ex))
	}
	if len(lst) > 1 {
		fmt.Println("warning: validation against arbitrary example target NYI:", ex.Example)
		return nil
	}
	return p.model.ValidateAgainstTypeSpec("example for "+ex.Target, &t.TypeSpec, ex.Example)
}

func (p *Parser) validateHttpPathTemplate(path string) error {
	i := strings.Index(path, "?")
	if i >= 0 {
		q := path[i+1:]
		path = path[:i]
		fmt.Printf("TODO: validate queryparams: %q from path template %q\n", q, path)
	}
	//check pathparams
	inParam := false
	for i, ch := range path {
		switch ch {
		case '{':
			if inParam {
				return fmt.Errorf("Bad http path template syntax (unexpected '{' at %d): %q", i, path)
			}
			inParam = true
		case '}':
			if !inParam {
				return fmt.Errorf("Bad http path template syntax (unexpected '}' at %d): %q", i, path)
			}
			inParam = false
		case '/':
			if inParam {
				return fmt.Errorf("Bad http path template syntax (variable cannot span elements at %d): %q", i, path)
			}
		}
	}
	if inParam {
		return fmt.Errorf("Bad http path template syntax (missing '}'): %q", path)
	}
	return nil
}

func (p *Parser) validateHttp(hact *HttpDef) error {
	err := p.validateHttpPathTemplate(hact.Path)
	if err != nil {
		return err
	}
	needsBody := hact.Method == "POST" || hact.Method == "PUT"
	bodyParam := ""
	for _, in := range hact.Inputs {
		//paramType, paramName := p.parameterSource(hact.Path, in.Name)
		if !in.Path && in.Query == "" && in.Header == "" {
			if needsBody {
				if bodyParam != "" {
					return fmt.Errorf("HTTP action cannot have more than one body parameter (%q is already that parameter): %s", bodyParam, Pretty(hact))
				}
				bodyParam = in.Name
			} else {
				return fmt.Errorf("Input parameter %q to HTTP action is not a header or a variable in the path: %s - %q", in.Name, Pretty(hact), hact.Method+" "+hact.Path)
			}
		}
	}
	return nil
}

func (p *Parser) validateAction(action *ActionDef) error {
	if action.Input != "" {
		t := p.model.FindType(action.Input)
		if t == nil {
			return fmt.Errorf("Action '%s' input type '%s' is not defined", action.Name, action.Input)
		}
	}
	if action.Output != "" {
		t := p.model.FindType(action.Output)
		if t == nil {
			return fmt.Errorf("Action '%s' output type '%s' is not defined", action.Name, action.Output)
		}
	}
	for _, etype := range action.Exceptions {
		t := p.model.FindType(etype)
		if t == nil {
			return fmt.Errorf("Action '%s' exception type '%s' is not defined", action.Name, etype)
		}
	}
	return nil
}

func (p *Parser) validateStringDef(td *TypeDef) error {
	if td.Pattern != "" {
		if td.Values != nil {
			return fmt.Errorf("Both 'pattern' and 'values' options cannot coexist in String type %s", td.Name)
		}
		//expand embedded references
		for {
			i := strings.Index(td.Pattern, "{")
			if i >= 0 {
				j := strings.Index(td.Pattern[i:], "}")
				if j > 0 {
					name := td.Pattern[i+1 : i+j]
					tpat := p.model.FindType(name)
					if tpat != nil {
						if tpat.Type == "String" {
							if tpat.Pattern != "" {
								td.Pattern = td.Pattern[:i] + tpat.Pattern + td.Pattern[i+j+1:]
							} else {
								return fmt.Errorf("Embedded pattern refers to string type %s, which has no pattern: %q", name, td.Pattern)
							}
						} else {
							return fmt.Errorf("Embedded pattern refers to non-string type %s", name)
						}
					} else {
						return fmt.Errorf("Embedded pattern refers to undefined type %s: %q", name, td.Pattern)
					}
				} else {
					break //unmatched {}, let the leading { just go through
				}
			} else {
				break
			}
		}
	}
	return p.validateReference(td)
}

func (p *Parser) validateReference(td *TypeDef) error {
	if td.Reference != "" {
		t := p.model.FindType(td.Reference)
		if t == nil {
			return fmt.Errorf("Undefined type '%s' for %s reference", td.Reference, td.Name)
		}
	}
	return nil
}

func (p *Parser) validateUnitValue(td *TypeDef) error {
	vt := p.model.FindType(td.Value)
	if vt == nil {
		return fmt.Errorf("Undefined value type '%s' for %s UnitValue type", td.Value, td.Name)
	}
	if !p.model.IsNumericType(&vt.TypeSpec) {
		return fmt.Errorf("UnitValue value type of %s is not numeric: %s", td.Name, vt.Name)
	}
	ut := p.model.FindType(td.Unit)
	if ut == nil {
		return fmt.Errorf("Undefined unit type '%s' for %s UnitValue unit", td.Unit, td.Name)
	}
	if ut.Type != "String" && ut.Type != "Enum" {
		return fmt.Errorf("UnitValue value type of %s is not String or Enum: %s", td.Name, vt.Name)
	}
	return nil
}

func (p *Parser) validateStruct(td *TypeDef) error {
	model := p.model
	for _, field := range td.Fields {
		ftd := model.FindType(field.Type)
		if ftd == nil {
			return fmt.Errorf("Undefined type '%s' in struct field '%s.%s'", field.Type, td.Name, field.Name)
		}
		if field.Default != nil {
			if field.Required {
				return fmt.Errorf("Cannot have a default value for required field: '%s.%s'", td.Name, field.Name)
			}
			err := model.ValidateAgainstTypeSpec(field.Type, &field.TypeSpec, field.Default)
			if err != nil {
				return err
			}
		}
		if field.Values != nil && field.Pattern != "" {
			return fmt.Errorf("Cannot have both 'values' and 'pattern' constraints in one string field: '%s.%s'", td.Name, field.Name)
		}
	}
	return nil
}

func (p *Parser) validateArray(td *TypeDef) error {
	model := p.model
	if td.Items == "Any" {
		return nil
	}
	itd := model.FindType(td.Items)
	if itd == nil {
		return fmt.Errorf("Undefined type '%s' for Array items '%s'", td.Items, td.Name)
	}
	return nil
}

func (p *Parser) validateMap(td *TypeDef) error {
	model := p.model
	if td.Items == "Any" {
		return nil
	}
	itd := model.FindType(td.Items)
	if itd == nil {
		return fmt.Errorf("Undefined type '%s' for Map items '%s'", td.Items, td.Name)
	}
	if td.Keys == "String" {
		return nil
	}
	ktd := model.FindType(td.Keys)
	if ktd == nil {
		return fmt.Errorf("Undefined type '%s' for Map keys '%s'", td.Keys, td.Name)
	}
	return nil
}

func (p *Parser) registerExtension(handler Extension) error {
	name := handler.Name()
	if p.extensions == nil {
		p.extensions = make(map[string]Extension, 0)
	}
	if _, ok := p.extensions[name]; ok {
		return fmt.Errorf("Extension already exists: %s", name)
	}
	p.extensions[name] = handler
	return nil
}

func (p *Parser) expectedDirectiveError() error {
	msg := "Expected one of 'type', 'name', 'version', 'base', "
	if p.extensions != nil {
		for k, _ := range p.extensions {
			msg = msg + fmt.Sprintf("'%s', ", k)
		}
	}
	msg = msg + " or an 'x_*' style extended annotation"
	return p.Error(msg)
}

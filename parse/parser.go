package parse

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/boynton/sadl"
)

//
// import "github.com/boynton/sadl/parse"
// ...
// model, err :- parse.File("/some/path")
//
func File(path string) (*sadl.Model, error) {
	return parseFile(path)
}

//
// import "github.com/boynton/sadl/parse"
// ...
// model, err :- parse.String("...")
//
func String(src string) (*sadl.Model, error) {
	return parseString(src)
}

//----------------

type Parser struct {
	path           string
	source         string
	scanner        *Scanner
	model          *sadl.Model
	schema         *sadl.Schema
	lastToken      *Token
	prevLastToken  *Token
	ungottenToken  *Token
	currentComment string
	extensions     map[string]ExtensionHandler
}

func parseFile(path string) (*sadl.Model, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	src := string(b)
	parser := &Parser{
		scanner: NewScanner(strings.NewReader(src)),
		path:    path,
		source:  src,
	}
	return parser.Parse()
}

func parseString(src string) (*sadl.Model, error) {
	parser := &Parser{
		scanner: NewScanner(strings.NewReader(src)),
		source:  src,
	}
	return parser.Parse()
}

func (p *Parser) ungetToken() {
	Debug("ungetToken() -> ", p.lastToken)
	p.ungottenToken = p.lastToken
	p.lastToken = p.prevLastToken
}

func (p *Parser) getToken() *Token {
	if p.ungottenToken != nil {
		p.lastToken = p.ungottenToken
		p.ungottenToken = nil
		Debug("getToken() -> ", p.lastToken)
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
	Debug("getToken() -> ", p.lastToken)
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

func (p *Parser) Parse() (*sadl.Model, error) {
	p.schema = &sadl.Schema{
		Types: make([]*sadl.TypeDef, 0),
	}
	if p.schema.Name == "" {
		p.schema.Name = BaseFileName(p.path)
	}
	comment := ""
	for {
		var err error
		tok := p.getToken()
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
			case "http":
				err = p.parseHttpDirective(comment)
			default:
				err = p.parseExtensionDirective(comment, tok.Text)
			}
			comment = ""
		case LINE_COMMENT:
			comment = p.mergeComment(comment, tok.Text)
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
	p.model, err = sadl.NewModel(p.schema)
	p.schema = nil
	if err != nil {
		return nil, err
	}
	return p.Validate()
}

func (p *Parser) parseNameDirective(comment string) error {
	p.schema.Comment = p.mergeComment(p.schema.Comment, comment)
	txt, err := p.expectText()
	if err == nil {
		p.schema.Name = txt
	}
	return err
}

func (p *Parser) parseVersionDirective(comment string) error {
	p.schema.Comment = p.mergeComment(p.schema.Comment, comment)
	tok := p.getToken()
	if tok == nil {
		return p.endOfFileError()
	}
	switch tok.Type {
	case NUMBER, SYMBOL, STRING:
		p.schema.Version = tok.Text
		return nil
	default:
		return p.Error("Bad version value: " + tok.Text)
	}
}

func (p *Parser) parseHttpDirective(comment string) error {
	sym, err := p.expectIdentifier()
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
	pathTemplate, err := p.expectString()
	if err != nil {
		return err
	}
	options, err := p.parseOptions("http", []string{"operation"})
	if err != nil {
		return err
	}
	op := &sadl.HttpDef{
		Method:      method,
		Path:        pathTemplate,
		Name:        options.Operation,
		Annotations: options.Annotations,
	}
	tok := p.getToken()
	if tok == nil {
		return p.endOfFileError()
	}
	if tok.Type == OPEN_BRACE {
		comment = p.parseTrailingComment(comment)
		for {
			done, comment2, err := p.isBlockDone(comment)
			if done {
				comment = comment2
				break
			}
			in, out, err := p.parseHttpSpec(pathTemplate, true)
			if err != nil {
				return err
			}
			if in != nil {
				op.Inputs = append(op.Inputs, in)
			} else if out != nil {
				if out.Error {
					op.Errors = append(op.Errors, out)
				} else {
					op.Output = out
				}
			} else {
				break
			}
		}
		comment, err = p.endOfStatement(comment)
		op.Comment = comment
		p.schema.Operations = append(p.schema.Operations, op)
	} else {
		return p.syntaxError()
	}
	return nil
}

func (p *Parser) parseHttpSpec(pathTemplate string, top bool) (*sadl.HttpParamSpec, *sadl.HttpResponseSpec, error) {
	ename, err := p.expectIdentifier()
	if err != nil {
		return nil, nil, err
	}
	if ename == "expect" || ename == "except" {
		if !top {
			return nil, nil, p.syntaxError()
		}
		output, err := p.parseHttpResponseSpec(ename)
		return nil, output, err
	}
	etype, err := p.expectIdentifier()
	if err != nil {
		return nil, nil, err
	}
	options, err := p.parseOptions("HttpParam", []string{"header", "default"})
	if err != nil {
		return nil, nil, err
	}
	spec := &sadl.HttpParamSpec{
		Name:        ename,
		Type:        etype,
		Annotations: options.Annotations,
	}
	if options.Default != "" {
		spec.Default = options.Default
	}
	if options.Header != "" {
		spec.Header = options.Header
	} else if top {
		paramType, paramName := p.parameterSource(pathTemplate, ename)
		switch paramType {
		case "path":
			spec.Path = true
		case "query":
			spec.Query = paramName
		default:
			//must be the body. Should I require the name "body"?
		}
	}
	return spec, nil, err
}

func (p *Parser) parseHttpResponseSpec(ename string) (*sadl.HttpResponseSpec, error) {
	estatus, err := p.expectInt32()
	if err != nil {
		return nil, err
	}
	options, err := p.parseOptions("HttpResponse", []string{})
	if err != nil {
		return nil, err
	}
	output := &sadl.HttpResponseSpec{
		Status:      estatus,
		Annotations: options.Annotations,
		Error: ename != "expect",
	}
	comment := ""
   tok := p.getToken()
   if tok == nil {
      return nil, p.endOfFileError()
   }
	if tok.Type == OPEN_BRACE {
		comment = p.parseTrailingComment(comment)
		for {
			done, comment2, err := p.isBlockDone(comment)
			if done {
				comment = comment2
				break
			}
			out, _, err := p.parseHttpSpec("", false)
			if err != nil {
				return nil, err
			}
			output.Outputs = append(output.Outputs, out)
		}
	} else {
		p.ungetToken()
		output.Comment, err = p.endOfStatement(comment)
	}
	return output, nil
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
	for _, qparam := range strings.Split(query, "&") {
		kv := strings.Split(qparam, "=")
		if len(kv) > 1 && kv[1] == match {
			return "query", kv[0]
		}
	}
	return "", ""
}

func (p *Parser) isBlockDone(comment string) (bool, string, error) {
	tok := p.getToken()
	if tok == nil {
		return false, comment, p.endOfFileError()
	}
	for {
		if tok.Type == CLOSE_BRACE {
			return true, comment, nil
		} else if tok.Type == LINE_COMMENT {
			comment = p.mergeComment("", tok.Text)
			tok = p.getToken()
			if tok == nil {
				return false, comment, p.endOfFileError()
			}
		} else if tok.Type == NEWLINE {
			tok = p.getToken()
			if tok == nil {
				return false, comment, p.endOfFileError()
			}
		} else {
			p.ungetToken()
			return false, comment, nil
		}
	}
}

func (p *Parser) parseTypeDirective(comment string) error {
	typeName, err := p.expectIdentifier()
	if err != nil {
		return err
	}
	superName, params, fields, elements, options, comment2, err := p.parseTypeSpec() //note that this can return user-defined types
	if err != nil {
		return err
	}
	comment = p.mergeComment(comment, comment2)
	td := &sadl.TypeDef{
		TypeSpec: sadl.TypeSpec{
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
	case "Quantity":
		err = p.parseQuantityDef(td, params)
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

func (p *Parser) parseTypeSpec() (string, []string, []*sadl.StructFieldDef, []*sadl.EnumElementDef, *Options, string, error) {
	options := &Options{}
	typeName, err := p.expectIdentifier()
	if err != nil {
		return "", nil, nil, nil, options, "", err
	}
	tok := p.getToken()
	if tok == nil {
		return typeName, nil, nil, nil, options, "", nil
	}
	if tok.Type == OPEN_ANGLE {
		var params []string
		var expectedParams int
		switch typeName {
		case "Array":
			expectedParams = 1
		case "Map", "Quantity":
			expectedParams = 2
		case "Union":
			expectedParams = -1
		default:
			return typeName, nil, nil, nil, options, "", p.syntaxError()
		}
		for {
			tok = p.getToken()
			if tok == nil {
				return typeName, nil, nil, nil, options, "", p.endOfFileError()
			}
			if tok.Type != COMMA {
				if tok.Type == CLOSE_ANGLE {
					if expectedParams >= 0 && expectedParams != len(params) {
						return typeName, nil, nil, nil, options, "", p.syntaxError()
					}
					return typeName, params, nil, nil, options, "", nil
				}
				if tok.Type != SYMBOL {
					return typeName, params, nil, nil, options, "", p.syntaxError()
				}
				params = append(params, tok.Text)
			}
		}
	} else if typeName == "Struct" || typeName == "Enum" {
		if tok.Type != OPEN_BRACE {
			p.ungetToken()
			options, err = p.parseOptions(typeName, []string{})
			if err != nil {
				return typeName, nil, nil, nil, options, "", err
			}
			tok = p.getToken()
			if tok == nil {
				return typeName, nil, nil, nil, options, "", p.endOfFileError()
			}
		}
		if tok.Type == OPEN_BRACE {
			comment := p.parseTrailingComment("")
			switch typeName {
			case "Struct":
				var fields []*sadl.StructFieldDef
				tok := p.getToken()
				if tok == nil {
					return typeName, nil, fields, nil, options, comment, p.syntaxError()
				}
				if tok.Type != NEWLINE {
					p.ungetToken()
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
				comment, err = p.endOfStatement(comment)
				p.ungetToken() //the closing brace
				return typeName, nil, fields, nil, options, comment, nil
			case "Enum":
				var elements []*sadl.EnumElementDef
				tok := p.getToken()
				if tok == nil {
					return typeName, nil, nil, nil, options, comment, p.syntaxError()
				}
				if tok.Type != NEWLINE {
					p.ungetToken()
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
				comment, err = p.endOfStatement(comment)
				p.ungetToken() //the closing brace
				return typeName, nil, nil, elements, options, comment, nil
			}
			return typeName, nil, nil, nil, options, comment, p.syntaxError()
		}
	}
	p.ungetToken()
	return typeName, nil, nil, nil, options, "", nil
}

func (p *Parser) parseAnyDef(td *sadl.TypeDef) error {
	err := p.parseTypeOptions(td)
	if err == nil {
		td.Comment, err = p.endOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseBoolDef(td *sadl.TypeDef) error {
	err := p.parseTypeOptions(td)
	if err == nil {
		td.Comment, err = p.endOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseNumberDef(td *sadl.TypeDef) error {
	err := p.parseTypeOptions(td, "min", "max")
	if err == nil {
		td.Comment, err = p.endOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseBytesDef(td *sadl.TypeDef) error {
	err := p.parseTypeOptions(td, "minsize", "maxsize")
	if err == nil {
		td.Comment, err = p.endOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseStringDef(td *sadl.TypeDef) error {
	err := p.parseTypeOptions(td, "minsize", "maxsize", "pattern", "values")
	if err == nil {
		td.Comment, err = p.endOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseTimestampDef(td *sadl.TypeDef) error {
	err := p.parseTypeOptions(td)
	if err == nil {
		td.Comment, err = p.endOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseUUIDDef(td *sadl.TypeDef) error {
	err := p.parseTypeOptions(td)
	if err == nil {
		td.Comment, err = p.endOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseQuantityDef(td *sadl.TypeDef, params []string) error {
	err := p.parseTypeOptions(td)
	if err == nil {
		td.Value, td.Unit, err = p.quantityParams(params)
		if err == nil {
			td.Comment, err = p.endOfStatement(td.Comment)
		}
	}
	return err
}

func (p *Parser) parseArrayDef(td *sadl.TypeDef, params []string) error {
	var err error
	td.Items, err = p.arrayParams(params)
	if err == nil {
		err = p.parseTypeOptions(td, "minsize", "maxsize")
		if err == nil {
			td.Comment, err = p.endOfStatement(td.Comment)
		}
	}
	return err
}

func (p *Parser) parseMapDef(td *sadl.TypeDef, params []string) error {
	var err error
	td.Keys, td.Items, err = p.mapParams(params)
	if err == nil {
		err = p.parseTypeOptions(td, "minsize", "maxsize")
		if err == nil {
			td.Comment, err = p.endOfStatement(td.Comment)
		}
	}
	return err
}

func (p *Parser) parseStructDef(td *sadl.TypeDef, fields []*sadl.StructFieldDef) error {
	err := p.parseTypeOptions(td)
	if err == nil {
		td.Fields = fields
	}
	return err
}

func (p *Parser) parseEnumDef(td *sadl.TypeDef, elements []*sadl.EnumElementDef) error {
	td.Elements = elements
	return nil
}

func (p *Parser) parseUnionDef(td *sadl.TypeDef, params []string) error {
	err := p.parseTypeOptions(td)
	if err == nil {
		td.Variants = params
		td.Comment, err = p.endOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) Error(msg string) error {
	Debug("*** error, last token:", p.lastToken)
	return fmt.Errorf("*** %s\n", formattedAnnotation(p.path, p.Source(), "", msg, p.lastToken, RED, 5))
}

func (p *Parser) syntaxError() error {
	return p.Error("Syntax error")
}

func (p *Parser) endOfFileError() error {
	return p.Error("Unexpected end of file")
}

func (p *Parser) assertIdentifier(tok *Token) (string, error) {
	if tok == nil {
		return "", p.endOfFileError()
	}
	if tok.Type == SYMBOL {
		return tok.Text, nil
	}
	return tok.Text, p.Error(fmt.Sprintf("Expected symbol, found %v", tok.Type))
}

func (p *Parser) expectIdentifier() (string, error) {
	tok := p.getToken()
	return p.assertIdentifier(tok)
}

func (p *Parser) expectEqualsIdentifier() (string, error) {
	err := p.expect(EQUALS)
	if err != nil {
		return "", err
	}
	return p.expectIdentifier()
}

func (p *Parser) assertString(tok *Token) (string, error) {
	if tok == nil {
		return "", p.endOfFileError()
	}
	if tok.Type == STRING {
		return tok.Text, nil
	}
	return tok.Text, p.Error(fmt.Sprintf("Expected string, found %v", tok.Type))
}

func (p *Parser) expectString() (string, error) {
	tok := p.getToken()
	return p.assertString(tok)
}

func (p *Parser) expectEqualsString() (string, error) {
	err := p.expect(EQUALS)
	if err != nil {
		return "", err
	}
	return p.expectString()
}

func (p *Parser) expectText() (string, error) {
	tok := p.getToken()
	if tok == nil {
		return "", fmt.Errorf("Unexpected end of file")
	}
	if tok.IsText() {
		return tok.Text, nil
	}
	return "", fmt.Errorf("Expected symbol or string, found %v", tok.Type)
}

func (p *Parser) expectInt32() (int32, error) {
	tok := p.getToken()
	if tok == nil {
		return 0, p.endOfFileError()
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

func (p *Parser) expectNumber() (*sadl.Decimal, error) {
	tok := p.getToken()
	if tok == nil {
		return nil, p.endOfFileError()
	}
	if tok.IsNumeric() {
		return sadl.ParseDecimal(tok.Text)
	}
	return nil, p.Error(fmt.Sprintf("Expected number, found %v", tok.Type))
}

func (p *Parser) expectEqualsNumber() (*sadl.Decimal, error) {
	err := p.expect(EQUALS)
	if err != nil {
		return nil, err
	}
	return p.expectNumber()
}

func (p *Parser) expect(toktype TokenType) error {
	tok := p.getToken()
	if tok == nil {
		return p.endOfFileError()
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

func (p *Parser) parseTypeOptions(td *sadl.TypeDef, acceptable ...string) error {
	options, err := p.parseOptions(td.Type, acceptable)
	if err == nil {
		td.Pattern = options.Pattern
		td.Values = options.Values
		td.MinSize = options.MinSize
		td.MaxSize = options.MaxSize
		td.Min = options.Min
		td.Max = options.Max
		td.Annotations = options.Annotations
	}
	return err
}

type Options struct {
	Required    bool
	Default     interface{}
	Pattern     string
	Values      []string
	MinSize     *int32
	MaxSize     *int32
	Min         *sadl.Decimal
	Max         *sadl.Decimal
	Operation   string
	Header      string
	Annotations map[string]string
}

func (p *Parser) parseOptions(typeName string, acceptable []string) (*Options, error) {
	options := &Options{}
	var err error
	tok := p.getToken()
	if tok == nil {
		return options, nil
	}
	if tok.Type == OPEN_PAREN {
		for {
			tok := p.getToken()
			if tok == nil {
				return nil, p.syntaxError()
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
						options.MinSize, err = p.expectEqualsInt32()
					case "maxsize":
						options.MaxSize, err = p.expectEqualsInt32()
					case "pattern":
						options.Pattern, err = p.expectEqualsString()
					case "values":
						options.Values, err = p.expectEqualsStringArray()
					case "required":
						options.Required = true
					case "default":
						options.Default, err = p.parseEqualsLiteral()
					case "operation":
						options.Operation, err = p.expectEqualsIdentifier()
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
				return nil, p.syntaxError()
			}
		}
	} else {
		p.ungetToken()
		return options, nil
	}
}

func (p *Parser) parseExtendedOption(annos map[string]string, anno string) (map[string]string, error) {
	var err error
	var val string
	tok := p.getToken()
	if tok != nil {
		if tok.Type == EQUALS {
			val, err = p.expectString()
		} else {
			p.ungetToken()
		}
	} else {
		err = p.endOfFileError()
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

func (p *Parser) parseBytesOptions(typedef *sadl.TypeDef) error {
	tok := p.getToken()
	if tok == nil {
		return p.syntaxError()
	}
	expected := ""
	if tok.Type == OPEN_PAREN {
		for {
			tok := p.getToken()
			if tok == nil {
				return p.syntaxError()
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
					return p.syntaxError()
				}
			} else if tok.Type == NUMBER {
				if expected == "" {
					return p.syntaxError()
				}
				val, err := sadl.ParseDecimal(tok.Text)
				if err != nil {
					return err
				}
				if expected == "minsize" {
					i := val.AsInt32()
					typedef.MinSize = &i
				} else if expected == "maxsize" {
					i := val.AsInt32()
					typedef.MinSize = &i
				} else {
					return p.Error("bytes option must have numeric value")
				}
				expected = ""
			}
		}
	} else {
		p.ungetToken()
		return nil
	}
}

func (p *Parser) expectEqualsStringArray() ([]string, error) {
	var values []string
	tok := p.getToken()
	if tok == nil {
		return nil, p.endOfFileError()
	}
	if tok.Type != EQUALS {
		return nil, p.syntaxError()
	}

	tok = p.getToken()
	if tok == nil {
		return nil, p.endOfFileError()
	}
	if tok.Type != OPEN_BRACKET {
		return nil, p.syntaxError()
	}
	for {
		tok = p.getToken()
		if tok == nil {
			return nil, p.endOfFileError()
		}
		if tok.Type == CLOSE_BRACKET {
			break
		}
		if tok.Type == STRING {
			values = append(values, tok.Text)
		} else if tok.Type == COMMA {
			//ignore
		} else {
			return nil, p.syntaxError()
		}
	}
	return values, nil
}

func (p *Parser) parseEnumElementDef() (*sadl.EnumElementDef, error) {
	comment := ""
	sym := ""
	var err error
	for {
		tok := p.getToken()
		if tok == nil {
			return nil, p.endOfFileError()
		}
		if tok.Type == CLOSE_BRACE {
			return nil, nil
		} else if tok.Type == LINE_COMMENT {
			comment = p.mergeComment(comment, tok.Text)
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
	options, err := p.parseOptions("Enum", []string{})
	if err != nil {
		return nil, err
	}
	comment = p.parseTrailingComment(comment)
	return &sadl.EnumElementDef{
		Symbol:      sym,
		Comment:     comment,
		Annotations: options.Annotations,
	}, nil
}

func (p *Parser) expectNewline() error {
	tok := p.getToken()
	if tok == nil {
		return p.syntaxError()
	}
	if tok.Type != NEWLINE {
		p.ungetToken()
		return p.syntaxError()
	}
	return nil
}

func (p *Parser) parseStructFieldDef() (*sadl.StructFieldDef, error) {
	var comment string
	tok := p.getToken()
	if tok == nil {
		return nil, p.endOfFileError()
	}
	for {
		if tok.Type == CLOSE_BRACE {
			return nil, nil
		} else if tok.Type == LINE_COMMENT {
			comment = p.mergeComment("", tok.Text)
			tok = p.getToken()
			if tok == nil {
				return nil, p.endOfFileError()
			}
		} else if tok.Type == NEWLINE {
			tok = p.getToken()
			if tok == nil {
				return nil, p.endOfFileError()
			}
		} else {
			p.ungetToken()
			break
		}
	}
	fname, err := p.expectIdentifier()
	if err != nil {
		return nil, err
	}
	ftype, fparams, ffields, felements, foptions, fcomment, err := p.parseTypeSpec()
	if err != nil {
		return nil, err
	}
	comment = p.mergeComment(comment, fcomment)

	var fkeys, fitems string
	var funit, fvalue string
	switch ftype {
	case "Array":
		fitems, err = p.arrayParams(fparams)
	case "Map":
		fkeys, fitems, err = p.mapParams(fparams)
	case "Quantity":
		fvalue, funit, err = p.quantityParams(fparams)
	default:
		if len(fparams) != 0 {
			//unions!
			return nil, p.syntaxError()
		}
	}
	if err != nil {
		return nil, err
	}
	field := &sadl.StructFieldDef{
		Name:    fname,
		Comment: comment,
		TypeSpec: sadl.TypeSpec{
			Type:     ftype,
			Items:    fitems,
			Keys:     fkeys,
			Value:    fvalue,
			Unit:     funit,
			Fields:   ffields,
			Elements: felements,
		},
	}
	err = p.parseStructFieldOptions(field)
	if err == nil {
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
		field.Comment, err = p.endOfStatement(field.Comment)
	}
	return field, nil
}

func (p *Parser) parseStructFieldOptions(field *sadl.StructFieldDef) error {
	var acceptable []string
	switch field.Type {
	case "String":
		acceptable = []string{"pattern", "values", "minsize", "maxsize"}
	case "Int8", "Int16", "Int32", "Int64", "Float32", "Float64", "Decimal":
		acceptable = []string{"min", "max"}
	case "Bytes", "Array", "Map":
		acceptable = []string{"minsize", "maxsize"}
	}
	acceptable = append(acceptable, "required")
	acceptable = append(acceptable, "default")
	options, err := p.parseOptions(field.Type, acceptable)
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
	tok := p.getToken()
	if tok == nil {
		return nil, p.syntaxError()
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
		return nil, p.syntaxError()
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
		return tok.Text, nil
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
	num, err := sadl.ParseDecimal(tok.Text)
	if err != nil {
		return nil, p.Error(fmt.Sprintf("Not a valid number: %s", tok.Text))
	}
	return num, nil
}

func (p *Parser) parseLiteralArray() (interface{}, error) {
	var ary []interface{}
	for {
		tok := p.getToken()
		if tok == nil {
			return nil, p.endOfFileError()
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
		tok := p.getToken()
		if tok == nil {
			return nil, p.endOfFileError()
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
		return "", p.syntaxError()
	}
	return items, nil
}

func (p *Parser) parseCollectionOptions(typedef *sadl.TypeDef) error {
	tok := p.getToken()
	if tok == nil {
		return p.syntaxError()
	}
	if tok.Type == OPEN_PAREN {
		for {
			tok := p.getToken()
			if tok == nil {
				return p.syntaxError()
			}
			if tok.Type == CLOSE_PAREN {
				return nil
			}
			if tok.Type == SYMBOL {
				switch tok.Text {
				case "minsize":
					num, err := p.expectEqualsInt32()
					if err != nil {
						return err
					}
					typedef.MinSize = num
				case "maxsize":
					num, err := p.expectEqualsInt32()
					if err != nil {
						return err
					}
					typedef.MaxSize = num
				}
			} else {
				return p.syntaxError()
			}
		}
	} else {
		p.ungetToken()
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
		return "", "", p.syntaxError()
	}
	return keys, items, nil
}

func (p *Parser) quantityParams(params []string) (string, string, error) {
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
		err = p.syntaxError()
	}
	return value, unit, err
}

func (p *Parser) endOfStatement(comment string) (string, error) {
	for {
		tok := p.getToken()
		if tok == nil {
			return comment, nil
		}
		if tok.Type == SEMICOLON {
			//ignore it
		} else if tok.Type == LINE_COMMENT {
			comment = p.mergeComment(comment, tok.Text)
		} else if tok.Type == NEWLINE {
			return comment, nil
		} else {
			return comment, p.syntaxError()
		}
	}
}

func (p *Parser) parseLeadingComment(comment string) string {
	for {
		tok := p.getToken()
		if tok == nil {
			return comment
		}
		if tok.Type == LINE_COMMENT {
			comment = p.mergeComment(comment, tok.Text)
		} else {
			p.ungetToken()
			return comment
		}
	}
}

func (p *Parser) parseTrailingComment(comment string) string {
	tok := p.getToken()
	if tok != nil && tok.Type == LINE_COMMENT {
		comment = p.mergeComment(comment, tok.Text)
	} else {
		p.ungetToken()
	}
	return comment
}

func (p *Parser) mergeComment(comment1 string, comment2 string) string {
	return strings.TrimSpace(comment1 + " " + comment2)
}

func (p *Parser) Validate() (*sadl.Model, error) {
	var err error
	for _, td := range p.model.Types {
		switch td.Type {
		case "Struct":
			err = p.validateStruct(td)
		case "Quantity":
			err = p.validateQuantity(td)
		default:
			//			fmt.Println("VALIDATE ME:", td)
		}
		if err != nil {
			return nil, err
		}
	}
	return p.model, err
}

func (p *Parser) validateQuantity(td *sadl.TypeDef) error {
	vt := p.model.FindType(td.Value)
	if vt == nil {
		return fmt.Errorf("Undefined type '%s' for %s quantity type", td.Value, td.Name)
	}
	if !p.model.IsNumericType(&vt.TypeSpec) {
		return fmt.Errorf("Quantity value type of %s is not numeric: %s", td.Name, vt.Name)
	}
	ut := p.model.FindType(td.Unit)
	if ut == nil {
		return fmt.Errorf("Undefined type '%s' for %s quantity unit", td.Unit, td.Name)
	}
	if ut.Type != "String" && ut.Type != "Enum" {
		return fmt.Errorf("Quantity value type of %s is not String or Enum: %s", td.Name, vt.Name)
	}
	return nil
}

func (p *Parser) validateStruct(td *sadl.TypeDef) error {
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
			err := model.ValidateAgainstTypeSpec(&field.TypeSpec, field.Default)
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

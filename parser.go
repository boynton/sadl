package sadl

import(
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	
	"github.com/boynton/sadl/scanner"
)

type Parser struct {
	path string
	source string
	tokens []*scanner.Token
	schema *Schema
	lastToken *scanner.Token
	ungottenToken *scanner.Token
	currentComment string
	extensions map[string]ExtensionHandler
}

func ParseFile(path string) (*Schema, error) {
	parser := &Parser{
		path: path,
		schema: &Schema{
			Types: make([]*TypeDef, 0),
		},
	}
	err := parser.ScanFile(path)
	if err != nil {
		return nil, err
	}
	err = parser.Parse()
	if err != nil {
		return nil, err
	}
	return parser.schema, nil
}

func ParseString(src string) (*Schema, error) {
	parser := &Parser{
		path: "",
		source: src,
		schema: &Schema{
			Types: make([]*TypeDef, 0),
		},
	}
	err := parser.ScanString(src)
	if err != nil {
		return nil, err
	}
	err = parser.Parse()
	if err != nil {
		return nil, err
	}
	return parser.schema, nil
}

func (p *Parser) Parse() error {
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
		case scanner.SYMBOL:
			switch tok.Text {
			case "name":
				err = p.parseNameDirective()
			case "namespace":
				err = p.parseNamespaceDirective()
			case "version":
				err = p.parseVersionDirective()
			case "type":
				err = p.parseTypeDirective(comment)
				comment = ""
			default:
				err = p.parseExtensionDirective(comment, tok.Text)
				comment = ""
			}
		case scanner.LINE_COMMENT:
			comment = p.mergeComment(comment, tok.Text)
		case scanner.SEMICOLON:
			/* ignore */
		case scanner.NEWLINE:
			/* ignore */
		default:
			return p.expectedDirectiveError()
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) parseNameDirective() error {
	txt, err := p.expectText()
	if err == nil {
		p.schema.Name = txt
	}
	return err
}

func (p *Parser) parseNamespaceDirective() error {
	txt, err := p.expectText()
	if err == nil {
		p.schema.Namespace = txt
	}
	return err
}

func (p *Parser) parseVersionDirective() error {
	tok := p.getToken()
	if tok == nil {
		return p.endOfFileError()
	}
	switch tok.Type {
	case scanner.NUMBER, scanner.SYMBOL, scanner.STRING:
		p.schema.Version = tok.Text
		return nil
	default:
		return p.Error("Bad version value: " + tok.Text)
	}
}

func (p *Parser) parseTypeDirective(comment string) error {
	typeName, err := p.expectIdentifier()
	if err != nil {
		return err
	}
	superName, params, err := p.parseTypeSpec()
	if err != nil {
		return err
	}
	td := &TypeDef{
		Name: typeName,
		Type: superName,
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
	case "Struct": //no inheritance, period. Use composition instead, just a lot clearer
		err = p.parseStructDef(td)
	case "Enum":
		err = p.parseEnumDef(td)
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

func (p *Parser) parseTypeSpec() (string, []string, error) {
	typeName, err := p.expectIdentifier()
	if err != nil {
		return "", nil, err
	}
	tok := p.getToken()
	if tok == nil {
		return typeName, nil, nil
	}
	if tok.Type == scanner.OPEN_ANGLE {
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
			return "", nil, p.syntaxError()
		}
		for {
			tok = p.getToken()
			if tok == nil {
				return typeName, nil, p.endOfFileError()
			}
			if tok.Type != scanner.COMMA {
				if tok.Type == scanner.CLOSE_ANGLE {
					if expectedParams >= 0 && expectedParams != len(params) {
						return "", nil, p.syntaxError()
					}
					return typeName, params, nil
				}
				if tok.Type != scanner.SYMBOL {
					return typeName, params, p.syntaxError()
				}
				params = append(params, tok.Text)
			}
		}
	}
	p.ungetToken()
	return typeName, nil, nil
}

func (p *Parser) parseAnyDef(td *TypeDef) error {
	err := p.parseTypeOptions(td)
	if err == nil {
		td.Comment, err = p.endOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseBoolDef(td *TypeDef) error {
	err := p.parseTypeOptions(td)
	if err == nil {
		td.Comment, err = p.endOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseNumberDef(td *TypeDef) error {
	err := p.parseTypeOptions(td, "min", "max")
	if err == nil {
		td.Comment, err = p.endOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseBytesDef(td *TypeDef) error {
	err := p.parseTypeOptions(td, "minsize", "maxsize")
	if err == nil {
		td.Comment, err = p.endOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseStringDef(td *TypeDef) error {
	err := p.parseTypeOptions(td, "minsize", "maxsize", "pattern", "values")
	if err == nil {
		td.Comment, err = p.endOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseTimestampDef(td *TypeDef) error {
	err := p.parseTypeOptions(td)
	if err == nil {
		td.Comment, err = p.endOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseUUIDDef(td *TypeDef) error {
	err := p.parseTypeOptions(td)
	if err == nil {
		td.Comment, err = p.endOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseQuantityDef(td *TypeDef, params []string) error {
	err := p.parseTypeOptions(td)
	if err == nil {
		td.Value, td.Unit, err = p.quantityParams(params)
		if err == nil {
			td.Comment, err = p.endOfStatement(td.Comment)
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
			td.Comment, err = p.endOfStatement(td.Comment)
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
			td.Comment, err = p.endOfStatement(td.Comment)
		}
	}
	return err
}

func (p *Parser) parseStructDef(td *TypeDef) error {
	err := p.parseTypeOptions(td)
	if err == nil {
		tok := p.getToken()
		if tok.Type == scanner.OPEN_BRACE {
			td.Comment = p.parseTrailingComment(td.Comment)
			tok := p.getToken()
			if tok == nil {
				return p.syntaxError()
			}
			if tok.Type != scanner.NEWLINE {
				p.ungetToken()
			}
			for {
				field, err := p.parseStructFieldDef()
				if err != nil {
					return err
				}
				if field == nil {
					break
				}
				td.Fields = append(td.Fields, field)
			}
			td.Comment, err = p.endOfStatement(td.Comment)
		} else {
			p.ungetToken()
			td.Comment, err = p.endOfStatement(td.Comment)
		}
	}
	return err
}


func (p *Parser) parseEnumDef(td *TypeDef) error {
	err := p.parseTypeOptions(td)
	if err == nil {
		tok := p.getToken()
		if tok.Type != scanner.OPEN_BRACE {
			return p.syntaxError()
		}
		td.Comment = p.parseTrailingComment(td.Comment)
		for {
			elem, err := p.parseEnumElementDef()
			if err != nil {
				return err
			}
			if elem == nil {
				break
			}
			td.Elements = append(td.Elements, elem)
		}
		td.Comment, err = p.endOfStatement(td.Comment)
	}
	return err
}

func (p *Parser) parseUnionDef(td *TypeDef, params []string) error {
	err := p.parseTypeOptions(td)
	if err == nil {
		td.Variants = params
		td.Comment, err = p.endOfStatement(td.Comment)
	}
	return err
}




func (p *Parser) ScanFile(path string) error {
	p.path = path
	fi, err := os.Open(p.path)
	if err != nil {
		return fmt.Errorf("Can't open file %q\n", p.path)
	}
	defer fi.Close()	
	reader := bufio.NewReader(fi)
	return p.Scan(reader)
}

func (p *Parser) ScanString(src string) error {
	p.path = ""
	reader := strings.NewReader(src)
	return p.Scan(reader)
}

func (p *Parser) Source() string {
	source := p.source
	if p.path != "" {
		data, err := ioutil.ReadFile(p.path)
		if err == nil {
			source = string(data)
		}
	}
	return source
}

func (p *Parser) Scan(reader io.Reader) error {
	scanr := scanner.New(p.path, reader)
	var tokens []*scanner.Token
	for {
		tok := scanr.Scan()
		if tok.Type == scanner.EOF {
			break
		}
		if tok.Type == scanner.ILLEGAL {
			return fmt.Errorf("*** %s\n", formattedAnnotation(p.path, p.Source(), "", "Syntax error", &tok, RED, 5))
			os.Exit(1)
		} else if tok.Type != scanner.BLOCK_COMMENT {
			tokens = append(tokens, &tok)
		}
	}
	p.tokens = tokens
	return nil;
}

func (p *Parser) Error(msg string) error {
	debug("error, last token:", p.lastToken)
	return fmt.Errorf("*** %s\n", formattedAnnotation(p.path, p.Source(), "", msg, p.lastToken, RED, 5))
}

func (p *Parser) syntaxError() error {
	return p.Error("Syntax error")
}

func (p *Parser) ungetToken() {
	debug("ungetToken() -> ", p.lastToken)
	p.ungottenToken = p.lastToken
	p.lastToken = nil
}

func (p *Parser) getToken() *scanner.Token {
	if p.ungottenToken != nil {
		p.lastToken =  p.ungottenToken
		p.ungottenToken = nil
		return p.lastToken
	}
	if len(p.tokens) == 0 {
		return nil
	}
	p.lastToken = p.tokens[0]
	p.tokens = p.tokens[1:]
	debug("getToken() -> ", p.lastToken)
	return p.lastToken
}

func (p *Parser) endOfFileError() error {
	return p.Error("Unexpected end of file")
}

func (p *Parser) assertIdentifier(tok *scanner.Token) (string, error) {
	if tok == nil {
		return "", p.endOfFileError()
	}
	if tok.Type == scanner.SYMBOL {
		return tok.Text, nil
	}
	return tok.Text, p.Error(fmt.Sprintf("Expected symbol, found %v", tok.Type))
}

func (p *Parser) expectIdentifier() (string, error) {
	tok := p.getToken()
	return p.assertIdentifier(tok)
}

func (p *Parser) assertString(tok *scanner.Token) (string, error) {
	if tok == nil {
		return "", p.endOfFileError()
	}
	if tok.Type == scanner.STRING {
		return tok.Text, nil
	}
	return tok.Text, p.Error(fmt.Sprintf("Expected string, found %v", tok.Type))
}

func (p *Parser) expectString() (string, error) {
	tok := p.getToken()
	return p.assertString(tok)
}

func (p *Parser) expectEqualsString() (string, error) {
	err := p.expect(scanner.EQUALS)
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
	panic("HERE")
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
	err := p.expect(scanner.EQUALS)
	if err != nil {
		return nil, err
	}
	val, err = p.expectInt32()
	if err != nil {
		return nil, err
	}
	return &val, nil
}

func (p *Parser) expectNumber() (*decimal, error) {
	tok := p.getToken()
	if tok == nil {
		return nil, p.endOfFileError()
	}
	if tok.IsNumeric() {
		return parseDecimal(tok.Text)
	}
	return nil, p.Error(fmt.Sprintf("Expected number, found %v", tok.Type))
}

func (p *Parser) expectEqualsNumber() (*decimal, error) {
	err := p.expect(scanner.EQUALS)
	if err != nil {
		return nil, err
	}
	return p.expectNumber()
}

func (p *Parser) expect(toktype scanner.TokenType) error {
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

func (p *Parser) parseTypeOptions(td *TypeDef, acceptable ...string) error {
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
	Required bool
	Default interface{}
	Pattern string
	Values []string
	MinSize *int32
	MaxSize *int32
	Min *decimal
	Max *decimal
	Annotations map[string]string
}

func (p *Parser) parseOptions(datatype string, acceptable []string) (*Options, error) {
	fmt.Println("acceptable:", acceptable)
	options := &Options{}
	var err error
	tok := p.getToken()
	if tok == nil {
		return options, nil
	}
	if tok.Type == scanner.OPEN_PAREN {
		for {
			tok := p.getToken()
			if tok == nil {
				return nil, p.syntaxError()
			}
			if tok.Type == scanner.CLOSE_PAREN {
				return options, nil
			}
			if tok.Type == scanner.SYMBOL {
				if strings.HasPrefix(tok.Text, "x_") {
					options.Annotations, err = p.parseExtendedOption(options.Annotations, tok.Text)
				} else if containsOption(acceptable,tok.Text) {
					switch tok.Text {
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
						options.Default, err = p.parseEqualsLiteral(datatype)
					default:
						err = p.Error("Unrecognized option: " + tok.Text)
					}
				} else {
					err = p.Error(fmt.Sprintf("Unrecognized type option for %s: %s", datatype, tok.Text))
				}
				if err != nil {
					fmt.Println("err", err)
						panic("HERE")
					return nil, err
				}
			} else if tok.Type == scanner.COMMA {
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
		if tok.Type == scanner.EQUALS {
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

func (p *Parser) parseBytesOptions(typedef *TypeDef) error {
	tok := p.getToken()
	if tok == nil {
		return p.syntaxError()
	}
	expected := ""
	if tok.Type == scanner.OPEN_PAREN {
		for {
			tok := p.getToken()
			if tok == nil {
				return p.syntaxError()
			}
			if tok.Type == scanner.CLOSE_PAREN {
				return nil
			}
			if tok.Type == scanner.SYMBOL {
				switch tok.Text {
				case "minsize", "maxsize":
					expected = tok.Text
				default:
					return p.Error("invalid bytes option: " + tok.Text)
				}
			} else if tok.Type == scanner.EQUALS {
				if expected == "" {
					return p.syntaxError()
				}
			} else if tok.Type == scanner.NUMBER {
				if expected == "" {
					return p.syntaxError()
				}
				val, err := parseDecimal(tok.Text)
				if err != nil {
					return err
				}
				if expected == "minsize" {
					n := decimalToInt64(val)
					i := int32(n)
					typedef.MinSize = &i
				} else if expected == "maxsize" {
					n := decimalToInt64(val)
					i := int32(n)
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
	if tok.Type != scanner.EQUALS {
		return nil, p.syntaxError()
	}
	
	tok = p.getToken()
	if tok == nil {
		return nil, p.endOfFileError()
	}
	if tok.Type != scanner.OPEN_BRACKET {
		return nil, p.syntaxError()
	}
	for {
		tok = p.getToken()
		if tok == nil {
			return nil, p.endOfFileError()
		}
		if tok.Type == scanner.CLOSE_BRACKET {
			break
		}
		if tok.Type == scanner.STRING {
			values = append(values, tok.Text)
		} else if tok.Type == scanner.COMMA {
			//ignore
		} else {
			return nil, p.syntaxError()
		}
	}
	return values, nil
}

func (p *Parser) parseEnumElementDef() (*EnumElementDef, error) {
	comment := ""
	sym := ""
	var err error
	for {
		tok := p.getToken()
		if tok == nil {
			return nil, p.endOfFileError()
		}
		if tok.Type == scanner.CLOSE_BRACE {
			return nil, nil
		} else if tok.Type == scanner.LINE_COMMENT {
			comment = p.mergeComment(comment, tok.Text)
		} else if tok.Type == scanner.SEMICOLON || tok.Type == scanner.NEWLINE || tok.Type == scanner.COMMA {
			//ignore
		} else {
			sym, err = p.assertIdentifier(tok)
			if err != nil {
				return nil, err
			}
			break
		}
	}
	comment = p.parseTrailingComment(comment)
	return &EnumElementDef{
		Symbol: sym,
		Comment: comment,
	}, nil
}

func (p *Parser) expectNewline() error {
	tok := p.getToken()
	if tok == nil {
		return p.syntaxError()
	}
	if tok.Type != scanner.NEWLINE {
		p.ungetToken()
		return p.syntaxError()
	}
	return nil
}

func (p *Parser) parseStructFieldDef() (*StructFieldDef, error) {
	ftype, fparams, err := p.parseTypeSpec()
	if err != nil {
		if p.lastToken.Type == scanner.CLOSE_BRACE {
			err = nil
		}
		return nil, err
	}
	fname, err := p.expectIdentifier()
	if err != nil {
		return nil, err
	}
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
			return nil, p.syntaxError()
		}
	}
	if err != nil {
		return nil, err
	}
	field := &StructFieldDef{
		Name: fname,
		Type: ftype,
		Items: fitems,
		Keys: fkeys,
		Value: fvalue,
		Unit: funit,
   }
	err = p.parseStructFieldOptions(field)
	if err == nil {
		field.Comment, err = p.endOfStatement(field.Comment)
	}
	return field, nil
}

func (p *Parser) parseStructFieldOptions(field *StructFieldDef) error {
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
/*
	//parse options here: generic: ['required', 'default', values, x_*], bytes: [minsize, maxsize], string: [pattern, minsize, maxsize], numeric: [min, max]
	tok := p.getToken()
	if tok == nil {
		return p.syntaxError()
	}
	var err error
	var pattern string //string
	var values []string //string
	var min, max *decimal //numbers
	var minsize, maxsize *int32 //string, bytes, array, map
	generateFieldType := false
	if tok.Type == scanner.OPEN_PAREN {
		for {
			tok := p.getToken()
			if tok == nil {
				return p.syntaxError()
			}
			if tok.Type == scanner.CLOSE_PAREN {
				return nil
			}
			if tok.Type == scanner.SYMBOL {
				switch (tok.Text) {
				case "required":
					field.Required = true
				case "default":
					obj, err := p.parseEqualsLiteral(field.Type)
					if err != nil {
						return err
					}
					field.Default = obj
				case "pattern":
					if field.Type != "String" {
						return p.Error(fmt.Sprintf("Bad option for a '%s' type field: %s", field.Type, tok.Text))
					}
					pattern, err = p.expectEqualsString()
					generateFieldType = true
				case "values":
					if field.Type != "String" {
						return p.Error(fmt.Sprintf("Bad option for a '%s' type field: %s", field.Type, tok.Text))
					}
					values, err = p.expectEqualsStringArray()
					generateFieldType = true
//				case "minsize":
//					if field.Type != "String" && 
				default:
					//all the options end up here, but we must generate tmp classes for most of them.
					fmt.Println("FIXME define this option:", tok.Type)
				}
			} else {
				fmt.Println("FIXME Ignoring field option token:", tok)
			}
			if err != nil {
				return err
			}
		}
	} else {
		p.ungetToken()
		return nil
	}
}
*/

func (p *Parser) parseEqualsLiteral(expectedType string) (interface{}, error) {
	err := p.expect(scanner.EQUALS)
	if err != nil {
		return 0, err
	}
	return p.parseLiteralValue(expectedType)
}

func (p *Parser) parseLiteralValue(expectedType string) (interface{}, error) {
	tok := p.getToken()
	if tok == nil {
		return nil, p.syntaxError()
	}
	return p.parseLiteral(expectedType, tok)
}

func (p *Parser) parseLiteral(expectedType string, tok *scanner.Token) (interface{}, error) {
	switch tok.Type {
	case scanner.SYMBOL:
		return p.parseLiteralSymbol(tok)
	case scanner.STRING:
		return p.parseLiteralString(tok)
	case scanner.NUMBER:
		return p.parseLiteralNumber(expectedType, tok)
	case scanner.OPEN_BRACKET:
		return p.parseLiteralArray()
	case scanner.OPEN_BRACE:
		return p.parseLiteralObject()
	default:
		return nil, p.syntaxError()
	}
}

func (p *Parser) parseLiteralSymbol(tok *scanner.Token) (interface{}, error) {
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
func (p *Parser) parseLiteralString(tok *scanner.Token) (*string, error) {
	s := "\"" + tok.Text +"\""
	q, err := strconv.Unquote(s)
	if err != nil {
		return nil, p.Error("Improperly escaped string: " + tok.Text)
	}
	return &q, nil
}

func (p *Parser) parseLiteralNumber(expectedType string, tok *scanner.Token) (interface{}, error) {
	num, err := parseDecimal(tok.Text)
	if err != nil {
		return nil, p.Error(fmt.Sprintf("Not a valid number: %s", tok.Text))
	}
	switch expectedType {
	case "Int8", "Int16", "Int32", "Int64":
		return decimalToInt64(num), nil
	case "Float32", "Float64":
		return decimalToFloat64(num), nil
	default:
		return num, nil
	}
}

func (p *Parser) parseLiteralArray() (interface{}, error) {
	var ary []interface{}
	for {
		tok := p.getToken()
		if tok == nil {
			return nil, p.endOfFileError()
		}
		if tok.Type == scanner.CLOSE_BRACKET {
			return ary, nil
		}
		if tok.Type != scanner.COMMA {
			obj, err := p.parseLiteral("", tok)
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
		if tok.Type == scanner.CLOSE_BRACE {
			return obj, nil
		}
		if tok.Type == scanner.STRING {
			pkey, err := p.parseLiteralString(tok)
			if err != nil {
				return nil, err
			}
			err = p.expect(scanner.COLON)
			if err != nil {
				return nil, err
			}
			val, err := p.parseLiteralValue("")
			if err != nil {
				return nil, err
			}
			obj[*pkey] = val
		} else {
			fmt.Println("ignoring this token:", tok)
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

func (p *Parser) parseCollectionOptions(typedef *TypeDef) error {
	tok := p.getToken()
	if tok == nil {
		return p.syntaxError()
	}
	if tok.Type == scanner.OPEN_PAREN {
		for {
			tok := p.getToken()
			if tok == nil {
				return p.syntaxError()
			}
			if tok.Type == scanner.CLOSE_PAREN {
				return nil
			}
			if tok.Type == scanner.SYMBOL {
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
		if tok.Type == scanner.SEMICOLON {
			//ignore it
		} else if tok.Type == scanner.LINE_COMMENT {
			comment = p.mergeComment(comment, tok.Text)
		} else if tok.Type == scanner.NEWLINE {
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
		if tok.Type == scanner.LINE_COMMENT {
			comment = p.mergeComment(comment, tok.Text)
		} else {
			p.ungetToken()
			return comment
		}
	}
}

func (p *Parser) parseTrailingComment(comment string) string {
	tok := p.getToken()
	if tok != nil && tok.Type == scanner.LINE_COMMENT {
		comment = p.mergeComment(comment, tok.Text)
	} else {
		p.ungetToken()
	}
	return comment
}

func (p *Parser) mergeComment(comment1 string, comment2 string) string {
   if comment1 != "" {
      if comment2 != "" {
         return comment1 + " " + comment2
      }
      return comment1
   }
   return comment2
}

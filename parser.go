package sadl

import(
	"bufio"
	"fmt"
	"os"
	"strconv"
)

var Verbose bool

func str(arg interface{}) string {
	return fmt.Sprintf("%v", arg)
}

func debug(args ...interface{}) {
	if Verbose {
		max := len(args) - 1
		for i := 0; i < max; i++ {
			fmt.Print(str(args[i]))
		}
		fmt.Println(str(args[max]))
	}
}

func Parse(path string) (*Schema, error) {
	parser := &Parser{
		path: path,
		schema: &Schema{
			Types: make([]*TypeDef, 0),
		},
	}
	err := parser.Scan()
	if err != nil {
		return nil, err
	}
	err = parser.Parse()
	if err != nil {
		return nil, err
	}
	return parser.schema, nil
}

type Parser struct {
	path string
	tokens []*Token
	schema *Schema
	lastToken *Token
	ungottenToken *Token
}

func (p *Parser) Scan() error {
	fi, err := os.Open(p.path)
	if err != nil {
		return fmt.Errorf("Can't open file %q\n", p.path)
	}
	defer fi.Close()	
	reader := bufio.NewReader(fi)
	scanner := NewScanner(p.path, reader)
	var tokens []*Token
	for {
		tok := scanner.Scan()
		if tok.Type == EOF {
			break
		}
		if tok.Type == ILLEGAL {
			return fmt.Errorf("*** %s\n", formattedAnnotation(p.path, "", "Syntax error", &tok, RED, 5))
			os.Exit(1)
		} else if tok.Type != BLOCK_COMMENT {
			tokens = append(tokens, &tok)
		}
	}
	p.tokens = tokens
	return nil;
}

func (p *Parser) Error(msg string) error {
	debug("error, last token:", p.lastToken)
	return fmt.Errorf("*** %s\n", formattedAnnotation(p.path, "", msg, p.lastToken, RED, 5))
}

func (p *Parser) syntaxError() error {
	return p.Error("Syntax error")
}

func (p *Parser) ungetToken() {
	debug("ungetToken() -> ", p.lastToken)
	p.ungottenToken = p.lastToken
	p.lastToken = nil
}

func (p *Parser) getToken() *Token {
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

func (p *Parser) Parse() error {
	//process the tokens
	if p.schema.Name == "" {
		p.schema.Name = BaseFileName(p.path)
		//? should the name be an identifier? Should it get 
	}
	//namespace is optional
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
				err = p.parseName()
			case "namespace":
				err = p.parseNamespace()
			case "version":
				err = p.parseVersion()
			case "type":
				err = p.parseTypeDef(comment)
				comment = ""
			default:
				return p.Error("Expected 'type', 'name', 'namespace', or 'version'")
			}
		case LINE_COMMENT:
			comment = p.mergeComment(comment, tok.Text)
		case SEMICOLON:
			/* ignore */
		case NEWLINE:
			/* ignore */
		default:
			return p.Error("Expected 'type', 'name', 'namespace', or 'version'")
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) endOfFileError() error {
	return p.Error("Unexpected end of file")
//		return fmt.Errorf("Unexpected end of file")
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

func (p *Parser) expectText() (string, error) {
	tok := p.getToken()
	if tok == nil {
		return "", fmt.Errorf("Unexpected end of file")
	}
	if tok.Type == SYMBOL || tok.Type == STRING {
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
	if tok.Type == NUMBER {
		l, err := strconv.ParseInt(tok.Text, 10, 32)
		return int32(l), err
	}
	return 0, p.Error(fmt.Sprintf("Expected number, found %v", tok.Type))
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

func (p *Parser) parseName() error {
	txt, err := p.expectText()
	if err == nil {
		p.schema.Name = txt
	}
	return err
}

func (p *Parser) parseNamespace() error {
	txt, err := p.expectText()
	if err == nil {
		p.schema.Namespace = txt
	}
	return err
}

func (p *Parser) parseVersion() error {
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

func (p *Parser) parseTypeSpec() (string, string, string, error) {
	typeName, err := p.expectIdentifier()
	if err != nil {
		return "", "", "", err
	}
	tok := p.getToken()
	if tok == nil {
		return typeName, "", "", nil
	}
	if tok.Type == OPEN_ANGLE {
		if typeName == "Array" {
			items, err := p.expectIdentifier()
			if err != nil {
				return typeName, "", "", err
			}
			tok = p.getToken()
			if tok == nil || tok.Type != CLOSE_ANGLE {
				return typeName, "", "", p.syntaxError()
			}
			return typeName, items, "", nil
		} else if typeName == "Map" {
			keys, err := p.expectIdentifier()
			if err != nil {
				return typeName, "", "", err
			}
			err = p.expect(COMMA)
			items, err := p.expectIdentifier()
			if err != nil {
				return typeName, "", "", err
			}
			tok = p.getToken()
			if tok == nil || tok.Type != CLOSE_ANGLE {
				return typeName, "", "", p.syntaxError()
			}
			return typeName, items, keys, nil
		} else {
			return "", "", "", p.syntaxError()
		}
	}
	p.ungetToken()
	return typeName, "", "", nil
}

func (p *Parser) parseTypeDef(comment string) error {
	typeName, err := p.expectIdentifier()
	if err != nil {
		return err
	}
	superName, items, keys, err := p.parseTypeSpec()
//	superName, err := p.expectIdentifier()
	if err != nil {
		return err
	}
	//note: unlike RDL, there is not type extension of user types. There was never any runtime inheritance anyway. This is just clearer
	switch superName {
	case "Struct": //no inheritance this time, period. Use composition instead, just a lot clearer
		return p.parseStructDef(typeName, comment)
	case "Enum":
		return p.parseEnumDef(typeName, comment)
	case "String":
		return p.parseStringDef(typeName, comment)
	case "Int8", "Int16", "Int32", "Int64", "Float32", "Float64", "Decimal":
		return p.parseNumericDef(typeName, superName, comment)
	case "Array":
		return p.parseArrayDef(typeName, items, comment)
	case "Map":
		return p.parseMapDef(typeName, keys, items, comment)
	}
	return p.Error(fmt.Sprintf("Super type must be a base type: %v", superName))
//	return fmt.Errorf("NYI: parseType %s called %s -- NYI", superName, typeName)
}

func (p *Parser) parseNumericDef(typeName string, superName string, comment string) error {
	typedef := &TypeDef{
		Name: typeName,
		Type: superName,
	}
	err := p.parseNumericOptions(typedef)
	if err == nil {
		comment, err = p.endOfStatement(comment)
		if err == nil {
			typedef.Comment = comment
			p.schema.Types = append(p.schema.Types, typedef)
			return nil
		}
	}
	return err
}

func (p *Parser) parseNumericOptions(typedef *TypeDef) error {
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
				case "min", "max":
					expected = tok.Text
				default:
					return p.Error("invalid numeric option: " + tok.Text)
				}
			} else if tok.Type == EQUALS {
				if expected == "" {
					return p.syntaxError()
				}
			} else if tok.Type == NUMBER {
				if expected == "" {
					return p.syntaxError()
				}
				val, err := parseDecimal(tok.Text)
				if err != nil {
					return err
				}
				if expected == "min" {
					typedef.Min = val
				} else if expected == "max" {
					typedef.Min = val
				} else {
					return p.Error("numeric option must have numeric value")
				}
				expected = ""
			}
		}
	} else {
		p.ungetToken()
		return nil
	}
}


func (p *Parser) parseStringDefValues() ([]string, error) {
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

func (p *Parser) parseStringDefPattern() (string, error) {
	tok := p.getToken()
	if tok == nil {
		return "", p.endOfFileError()
	}
	if tok.Type != EQUALS {
		return "", p.syntaxError()
	}
	return p.expectString()
}

func (p *Parser) parseStringDef(typeName string, comment string) error {
	typedef := &TypeDef{
		Name: typeName,
		Type: "String",
	}
	var err error
	tok := p.getToken()
	if tok == nil {
		return p.endOfFileError()
	}
	var values []string
	var pattern string
	if tok.Type == OPEN_PAREN {
		for {
			tok = p.getToken()
			if tok == nil {
				return p.endOfFileError()
			}
			if tok.Type == CLOSE_PAREN {
				break
			}
			if tok.Type == SYMBOL {
				switch tok.Text {
				case "values":
					values, err = p.parseStringDefValues()
					if err != nil {
						return err
					}
					typedef.Values = values
				case "pattern":
					pattern, err = p.parseStringDefPattern()
					if err != nil {
						return err
					}
					typedef.Pattern = pattern
				case "minsize":
					num, err := p.expectEqualsIntLiteral()
					if err != nil {
						return err
					}
					typedef.MinSize = &num
				case "maxsize":
					num, err := p.expectEqualsIntLiteral()
					if err != nil {
						return err
					}
					typedef.MaxSize = &num
				default:
					return p.Error("Unknown string option: " + tok.Text)
				}
			} else {
				return p.syntaxError()
			}
		}
	}
	comment, err = p.endOfStatement(comment)
	if err != nil {
		return err
	}
	typedef.Comment = comment
	p.schema.Types = append(p.schema.Types, typedef)
	return nil
}

func (p *Parser) parseEnumDef(typeName string, comment string) error {
	tok := p.getToken()
	if tok.Type != OPEN_BRACE {
		return p.syntaxError()
	}
	comment = p.parseTrailingComment(comment)

	var elements []*EnumElementDef
	for {
		elem, err := p.parseEnumElementDef()
		if err != nil {
			return err
		}
		if elem == nil {
			break
		}
		elements = append(elements, elem)
	}
	typedef := &TypeDef{
		Name: typeName,
		Type: "Enum",
		Comment: comment,
		Elements: elements,
	}
	p.schema.Types = append(p.schema.Types, typedef)
	return nil
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
	if tok.Type != NEWLINE {
		p.ungetToken()
		return p.syntaxError()
	}
	return nil
}

func (p *Parser) parseStructDef(typeName string, comment string) error {
	tok := p.getToken()
	var fields []*StructFieldDef
	var err error
	if tok.Type == OPEN_BRACE {
		comment = p.parseTrailingComment(comment)
		tok := p.getToken()
		if tok == nil {
			return p.syntaxError()
		}
		if tok.Type != NEWLINE {
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
			fields = append(fields, field)
		}
		tok = p.getToken()
		comment, err = p.endOfStatement(comment)
		if err != nil {
			return err
		}
	} else {
		comment, err = p.endOfStatement(comment)
		if err != nil {
			return err
		}
	}
	typedef := &TypeDef{
		Name: typeName,
		Type: "Struct",
		Comment: comment,
		Fields: fields,
	}
	p.schema.Types = append(p.schema.Types, typedef)
	return nil
}

func (p *Parser) parseStructFieldDef() (*StructFieldDef, error) {
	comment := ""
	ftype, fitems, fkeys, err := p.parseTypeSpec()
	if err != nil {
		if p.lastToken.Type == CLOSE_BRACE {
			err = nil
		}
		return nil, err
	}
	fname, err := p.expectIdentifier()
	if err != nil {
		return nil, err
	}
	field := &StructFieldDef{
		Name: fname,
		Type: ftype,
		Items: fitems,
		Keys: fkeys,
   }
	err = p.parseStructFieldOptions(field)
	if err != nil {
		return nil, err
	}
	comment, err = p.endOfStatement(comment)
	if err != nil {
		return nil, err
	}
	field.Comment = comment
	return field, nil
}

func (p *Parser) parseStructFieldOptions(field *StructFieldDef) error {
	//parse options here: generic: ['required', 'default', values, x_*], bytes: [minsize, maxsize], string: [pattern, minsize, maxsize], numeric: [min, max]
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
				switch (tok.Text) {
				case "required":
					field.Required = true
				case "default":
					obj, err := p.parseEqualsLiteral(field.Type)
					if err != nil {
						return err
					}
					field.Default = obj
				default:
					//all the options end up here, but we must generate tmp classes for most of them.
					fmt.Println("FIXME define this option:", tok.Type)
				}
			} else {
				fmt.Println("FIXME Ignoring field option token:", tok)
			}
		}
	} else {
		p.ungetToken()
		return nil
	}
}

func (p *Parser) parseEqualsLiteral(expectedType string) (interface{}, error) {
	err := p.expect(EQUALS)
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

func (p *Parser) parseLiteral(expectedType string, tok *Token) (interface{}, error) {
	switch tok.Type {
	case SYMBOL:
		return p.parseLiteralSymbol(tok)
	case STRING:
		return p.parseLiteralString(tok)
	case NUMBER:
		return p.parseLiteralNumber(expectedType, tok)
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
	s := "\"" + tok.Text +"\""
	q, err := strconv.Unquote(s)
	if err != nil {
		return nil, p.Error("Improperly escaped string: " + tok.Text)
	}
	return &q, nil
}

func (p *Parser) parseLiteralNumber(expectedType string, tok *Token) (interface{}, error) {
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
		if tok.Type == CLOSE_BRACKET {
			return ary, nil
		}
		if tok.Type != COMMA {
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
		if tok.Type == CLOSE_BRACE {
			return obj, nil
		}
		if tok.Type == STRING {
			pkey, err := p.parseLiteralString(tok)
			if err != nil {
				return nil, err
			}
			fmt.Println("Key:", *pkey)
			err = p.expect(COLON)
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

func (p *Parser) parseArrayDef(typeName string, items string, comment string) error {
	typedef := &TypeDef{
		Name: typeName,
		Type: "Array",
		Items: items,
	}
	err := p.parseCollectionOptions(typedef)
	if err == nil {
		comment, err = p.endOfStatement(comment)
		if err == nil {
			typedef.Comment = comment
			p.schema.Types = append(p.schema.Types, typedef)
		}
	}
	return err
}

func (p *Parser) expectEqualsIntLiteral() (int32, error) {
	err := p.expect(EQUALS)
	if err != nil {
		return 0, err
	}
	num, err := p.expectInt32()
	if err != nil {
		return 0, err
	}
	return num, nil
}

func (p *Parser) parseCollectionOptions(typedef *TypeDef) error {
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
					num, err := p.expectEqualsIntLiteral()
					if err != nil {
						return err
					}
					typedef.MinSize = &num
				case "maxsize":
					num, err := p.expectEqualsIntLiteral()
					if err != nil {
						return err
					}
					typedef.MaxSize = &num
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

func (p *Parser) parseMapDef(typeName string, keys string, items string, comment string) error {
	typedef := &TypeDef{
		Name: typeName,
		Type: "Map",
	}
	err := p.parseCollectionOptions(typedef)
	if err == nil {		
		p.schema.Types = append(p.schema.Types, typedef)
	}
	return err
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
   if comment1 != "" {
      if comment2 != "" {
         return comment1 + " " + comment2
      }
      return comment1
   }
   return comment2
}

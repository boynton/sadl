package smithy

import(
	"fmt"
	"io/ioutil"
	"strings"
	"strconv"
	
	"github.com/boynton/sadl"
	"github.com/boynton/sadl/util"
)

// a quick and dirty parser for Smithy 1.0 IDL

func parse(path string) (*AST, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	src := string(b)
	p := &Parser{
		scanner: util.NewScanner(strings.NewReader(src)),
		path:    path,
		source:  src,
		
	}
	err = p.Parse()
	if err != nil {
		return nil, err
	}
//	fmt.Println(sadl.Pretty(p.ast))
	return p.ast, nil
}


type Parser struct {
	path           string
	source         string
	scanner        *util.Scanner
	ast            *AST
	lastToken      *util.Token
	prevLastToken  *util.Token
	ungottenToken  *util.Token
	namespace      string
	name           string
	currentComment string
}

func (p *Parser) Parse() error {
	var comment string
	var traits map[string]interface{}
	p.ast = &AST{
		Version: "1.0",
	}
	for {
		var err error
		tok := p.GetToken()
		if tok == nil {
			break
		}
		switch tok.Type {
		case util.SYMBOL:
			switch tok.Text {
			case "namespace":
				if traits != nil {
					return p.SyntaxError()
				}
				err = p.parseNamespace(comment)
			case "metadata":
				if traits != nil {
					return p.SyntaxError()
				}
				err = p.parseMetadata()
			case "service":
				err = p.parseService(comment)
			case "string":
				traits, comment = withCommentTrait(traits, comment)
				err = p.parseString(traits)
				traits = nil
			case "structure":
				traits, comment = withCommentTrait(traits, comment)
				err = p.parseStructure(traits)
				traits = nil
			case "list":
				traits, comment = withCommentTrait(traits, comment)
				err = p.parseList(traits)
				traits = nil
			case "operation":
				traits, comment = withCommentTrait(traits, comment)
				err = p.parseOperation(traits)
				traits = nil
			default:
				err = p.Error(fmt.Sprintf("Unknown shape: %s", tok.Text))
			}
			comment = ""
		case util.LINE_COMMENT:
			if strings.HasPrefix(tok.Text, "/") { //a triple slash means doc comment
				comment = p.MergeComment(comment, tok.Text[1:])
			}
		case util.AT:
			traits, err = p.parseTrait(traits)
		case util.SEMICOLON, util.NEWLINE:
			/* ignore */
		default:
			return p.SyntaxError()
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) UngetToken() {
	util.Debug("UngetToken() -> ", p.lastToken)
	p.ungottenToken = p.lastToken
	p.lastToken = p.prevLastToken
}

func (p *Parser) GetToken() *util.Token {
	if p.ungottenToken != nil {
		p.lastToken = p.ungottenToken
		p.ungottenToken = nil
		util.Debug("GetToken() -> ", p.lastToken)
		return p.lastToken
	}
	p.prevLastToken = p.lastToken
	tok := p.scanner.Scan()
	for {
		if tok.Type == util.EOF {
			return nil //fixme
		} else if tok.Type != util.BLOCK_COMMENT {
			break
		}
		tok = p.scanner.Scan()
	}
	p.lastToken = &tok
	util.Debug("GetToken() -> ", p.lastToken)
	return p.lastToken
}

func (p *Parser) ignore(toktype util.TokenType) error {
	tok := p.GetToken()
	if tok == nil {
		return p.EndOfFileError()
	}
	if tok.Type == toktype {
		return nil
	}
	p.UngetToken()
	return nil
}

func (p *Parser) expect(toktype util.TokenType) error {
	tok := p.GetToken()
	if tok == nil {
		return p.EndOfFileError()
	}
	if tok.Type == toktype {
		return nil
	}
	return p.Error(fmt.Sprintf("Expected %v, found %v", toktype, tok.Type))
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

func (p *Parser) assertIdentifier(tok *util.Token) (string, error) {
	if tok == nil {
		return "", p.EndOfFileError()
	}
	if tok.Type == util.SYMBOL {
		return tok.Text, nil
	}
	return tok.Text, p.Error(fmt.Sprintf("Expected symbol, found %v", tok.Type))
}

func (p *Parser) ExpectIdentifier() (string, error) {
	tok := p.GetToken()
	return p.assertIdentifier(tok)
}

func (p *Parser) assertString(tok *util.Token) (string, error) {
	if tok == nil {
		return "", p.EndOfFileError()
	}
	if tok.Type == util.STRING {
		return tok.Text, nil
	}
	return tok.Text, p.Error(fmt.Sprintf("Expected string, found %v", tok.Type))
}

func (p *Parser) ExpectInt() (int, error) {
	tok := p.GetToken()
	if tok == nil {
		return 0, p.EndOfFileError()
	}
	if tok.IsNumeric() {
		l, err := strconv.ParseInt(tok.Text, 10, 32)
		return int(l), err
	}
	return 0, p.Error(fmt.Sprintf("Expected number, found %v", tok.Type))
}

func (p *Parser) ExpectString() (string, error) {
	tok := p.GetToken()
	return p.assertString(tok)
}

func (p *Parser) ExpectStringArray() ([]string, error) {
	tok := p.GetToken()
	if tok == nil {
		return nil, p.EndOfFileError()
	}
	if tok.Type != util.OPEN_BRACKET {
		return nil, p.SyntaxError()
	}
	var items []string
	for {
		tok := p.GetToken()
		if tok == nil {
			return nil, p.EndOfFileError()
		}
		if tok.Type == util.CLOSE_BRACKET {
			break
		}
		s, err := p.assertString(tok)
		if err != nil {
			return nil, err
		}
		items = append(items, s)
		p.expect(util.COMMA)
	}
	return items, nil
}

func (p *Parser) ExpectIdentifierArray() ([]string, error) {
	tok := p.GetToken()
	if tok == nil {
		return nil, p.EndOfFileError()
	}
	if tok.Type != util.OPEN_BRACKET {
		return nil, p.SyntaxError()
	}
	var items []string
	for {
		tok := p.GetToken()
		if tok == nil {
			return nil, p.EndOfFileError()
		}
		if tok.Type == util.CLOSE_BRACKET {
			break
		}
		if tok.Type == util.SYMBOL {
			items = append(items, tok.Text)
		} else if tok.Type == util.COMMA || tok.Type == util.NEWLINE {
			//ignore
		} else {
			return nil, p.SyntaxError()
		}
	}
	return items, nil
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

func (p *Parser) Error(msg string) error {
	util.Debug("*** error, last token:", p.lastToken)
	return fmt.Errorf("*** %s\n", util.FormattedAnnotation(p.path, p.source, "", msg, p.lastToken, util.RED, 5))
}

func (p *Parser) SyntaxError() error {
	return p.Error("Syntax error")
}

func (p *Parser) EndOfFileError() error {
	return p.Error("Unexpected end of file")
}

func (p *Parser) parseMetadata() error {
	key, err := p.ExpectIdentifier()
	fmt.Println("key:", key)
	if err != nil {
		return err
	}
	err = p.expect(util.EQUALS)
	if err != nil {
		return err
	}
	val, err := p.parseLiteralValue()
	fmt.Println("val:", val)
	if err != nil {
		return err
	}
	if p.ast.Metadata == nil {
		p.ast.Metadata = make(map[string]interface{}, 0)
	}
	p.ast.Metadata[key] = val
	return nil
}

func (p *Parser) parseNamespace(comment string) error {
	//	p.schema.Comment = p.MergeComment(p.schema.Comment, comment)
	if p.namespace != "" {
		return p.Error("Only one namespace per file allowed")
	}
	ns := ""
	txt, err := p.expectText()
	if err != nil {
		return err
	}
	ns = txt
	for {
		tok := p.GetToken()
		if tok == nil {
			break
		}
		if tok.Type != util.DOT {
			p.UngetToken()
			break
		}
		ns = ns + "."
		txt, err = p.expectText()
		if err != nil {
			return err
		}
		ns = ns + txt
	}
	p.namespace = ns
	return err
}

func (p *Parser) addShapeDefinition(name string, shape *Shape) {
	if p.ast.Shapes == nil {
		p.ast.Shapes = make(map[string]*Shape, 0)
	}
	p.ast.Shapes[p.ensureNamespaced(name)] = shape
}

func (p *Parser) parseString(traits map[string]interface{}) error {
	tname, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	shape := &Shape{
		Type: "string",
	}
	if traits != nil {
		shape.Traits = traits
	}
	p.addShapeDefinition(tname, shape)
	return nil
}

func (p *Parser) parseList(traits map[string]interface{}) error {
	name, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	tok := p.GetToken()
	if tok == nil {
		return p.EndOfFileError()
	}
	if tok.Type != util.OPEN_BRACE {
		return p.SyntaxError()
	}	
	shape := &Shape{
		Type: "list",
		Traits: traits,
	}
	var mtraits map[string]interface{}
	for {
		tok := p.GetToken()
		if tok == nil {
			return p.EndOfFileError()
		}
		if tok.Type == util.NEWLINE {
			continue
		}
		if tok.Type == util.CLOSE_BRACE {
			break
		}
		if tok.Type == util.AT {
			mtraits, err = p.parseTrait(mtraits)
			if err != nil {
				return err
			}
		} else if tok.Type == util.SYMBOL {
			fname := tok.Text
			err = p.expect(util.COLON)
			if err != nil {
				return err
			}
			if fname != "member" {
				return p.SyntaxError()
			}
			
			ftype, err := p.ExpectIdentifier()
			if err != nil {
				return err
			}
			err = p.ignore(util.COMMA)
			shape.Member = &Member{
				Target: p.ensureNamespaced(ftype),
				Traits: mtraits,
			}
		} else {
			return p.SyntaxError()
		}
	}
	p.addShapeDefinition(name, shape)
	return nil
}

func (p *Parser) parseStructure(traits map[string]interface{}) error {
	name, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	tok := p.GetToken()
	if tok == nil {
		return p.EndOfFileError()
	}
	if tok.Type != util.OPEN_BRACE {
		return p.SyntaxError()
	}	
	shape := &Shape{
		Type: "structure",
		Traits: traits,
	}
	mems := make(map[string]*Member, 0)
	var mtraits map[string]interface{}
	for {
		tok := p.GetToken()
		if tok == nil {
			return p.EndOfFileError()
		}
		if tok.Type == util.NEWLINE {
			continue
		}
		if tok.Type == util.CLOSE_BRACE {
			break
		}
		if tok.Type == util.AT {
			mtraits, err = p.parseTrait(mtraits)
			if err != nil {
				return err
			}
		} else if tok.Type == util.SYMBOL {
			fname := tok.Text
			err = p.expect(util.COLON)
			if err != nil {
				return err
			}
			ftype, err := p.ExpectIdentifier()
			if err != nil {
				return err
			}
			err = p.ignore(util.COMMA)
			mems[fname] = &Member{
				Target: p.ensureNamespaced(ftype),
				Traits: mtraits,
			}
			mtraits = nil
		} else {
			return p.SyntaxError()
		}
	}
	shape.Members = mems
	p.addShapeDefinition(name, shape)
	return nil
}

func (p *Parser) parseOperation(traits map[string]interface{}) error {
	name, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	tok := p.GetToken()
	if tok == nil {
		return p.EndOfFileError()
	}
	if tok.Type != util.OPEN_BRACE {
		return p.SyntaxError()
	}	
	shape := &Shape{
		Type: "operation",
		Traits: traits,
	}
	for {
		tok := p.GetToken()
		if tok == nil {
			return p.EndOfFileError()
		}
		if tok.Type == util.NEWLINE {
			continue
		}
		if tok.Type == util.CLOSE_BRACE {
			break
		}
		if tok.Type != util.COLON {
			p.UngetToken()
		}
		fname, err := p.ExpectIdentifier()
		if err != nil {
			return err
		}
		err = p.expect(util.COLON)
		if err != nil {
			return err
		}
		switch fname {
		case "input":
			shape.Input, err = p.expectShapeRef()
		case "output":
			shape.Output, err = p.expectShapeRef()
		case "errors":
			shape.Errors, err = p.expectShapeRefs()
		default:
			return p.SyntaxError()
		}
		if err != nil {
			return err
		}
		err = p.ignore(util.COMMA)
	}
	p.addShapeDefinition(name, shape)
	return nil
}

func (p *Parser) parseService(comment string) error {
	name, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	tok := p.GetToken()
	if tok == nil {
		return p.EndOfFileError()
	}
	if tok.Type != util.OPEN_BRACE {
		return p.SyntaxError()
	}	
	shape := &Shape{
		Type: "service",
	}
	for {
		tok := p.GetToken()
		if tok == nil {
			return p.EndOfFileError()
		}
		if tok.Type == util.NEWLINE {
			continue
		}
		if tok.Type == util.CLOSE_BRACE {
			break
		}
		if tok.Type != util.COLON {
			p.UngetToken()
		}
		fname, err := p.ExpectIdentifier()
		if err != nil {
			return err
		}
		err = p.expect(util.COLON)
		if err != nil {
			return err
		}
		switch fname {
		case "version":
			shape.Version, err = p.ExpectString()
		case "operations":
			shape.Operations, err = p.expectShapeRefs()
		case "resources":
			shape.Resources, err = p.expectShapeRefs()
		default:
			return p.SyntaxError()
		}
		if err != nil {
			return err
		}
		err = p.ignore(util.COMMA)
	}
//Traits:
//	Operations []*ShapeRef `json:"operations,omitempty"`
//	Resources []*ShapeRef `json:"resources,omitempty"`
//	Version string `json:"version,omitempty"`

	p.addShapeDefinition(name, shape)
	return nil
}

func EnsureNamespaced(ns, name string) string {
	switch name {
	case "Boolean":
		return "Bool"
	case "Byte":
		return "Int8"
	case "Short":
		return "Int16"
	case "Integer":
		return "Int32"
	case "Long":
		return "Int64"
	case "Float":
		return "Float32"
	case "Double":
		return "Float64"
	case "BigInteger", "BigDecimal":
		return "Decimal"
	case "Blob":
		return "Bytes"
	case "String", "Timestamp", "UUID", "Map", "Enum", "Union":
		return name
	case "List":
		return "Array"
	case "Document", "Structure":
		return "Struct"
	}
	if strings.Index(name, "#") < 0 {
		return ns + "#" + name
	}
	return name
}

func (p *Parser) ensureNamespaced(name string) string {
	return EnsureNamespaced(p.namespace, name)
}
	
func (p *Parser) expectShapeRefs() ([]*ShapeRef, error) {
	targets, err := p.ExpectIdentifierArray()
   if err != nil {
      return nil, err
   }
	var refs []*ShapeRef
	for _, target := range targets {
		ref := &ShapeRef {
			Target: p.ensureNamespaced(target),
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

func (p *Parser) expectShapeRef() (*ShapeRef, error) {
	tname, err := p.ExpectIdentifier()
	if err != nil {
		return nil, err
	}
	ref := &ShapeRef {
		Target: p.ensureNamespaced(tname),
	}
	return ref, nil
}

func (p *Parser) parseTraitArgs() (map[string]interface{}, error) {
	var err error
	var args map[string]interface{}
	tok := p.GetToken()
	if tok == nil {
		return args, nil
	}
	if tok.Type == util.OPEN_PAREN {
		for {
			tok := p.GetToken()
			if tok == nil {
				return nil, p.SyntaxError()
			}
			if tok.Type == util.CLOSE_PAREN {
				return args, nil
			}
			if tok.Type == util.SYMBOL {
				p.ignore(util.COLON)
				match := tok.Text
				switch match {
				case "method", "uri", "inputToken", "outputToken", "pageSize", "maxResults":
					val, err := p.ExpectString()
					if err == nil {
						args = withTrait(args, match, val)
					}
				case "code":
					val, err := p.ExpectInt()
					if err == nil {
						args = withTrait(args, match, val)
					}
				default:
					err = p.Error("Unrecognized trait argument: " + tok.Text)
				}
				if err != nil {
					return nil, err
				}
			} else if tok.Type == util.COMMA {
				//ignore
			} else {
				return nil, p.SyntaxError()
			}
		}
	} else {
		p.UngetToken()
		return args, nil
	}
}

func (p *Parser) parseTrait(traits map[string]interface{}) (map[string]interface{}, error) {
	tname, err := p.ExpectIdentifier()
	if err != nil {
		return traits, err
	}
	switch tname {
	case "idempotent", "required", "httpLabel", "httpPayload", "readonly":
		return withTrait(traits, "smithy.api#" + tname, true), nil
	case "httpQuery", "httpHeader", "error", "documentation", "pattern":
		err := p.expect(util.OPEN_PAREN)
		if err != nil {
			return traits, err
		}
		s, err := p.ExpectString()
		if err != nil {
			return traits, err
		}
		err = p.expect(util.CLOSE_PAREN)
		if err != nil {
			return traits, err
		}
		return withTrait(traits, "smithy.api#" + tname, s), nil
	case "httpError":
		err := p.expect(util.OPEN_PAREN)
		if err != nil {
			return traits, err
		}
		n, err := p.ExpectInt()
		if err != nil {
			return traits, err
		}
		err = p.expect(util.CLOSE_PAREN)
		if err != nil {
			return traits, err
		}
		return withTrait(traits, "smithy.api#" + tname, n), nil
	case "http":
		args, err := p.parseTraitArgs()
		if err != nil {
			return traits, err
		}
		ht := &HttpTrait{
			Method: getString(args, "method"),
			Uri: getString(args, "uri"),
			Code: getInt(args, "code"),
		}
		return withTrait(traits, "smithy.api#http", ht), nil
	case "paginated":
		args, err := p.parseTraitArgs()
		if err != nil {
			return traits, err
		}
		//fixme: check
		return withTrait(traits, "smithy.api#paginated", args), nil
		
	default:
		return traits, p.Error(fmt.Sprintf("Unknown trait: @%s\n", tname))
	}
}

func withTrait(traits map[string]interface{}, key string, val interface{}) map[string]interface{} {
	if val != nil {
		if traits == nil {
			traits = make(map[string]interface{}, 0)
		}
		traits[key] = val
	}
	return traits
}

func withCommentTrait(traits map[string]interface{}, val string) (map[string]interface{}, string) {
	if val != "" {
		traits = withTrait(traits, "documentation", val)
	}
	return traits, ""
}

		
func (p *Parser) parseLiteralValue() (interface{}, error) {
	tok := p.GetToken()
	if tok == nil {
		return nil, p.SyntaxError()
	}
	return p.parseLiteral(tok)
}

func (p *Parser) parseLiteral(tok *util.Token) (interface{}, error) {
	switch tok.Type {
	case util.SYMBOL:
		return p.parseLiteralSymbol(tok)
	case util.STRING:
		return p.parseLiteralString(tok)
	case util.NUMBER:
		return p.parseLiteralNumber(tok)
	case util.OPEN_BRACKET:
		return p.parseLiteralArray()
	case util.OPEN_BRACE:
		return p.parseLiteralObject()
	default:
		return nil, p.SyntaxError()
	}
}

func (p *Parser) parseLiteralSymbol(tok *util.Token) (interface{}, error) {
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
func (p *Parser) parseLiteralString(tok *util.Token) (*string, error) {
	s := "\"" + tok.Text + "\""
	q, err := strconv.Unquote(s)
	if err != nil {
		return nil, p.Error("Improperly escaped string: " + tok.Text)
	}
	return &q, nil
}

func (p *Parser) parseLiteralNumber(tok *util.Token) (interface{}, error) {
	num, err := sadl.ParseDecimal(tok.Text)
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
		if tok.Type != util.NEWLINE {
			if tok.Type == util.CLOSE_BRACKET {
				return ary, nil
			}
			if tok.Type != util.COMMA {
				obj, err := p.parseLiteral(tok)
				if err != nil {
					return nil, err
				}
				ary = append(ary, obj)
			}
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
		if tok.Type == util.CLOSE_BRACE {
			return obj, nil
		}
		if tok.Type == util.STRING {
			pkey, err := p.parseLiteralString(tok)
			if err != nil {
				return nil, err
			}
			err = p.expect(util.COLON)
			if err != nil {
				return nil, err
			}
			val, err := p.parseLiteralValue()
			if err != nil {
				return nil, err
			}
			obj[*pkey] = val
		} else if tok.Type == util.SYMBOL {
			return nil, p.Error("Expected String key for JSON object, found symbol '" + tok.Text + "'")
		} else {
			//fmt.Println("ignoring this token:", tok)
		}
	}
}
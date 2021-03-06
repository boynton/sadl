package smithy

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/boynton/sadl"
)

// a quick and dirty parser for Smithy 1.0 IDL

func parse(path string) (*AST, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	src := string(b)
	p := &Parser{
		scanner: sadl.NewScanner(strings.NewReader(src)),
		path:    path,
		source:  src,
	}
	err = p.Parse()
	if err != nil {
		return nil, err
	}
	return p.ast, nil
}

type Parser struct {
	path           string
	source         string
	scanner        *sadl.Scanner
	ast            *AST
	lastToken      *sadl.Token
	prevLastToken  *sadl.Token
	ungottenToken  *sadl.Token
	namespace      string
	name           string
	currentComment string
	useTraits      map[string]string
}

func (p *Parser) Parse() error {
	var comment string
	var traits map[string]interface{}
	p.ast = &AST{
		Smithy: "1.0",
	}
	for {
		var err error
		tok := p.GetToken()
		if tok == nil {
			break
		}
		switch tok.Type {
		case sadl.SYMBOL:
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
			case "byte", "short", "integer", "long", "float", "double", "bigInteger", "bigDecimal", "string", "timestamp", "boolean":
				traits, comment = withCommentTrait(traits, comment)
				err = p.parseSimpleTypeDef(tok.Text, traits)
				traits = nil
			case "structure":
				traits, comment = withCommentTrait(traits, comment)
				err = p.parseStructure(traits)
				traits = nil
			case "union":
				traits, comment = withCommentTrait(traits, comment)
				err = p.parseUnion(traits)
				traits = nil
			case "list", "set":
				traits, comment = withCommentTrait(traits, comment)
				err = p.parseCollection(tok.Text, traits)
				traits = nil
			case "map":
				traits, comment = withCommentTrait(traits, comment)
				err = p.parseMap(tok.Text, traits)
				traits = nil
			case "operation":
				traits, comment = withCommentTrait(traits, comment)
				err = p.parseOperation(traits)
				traits = nil
			case "resource":
				traits, comment = withCommentTrait(traits, comment)
				err = p.parseResource(traits)
				traits = nil
			case "use":
				use, err := p.expectShapeId()
				if err == nil {
					shortName := stripNamespace(use)
					if p.useTraits == nil {
						p.useTraits = make(map[string]string, 0)
					}
					p.useTraits[shortName] = use
				}
			case "apply":
				var ftype string
				ftype, err = p.expectTarget()
				tok := p.GetToken()
				if tok == nil {
					return p.SyntaxError()
				}
				if tok.Type != sadl.AT {
					return p.SyntaxError()
				}
				if shape, ok := p.ast.Shapes[p.ensureNamespaced(ftype)]; ok {
					shape.Traits, err = p.parseTrait(shape.Traits)
				}
			default:
				err = p.Error(fmt.Sprintf("Unknown shape: %s", tok.Text))
			}
			comment = ""
		case sadl.LINE_COMMENT:
			if strings.HasPrefix(tok.Text, "/") { //a triple slash means doc comment
				comment = p.MergeComment(comment, tok.Text[1:])
			}
		case sadl.AT:
			traits, err = p.parseTrait(traits)
		case sadl.DOLLAR:
			variable, err := p.ExpectIdentifier()
			if err != nil {
				return err
			}
			err = p.expect(sadl.COLON)
			if err != nil {
				return err
			}
			v, err := p.parseLiteralValue()
			if err != nil {
				return err
			}
			switch variable {
			case "version":
				if s, ok := v.(*string); ok && strings.HasPrefix(*s, "1") {
				} else {
					return fmt.Errorf("Bad control statement (only version 1 or 1.0 is supported): $%s: %v\n", variable, v)
				}
			}
		case sadl.SEMICOLON, sadl.NEWLINE:
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
	sadl.Debug("UngetToken() -> ", p.lastToken)
	p.ungottenToken = p.lastToken
	p.lastToken = p.prevLastToken
}

func (p *Parser) GetToken() *sadl.Token {
	if p.ungottenToken != nil {
		p.lastToken = p.ungottenToken
		p.ungottenToken = nil
		sadl.Debug("GetToken() -> ", p.lastToken)
		return p.lastToken
	}
	p.prevLastToken = p.lastToken
	tok := p.scanner.Scan()
	for {
		if tok.Type == sadl.EOF {
			return nil //fixme
		} else if tok.Type != sadl.BLOCK_COMMENT {
			break
		}
		tok = p.scanner.Scan()
	}
	p.lastToken = &tok
	sadl.Debug("GetToken() -> ", p.lastToken)
	return p.lastToken
}

func (p *Parser) ignore(toktype sadl.TokenType) error {
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

func (p *Parser) expect(toktype sadl.TokenType) error {
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

func (p *Parser) assertIdentifier(tok *sadl.Token) (string, error) {
	if tok == nil {
		return "", p.EndOfFileError()
	}
	if tok.Type == sadl.SYMBOL {
		return tok.Text, nil
	}
	return tok.Text, p.Error(fmt.Sprintf("Expected symbol, found %v", tok.Type))
}

func (p *Parser) ExpectIdentifier() (string, error) {
	tok := p.GetToken()
	return p.assertIdentifier(tok)
}

func (p *Parser) assertString(tok *sadl.Token) (string, error) {
	if tok == nil {
		return "", p.EndOfFileError()
	}
	if tok.Type == sadl.STRING {
		return tok.Text, nil
	}
	if tok.Type == sadl.UNDEFINED {
		return tok.Text, p.Error(tok.Text)
	}
	return tok.Text, p.Error(fmt.Sprintf("Expected string, found %v", tok.Type))
}

func (p *Parser) ExpectNumber() (*sadl.Decimal, error) {
	tok := p.GetToken()
	if tok == nil {
		return nil, p.EndOfFileError()
	}
	if tok.IsNumeric() {
		return sadl.ParseDecimal(tok.Text)
	}
	return nil, p.Error(fmt.Sprintf("Expected number, found %v", tok.Type))
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
	return 0, p.Error(fmt.Sprintf("Expected integer, found %v", tok.Type))
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
	if tok.Type != sadl.OPEN_BRACKET {
		return nil, p.SyntaxError()
	}
	var items []string
	for {
		tok := p.GetToken()
		if tok == nil {
			return nil, p.EndOfFileError()
		}
		if tok.Type == sadl.CLOSE_BRACKET {
			break
		}
		s, err := p.assertString(tok)
		if err != nil {
			return nil, err
		}
		items = append(items, s)
		p.expect(sadl.COMMA)
	}
	return items, nil
}

func (p *Parser) ExpectIdentifierArray() ([]string, error) {
	tok := p.GetToken()
	if tok == nil {
		return nil, p.EndOfFileError()
	}
	if tok.Type != sadl.OPEN_BRACKET {
		return nil, p.SyntaxError()
	}
	var items []string
	for {
		tok := p.GetToken()
		if tok == nil {
			return nil, p.EndOfFileError()
		}
		if tok.Type == sadl.CLOSE_BRACKET {
			break
		}
		if tok.Type == sadl.SYMBOL {
			items = append(items, tok.Text)
		} else if tok.Type == sadl.COMMA || tok.Type == sadl.NEWLINE || tok.Type == sadl.LINE_COMMENT {
			//ignore
		} else {
			return nil, p.SyntaxError()
		}
	}
	return items, nil
}

func (p *Parser) ExpectIdentifierMap() (map[string]string, error) {
	tok := p.GetToken()
	if tok == nil {
		return nil, p.EndOfFileError()
	}
	if tok.Type != sadl.OPEN_BRACE {
		return nil, p.SyntaxError()
	}
	items := make(map[string]string, 0)
	for {
		tok := p.GetToken()
		var key string
		if tok == nil {
			return nil, p.EndOfFileError()
		}
		if tok.Type == sadl.CLOSE_BRACE {
			break
		}
		if tok.Type == sadl.SYMBOL {
			key = tok.Text
		} else if tok.Type == sadl.COMMA || tok.Type == sadl.NEWLINE || tok.Type == sadl.LINE_COMMENT {
			//ignore
			continue
		} else {
			return nil, p.SyntaxError()
		}
		err := p.expect(sadl.COLON)
		if err != nil {
			return nil, err
		}
		tok = p.GetToken()
		if tok == nil {
			return nil, p.EndOfFileError()
		}
		if tok.Type == sadl.CLOSE_BRACE {
			return nil, p.SyntaxError()
		}
		if tok.Type == sadl.SYMBOL {
			items[key] = tok.Text
		} else if tok.Type == sadl.COMMA || tok.Type == sadl.NEWLINE || tok.Type == sadl.LINE_COMMENT {
			//ignore
		} else {
			return nil, p.SyntaxError()
		}
	}
	return items, nil
}

func trimRightSpace(s string) string {
	return strings.TrimRight(s, " \t\n\v\f\r")
}

func (p *Parser) MergeComment(comment1 string, comment2 string) string {
	if comment1 == "" {
		return trimRightSpace(comment2)
	}
	return comment1 + "\n" + trimRightSpace(comment2)
}

func (p *Parser) Error(msg string) error {
	sadl.Debug("*** error, last token:", p.lastToken)
	return fmt.Errorf("*** %s\n", sadl.FormattedAnnotation(p.path, p.source, "", msg, p.lastToken, sadl.RED, 5))
}

func (p *Parser) SyntaxError() error {
	return p.Error("Syntax error")
}

func (p *Parser) EndOfFileError() error {
	return p.Error("Unexpected end of file")
}

func (p *Parser) parseMetadata() error {
	key, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	err = p.expect(sadl.EQUALS)
	if err != nil {
		return err
	}
	val, err := p.parseLiteralValue()
	if err != nil {
		return err
	}
	if p.ast.Metadata == nil {
		p.ast.Metadata = make(map[string]interface{}, 0)
	}
	p.ast.Metadata[key] = val
	return nil
}

func (p *Parser) expectTarget() (string, error) {
	ident, err := p.expectNamespacedIdentifier()
	if err != nil {
		return "", err
	}
	tok := p.GetToken()
	if tok == nil {
		return ident, nil
	}
	if tok.Type != sadl.HASH {
		p.UngetToken()
		return ident, nil
	}
	ident = ident + "#"
	txt, err := p.expectText()
	if err != nil {
		return "", err
	}
	return ident + txt, nil
}

func (p *Parser) expectNamespacedIdentifier() (string, error) {
	txt, err := p.expectText()
	if err != nil {
		return "", err
	}
	ident := txt
	for {
		tok := p.GetToken()
		if tok == nil {
			break
		}
		if tok.Type != sadl.DOT {
			p.UngetToken()
			break
		}
		ident = ident + "."
		txt, err = p.expectText()
		if err != nil {
			return "", err
		}
		ident = ident + txt
	}
	return ident, nil
}

func (p *Parser) expectShapeId() (string, error) {
	txt, err := p.expectText()
	if err != nil {
		return "", err
	}
	ident := txt
	for {
		tok := p.GetToken()
		if tok == nil {
			break
		}
		if tok.Type != sadl.DOT {
			p.UngetToken()
			break
		}
		ident = ident + "."
		txt, err = p.expectText()
		if err != nil {
			return "", err
		}
		ident = ident + txt
	}
	for {
		tok := p.GetToken()
		if tok == nil {
			break
		}
		if tok.Type == sadl.HASH {
			key, err := p.ExpectIdentifier()
			if err != nil {
				return "", err
			}
			ident = ident + "#" + key
		} else {
			p.UngetToken()
			break
		}
	}
	return ident, nil
}

func (p *Parser) parseNamespace(comment string) error {
	//	p.schema.Comment = p.MergeComment(p.schema.Comment, comment)
	if p.namespace != "" {
		return p.Error("Only one namespace per file allowed")
	}
	ns, err := p.expectNamespacedIdentifier()
	p.namespace = ns
	return err
}

func (p *Parser) addShapeDefinition(name string, shape *Shape) {
	if p.ast.Shapes == nil {
		p.ast.Shapes = make(map[string]*Shape, 0)
	}
	p.ast.Shapes[p.ensureNamespaced(name)] = shape
}

func (p *Parser) parseSimpleTypeDef(typeName string, traits map[string]interface{}) error {
	tname, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	shape := &Shape{
		Type:   typeName,
		Traits: traits,
	}
	p.addShapeDefinition(tname, shape)
	return nil
}

func (p *Parser) parseCollection(sname string, traits map[string]interface{}) error {
	name, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	tok := p.GetToken()
	if tok == nil {
		return p.EndOfFileError()
	}
	if tok.Type != sadl.OPEN_BRACE {
		return p.SyntaxError()
	}
	shape := &Shape{
		Type:   sname,
		Traits: traits,
	}
	var mtraits map[string]interface{}
	for {
		tok := p.GetToken()
		if tok == nil {
			return p.EndOfFileError()
		}
		if tok.Type == sadl.NEWLINE {
			continue
		}
		if tok.Type == sadl.CLOSE_BRACE {
			break
		}
		if tok.Type == sadl.AT {
			mtraits, err = p.parseTrait(mtraits)
			if err != nil {
				return err
			}
		} else if tok.Type == sadl.SYMBOL {
			fname := tok.Text
			err = p.expect(sadl.COLON)
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
			err = p.ignore(sadl.COMMA)
			shape.Member = &Member{
				Target: p.ensureNamespaced(ftype),
				Traits: mtraits,
			}
		} else {
			return p.SyntaxError()
		}
	}
	if shape.Member == nil {
		return p.Error("expected 'member' attribute, found none")
	}
	p.addShapeDefinition(name, shape)
	return nil
}

func (p *Parser) parseMap(sname string, traits map[string]interface{}) error {
	name, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	tok := p.GetToken()
	if tok == nil {
		return p.EndOfFileError()
	}
	if tok.Type != sadl.OPEN_BRACE {
		return p.SyntaxError()
	}
	shape := &Shape{
		Type:   sname,
		Traits: traits,
	}
	var mtraits map[string]interface{}
	for {
		tok := p.GetToken()
		if tok == nil {
			return p.EndOfFileError()
		}
		if tok.Type == sadl.NEWLINE {
			continue
		}
		if tok.Type == sadl.CLOSE_BRACE {
			break
		}
		if tok.Type == sadl.AT {
			mtraits, err = p.parseTrait(mtraits)
			if err != nil {
				return err
			}
		} else if tok.Type == sadl.SYMBOL {
			fname := tok.Text
			err = p.expect(sadl.COLON)
			if err != nil {
				return err
			}
			ftype, err := p.ExpectIdentifier()
			if err != nil {
				return err
			}
			err = p.ignore(sadl.COMMA)
			if fname == "key" {
				shape.Key = &Member{
					Target: p.ensureNamespaced(ftype),
					Traits: mtraits,
				}
				mtraits = nil
			} else if fname == "value" {
				shape.Value = &Member{
					Target: p.ensureNamespaced(ftype),
					Traits: mtraits,
				}
				mtraits = nil
			} else {
				return p.SyntaxError()
			}
		} else {
			return p.SyntaxError()
		}
	}
	if shape.Key == nil {
		return p.Error("expected 'key' attribute, found none")
	}
	if shape.Value == nil {
		return p.Error("expected 'value' attribute, found none")
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
	if tok.Type != sadl.OPEN_BRACE {
		return p.SyntaxError()
	}
	shape := &Shape{
		Type:   "structure",
		Traits: traits,
	}
	mems := make(map[string]*Member, 0)
	comment := ""
	var mtraits map[string]interface{}
	for {
		tok := p.GetToken()
		if tok == nil {
			return p.EndOfFileError()
		}
		if tok.Type == sadl.NEWLINE {
			continue
		}
		if tok.Type == sadl.CLOSE_BRACE {
			break
		}
		if tok.Type == sadl.AT {
			mtraits, err = p.parseTrait(mtraits)
			if err != nil {
				return err
			}
		} else if tok.Type == sadl.SYMBOL {
			fname := tok.Text
			err = p.expect(sadl.COLON)
			if err != nil {
				return err
			}
			ftype, err := p.ExpectIdentifier()
			if err != nil {
				return err
			}
			err = p.ignore(sadl.COMMA)
			if comment != "" {
				mtraits, comment = withCommentTrait(mtraits, comment)
				comment = ""
			}
			mems[fname] = &Member{
				Target: p.ensureNamespaced(ftype),
				Traits: mtraits,
			}
			mtraits = nil
		} else if tok.Type == sadl.LINE_COMMENT {
			if strings.HasPrefix(tok.Text, "/") { //a triple slash means doc comment
				comment = p.MergeComment(comment, tok.Text[1:])
			}
		} else {
			return p.SyntaxError()
		}
	}
	shape.Members = mems
	p.addShapeDefinition(name, shape)
	return nil
}

func (p *Parser) parseUnion(traits map[string]interface{}) error {
	name, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	tok := p.GetToken()
	if tok == nil {
		return p.EndOfFileError()
	}
	if tok.Type != sadl.OPEN_BRACE {
		return p.SyntaxError()
	}
	shape := &Shape{
		Type:   "union",
		Traits: traits,
	}
	mems := make(map[string]*Member, 0)
	var mtraits map[string]interface{}
	for {
		tok := p.GetToken()
		if tok == nil {
			return p.EndOfFileError()
		}
		if tok.Type == sadl.NEWLINE {
			continue
		}
		if tok.Type == sadl.CLOSE_BRACE {
			break
		}
		if tok.Type == sadl.AT {
			mtraits, err = p.parseTrait(mtraits)
			if err != nil {
				return err
			}
		} else if tok.Type == sadl.SYMBOL {
			fname := tok.Text
			err = p.expect(sadl.COLON)
			if err != nil {
				return err
			}
			ftype, err := p.expectTarget()
			if err != nil {
				return err
			}
			err = p.ignore(sadl.COMMA)
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
	if tok.Type != sadl.OPEN_BRACE {
		return p.SyntaxError()
	}
	shape := &Shape{
		Type:   "operation",
		Traits: traits,
	}
	for {
		tok := p.GetToken()
		if tok == nil {
			return p.EndOfFileError()
		}
		if tok.Type == sadl.NEWLINE {
			continue
		}
		if tok.Type == sadl.CLOSE_BRACE {
			break
		}
		if tok.Type != sadl.COLON {
			p.UngetToken()
		}
		fname, err := p.ExpectIdentifier()
		if err != nil {
			return err
		}
		err = p.expect(sadl.COLON)
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
		err = p.ignore(sadl.COMMA)
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
	if tok.Type != sadl.OPEN_BRACE {
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
		if tok.Type == sadl.NEWLINE {
			continue
		}
		if tok.Type == sadl.CLOSE_BRACE {
			break
		}
		if tok.Type != sadl.COLON {
			p.UngetToken()
		}
		fname, err := p.ExpectIdentifier()
		if err != nil {
			return err
		}
		err = p.expect(sadl.COLON)
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
		err = p.ignore(sadl.COMMA)
	}
	//Traits:
	//	Operations []*ShapeRef `json:"operations,omitempty"`
	//	Resources []*ShapeRef `json:"resources,omitempty"`
	//	Version string `json:"version,omitempty"`

	p.addShapeDefinition(name, shape)
	return nil
}

func (p *Parser) parseResource(traits map[string]interface{}) error {
	name, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	tok := p.GetToken()
	if tok == nil {
		return p.EndOfFileError()
	}
	if tok.Type != sadl.OPEN_BRACE {
		return p.SyntaxError()
	}
	shape := &Shape{
		Type:   "resource",
		Traits: traits,
	}
	var comment string
	traits, comment = withCommentTrait(traits, comment)
	for {
		tok := p.GetToken()
		if tok == nil {
			return p.EndOfFileError()
		}
		if tok.Type == sadl.NEWLINE {
			continue
		}
		if tok.Type == sadl.CLOSE_BRACE {
			break
		}
		if tok.Type == sadl.LINE_COMMENT {
			if strings.HasPrefix(tok.Text, "/") { //a triple slash means doc comment
				comment = p.MergeComment(comment, tok.Text[1:])
			}
			continue
		} else {
			p.UngetToken()
		}
		fname, err := p.ExpectIdentifier()
		if err != nil {
			return err
		}
		err = p.expect(sadl.COLON)
		if err != nil {
			return err
		}
		switch fname {
		case "identifiers":
			shape.Identifiers, err = p.expectNamedShapeRefs()
		case "create":
			shape.Create, err = p.expectShapeRef()
		case "put":
			shape.Put, err = p.expectShapeRef()
		case "read":
			shape.Read, err = p.expectShapeRef()
		case "update":
			shape.Update, err = p.expectShapeRef()
		case "delete":
			shape.Delete, err = p.expectShapeRef()
		case "list":
			shape.Delete, err = p.expectShapeRef()
		case "operations":
			shape.Operations, err = p.expectShapeRefs()
		case "collectionOperations":
			shape.CollectionOperations, err = p.expectShapeRefs()
		case "Resources":
			shape.Resources, err = p.expectShapeRefs()
		default:
			return p.SyntaxError()
		}
		if err != nil {
			return err
		}
		err = p.ignore(sadl.COMMA)
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
	case "Boolean", "Byte", "Short", "Integer", "Long", "Float", "Double", "BigInteger", "BigDecimal":
		return name
	case "Blob", "String", "Timestamp", "UUID", "Enum":
		return name
	case "List", "Map", "Set", "Document", "Structure", "Union":
		return name
	}
	if strings.Index(name, "#") < 0 {
		return ns + "#" + name
	}
	return name
}

func (p *Parser) ensureNamespaced(name string) string {
	return EnsureNamespaced(p.namespace, name)
}

func (p *Parser) expectNamedShapeRefs() (map[string]*ShapeRef, error) {
	targets, err := p.ExpectIdentifierMap()
	if err != nil {
		return nil, err
	}
	refs := make(map[string]*ShapeRef, 0)
	for k, target := range targets {
		ref := &ShapeRef{
			Target: p.ensureNamespaced(target),
		}
		refs[k] = ref
	}
	return refs, nil
}

func (p *Parser) expectShapeRefs() ([]*ShapeRef, error) {
	targets, err := p.ExpectIdentifierArray()
	if err != nil {
		return nil, err
	}
	var refs []*ShapeRef
	for _, target := range targets {
		ref := &ShapeRef{
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
	ref := &ShapeRef{
		Target: p.ensureNamespaced(tname),
	}
	return ref, nil
}

func (p *Parser) parseTraitArgs() (map[string]interface{}, interface{}, error) {
	var err error
	var args map[string]interface{}
	var literal interface{}
	tok := p.GetToken()
	if tok == nil {
		return args, nil, nil
	}
	if tok.Type == sadl.OPEN_PAREN {
		for {
			tok := p.GetToken()
			if tok == nil {
				return nil, nil, p.SyntaxError()
			}
			if tok.Type == sadl.CLOSE_PAREN {
				return args, literal, nil
			}
			if tok.Type == sadl.SYMBOL {
				p.ignore(sadl.COLON)
				match := tok.Text
				switch match {
				case "method", "uri", "inputToken", "outputToken", "pageSize", "maxResults", "items", "selector":
					val, err := p.ExpectString()
					if err == nil {
						args = withTrait(args, match, val)
					}
				case "min", "max":
					val, err := p.ExpectNumber()
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
					return nil, nil, err
				}
			} else if tok.Type == sadl.OPEN_BRACKET {
				literal, err = p.parseLiteralArray()
				if err != nil {
					return nil, nil, err
				}
			} else if tok.Type == sadl.COMMA {
				//ignore
			} else {
				return nil, nil, p.SyntaxError()
			}
		}
	} else {
		p.UngetToken()
		return args, nil, nil
	}
}

func (p *Parser) parseTrait(traits map[string]interface{}) (map[string]interface{}, error) {
	tname, err := p.ExpectIdentifier()
	if err != nil {
		return traits, err
	}
	switch tname {
	case "idempotent", "required", "httpLabel", "httpPayload", "readonly": //booleans
		return withTrait(traits, "smithy.api#"+tname, true), nil
	case "httpQuery", "httpHeader", "error", "documentation", "pattern", "title", "timestampFormat": //strings
		err := p.expect(sadl.OPEN_PAREN)
		if err != nil {
			return traits, err
		}
		s, err := p.ExpectString()
		if err != nil {
			return traits, err
		}
		err = p.expect(sadl.CLOSE_PAREN)
		if err != nil {
			return traits, err
		}
		return withTrait(traits, "smithy.api#"+tname, s), nil
	case "tags":
		_, tags, err := p.parseTraitArgs()
		return withTrait(traits, "smithy.api#tags", tags), err
	case "httpError":
		err := p.expect(sadl.OPEN_PAREN)
		if err != nil {
			return traits, err
		}
		n, err := p.ExpectInt()
		if err != nil {
			return traits, err
		}
		err = p.expect(sadl.CLOSE_PAREN)
		if err != nil {
			return traits, err
		}
		return withTrait(traits, "smithy.api#"+tname, n), nil
	case "http":
		args, _, err := p.parseTraitArgs()
		if err != nil {
			return traits, err
		}
		return withTrait(traits, "smithy.api#http", args), nil
	case "length":
		args, _, err := p.parseTraitArgs()
		if err != nil {
			return traits, err
		}
		return withTrait(traits, "smithy.api#length", args), nil
	case "range":
		args, _, err := p.parseTraitArgs()
		if err != nil {
			return traits, err
		}
		return withTrait(traits, "smithy.api#range", args), nil
	case "paginated":
		args, _, err := p.parseTraitArgs()
		if err != nil {
			return traits, err
		}
		return withTrait(traits, "smithy.api#paginated", args), nil
	case "enum":
		_, lit, err := p.parseTraitArgs()
		if err != nil {
			return traits, err
		}
		if lit == nil {
			return traits, p.SyntaxError()
		}
		return withTrait(traits, "smithy.api#enum", lit), nil
	case "examples":
		_, lit, err := p.parseTraitArgs()
		if err != nil {
			return traits, err
		}
		if lit == nil {
			return traits, p.SyntaxError()
		}
		return withTrait(traits, "smithy.api#examples", lit), nil
	case "trait":
		args, _, err := p.parseTraitArgs()
		if err != nil {
			return traits, err
		}
		return withTrait(traits, "smithy.api#trait", args), nil
	default:
		if ctrait, ok := p.useTraits[tname]; ok { //if the trait is defined in this namespace, shouldn't require 'use'
			args, lit, err := p.parseTraitArgs()
			if err != nil {
				return traits, err
			}
			if lit != nil {
				return withTrait(traits, ctrait, lit), nil
			}
			return withTrait(traits, ctrait, args), nil
		}
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
		if strings.HasSuffix(val, "\n") {
			if !strings.HasPrefix(val, "\n") {
				val = "\n" + val
			}
		}
		traits = withTrait(traits, "smithy.api#documentation", val)
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

func (p *Parser) parseLiteral(tok *sadl.Token) (interface{}, error) {
	switch tok.Type {
	case sadl.SYMBOL:
		return p.parseLiteralSymbol(tok)
	case sadl.STRING:
		return p.parseLiteralString(tok)
	case sadl.NUMBER:
		return p.parseLiteralNumber(tok)
	case sadl.OPEN_BRACKET:
		return p.parseLiteralArray()
	case sadl.OPEN_BRACE:
		return p.parseLiteralObject()
	default:
		return nil, p.SyntaxError()
	}
}

func (p *Parser) parseLiteralSymbol(tok *sadl.Token) (interface{}, error) {
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
func (p *Parser) parseLiteralString(tok *sadl.Token) (*string, error) {
	s := "\"" + tok.Text + "\""
	q, err := strconv.Unquote(s)
	if err != nil {
		return nil, p.Error("Improperly escaped string: " + tok.Text)
	}
	return &q, nil
}

func (p *Parser) parseLiteralNumber(tok *sadl.Token) (interface{}, error) {
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
		if tok.Type != sadl.NEWLINE {
			if tok.Type == sadl.CLOSE_BRACKET {
				return ary, nil
			}
			if tok.Type != sadl.COMMA {
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
		if tok.Type == sadl.CLOSE_BRACE {
			return obj, nil
		}
		if tok.Type == sadl.STRING {
			pkey, err := p.parseLiteralString(tok)
			if err != nil {
				return nil, err
			}
			err = p.expect(sadl.COLON)
			if err != nil {
				return nil, err
			}
			val, err := p.parseLiteralValue()
			if err != nil {
				return nil, err
			}
			obj[*pkey] = val
		} else if tok.Type == sadl.SYMBOL {
			return nil, p.Error("Expected String key for JSON object, found symbol '" + tok.Text + "'")
		} else {
			//fmt.Println("ignoring this token:", tok)
		}
	}
}

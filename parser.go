package main

import(
	"bufio"
	"fmt"
	"os"
)

//TODO remove this
func main() {
	if len(os.Args) != 2 {
		fmt.Println("usage: scanner file.sadl")
		os.Exit(1)
	}
	path := os.Args[1]
	schema, err := Parse(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	fmt.Println(Pretty(schema))
}

func Parse(path string) (*Schema, error) {
	parser := &Parser{
		path: path,
		schema: &Schema{
			Types: make(map[string]TypeDef, 0),
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
	return fmt.Errorf("*** %s\n", formattedAnnotation(p.path, "", msg, p.lastToken, RED, 5))
}

func (p *Parser) getToken() *Token {
	if len(p.tokens) == 0 {
		return nil
	}
	p.lastToken = p.tokens[0]
	p.tokens = p.tokens[1:]
	return p.lastToken
}

func (p *Parser) Parse() error {
	//process the tokens
	if p.schema.Name == "" {
		p.schema.Name = BaseFileName(p.path)
		//? should the name be an identifier? Should it get 
	}
	//namespace is optional
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
				err = p.parseTypeDef()
			}
		case SEMICOLON:
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

func (p *Parser) expectIdentifier() (string, error) {
	tok := p.getToken()
	if tok == nil {
		return "", fmt.Errorf("Unexpected end of file")
	}
	if tok.Type == SYMBOL {
		return tok.Text, nil
	}
	return "", fmt.Errorf("Expected symbol or string, found %v", tok.Type)
}

func (p *Parser) expectText() (string, error) {
	tok := p.getToken()
	if tok == nil {
		return "", fmt.Errorf("Unexpected end of file")
	}
	if tok.Type == SYMBOL || tok.Type == STRING {
		return tok.Text, nil
	}
	return "", fmt.Errorf("Expected symbol or string, found %v", tok.Type)
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
	txt, err := p.expectText()
	if err == nil {
		p.schema.Version = txt
	}
	return err
}

func (p *Parser) parseTypeDef() error {
	typeName, err := p.expectIdentifier()
	if err != nil {
		return err
	}
	superName, err := p.expectIdentifier()
	if err != nil {
		return err
	}
	switch superName {
	case "Struct": //no inheritance this time, period. Use composition instead, just a lot clearer
		return p.parseStructDef(typeName)
	}
	return p.Error("parseType() NYI for this type")
//	return fmt.Errorf("NYI: parseType %s called %s -- NYI", superName, typeName)
}

func (p *Parser) parseStructDef(typeName string) error {
	//struct options?
	//annotations?
	//do we want to handle self-refs? Not sure it makes sense
	tok := p.getToken()
	if tok.Type == SEMICOLON {
		//just an open struct, not fields defined.
		return p.Error("Open structs NYI")
	}
	if tok.Type != OPEN_BRACE {
		return p.Error("Syntax error")
	}
	//now parse fields
	def := &StructTypeDef{
	}
	for {
		field, err := p.parseStructFieldDef(def)
		if err != nil {
			return err
		}
		if field == nil {
			break
		}
		def.Fields = append(def.Fields, field)
	}
	typedef := &TypeDef{
		Name: typeName,
		Type: "Struct",
		Struct: def,
	}
	p.schema.Types[typeName] = *typedef
	return nil
}

func (p *Parser) parseStructFieldDef(def *StructTypeDef) (*StructFieldDef, error) {
	ftype, err := p.expectIdentifier()
	if err != nil {
		if p.lastToken.Type == CLOSE_BRACE {
			return nil, nil
		}
		return nil, err
	}
	fname, err := p.expectIdentifier()
	if err != nil {
		return nil, err
	}
	return &StructFieldDef{
		Name: fname,
		Type: ftype,
	}, nil
}


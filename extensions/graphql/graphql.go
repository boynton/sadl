package graphql

import(
	"fmt"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/parse"
)

type Model struct {
	sadl.Model
	GraphQL Def `json:"graphql,omitempty"`
}

type GraphQLEntityBinding struct {
   Name        string               `json:"name"`
	Operation string `json:"operation"`
	Comment     string            `json:"comment,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type Operation struct {
   Name        string               `json:"name"`
   Params      []*Param               `json:"params"`
   Return        *sadl.TypeSpec               `json:"return"`
}

type Param struct {
   Name        string            `json:"name"`
   Type        string            `json:"type"`
}

type Def struct {
	Path        string               `json:"path"`
	Comment     string               `json:"comment,omitempty"`
	Annotations map[string]string    `json:"annotations,omitempty"`
	Operations []*Operation    `json:"operations,omitempty"`
}

func ParseFile(path string) (*Model, error) {
	p := NewExtension()
	model, err := parse.File(path, p)
	if err != nil {
		return nil, err
	}
	return &Model{
		Model: *model,
		GraphQL: *p.Model,
	}, nil
}

func NewExtension() *Extension {
	return &Extension{
		Model: &Def{},
	}
}

type Extension struct {
	Model *Def
}

func (gql *Extension) Name() string {
	return "graphql"
}

func (gql *Extension) Parse(p *parse.Parser) error {
	path, err := p.ExpectString()
	if err != nil {
		return err
	}
	options, err := p.ParseOptions("graphql", []string{})
	if err != nil {
		return err
	}
	gql.Model.Path = path
	gql.Model.Annotations = options.Annotations
	gql.Model.Comment = p.CurrentComment()
	tok := p.GetToken()
	if tok == nil {
		return p.EndOfFileError()
	}
	if tok.Type == parse.OPEN_BRACE {
		gql.Model.Comment = p.ParseTrailingComment(gql.Model.Comment)
		comment := ""
		for {
			done, comment, err := p.IsBlockDone(comment)
			if done {
				gql.Model.Comment = p.MergeComment(gql.Model.Comment, comment)
				break
			}
			err = gql.parseQuerySpec(p, comment)
			if err != nil {
				return err
			}
		}
	} else {
		p.UngetToken()
	}
	gql.Model.Comment, err = p.EndOfStatement(gql.Model.Comment)
	fmt.Println("graphql parsed:", parse.Pretty(gql.Model))
	return err

}

/*
   character(id String) Character (operation=GetCharacter)
   characters Array<Character> (operation=ListCharacters)
   film(id String) Film (operation=GetFilm)
*/

func (gql *Extension) parseQuerySpec(p *parse.Parser, comment string) error {
	qName, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	fmt.Println("qName:", qName)
	params, err := gql.parseParams(p, qName)
	if err != nil {
		return err
	}
	fmt.Println("params:", parse.Pretty(params))
	ts, qOptions, qcomment, err := p.ParseTypeSpec(comment)
	if err != nil {
		return err
	}
	fmt.Println("return type:", parse.Pretty(ts), "qOptions:", qOptions, "qcomment:", qcomment)
	options, err := p.ParseOptions("graphql", []string{"operation"})
	if err != nil {
		return err
	}
	fmt.Println("options:", options)
	op := &Operation{
		Name: qName,
		Params: params,
		Return: ts,
	}
	gql.Model.Operations = append(gql.Model.Operations, op)
	return nil
}

func (gql *Extension) parseParams(p *parse.Parser, qName string) ([]*Param, error) {
	params := make([]*Param, 0)
	tok := p.GetToken()
	if tok == nil {
		return params, nil
	}
	if tok.Type == parse.OPEN_PAREN {
		for {
			tok := p.GetToken()
			if tok == nil {
				return nil, p.SyntaxError()
			}
			if tok.Type == parse.CLOSE_PAREN {
				return params, nil
			}
			if tok.Type == parse.SYMBOL {
				pName := tok.Text
				pType, err := p.ExpectIdentifier()
				if err != nil {
					return nil, err
				}
				param := &Param{
					Name: pName,
					Type: pType,
				}
				params = append(params, param)
			} else if tok.Type == parse.COMMA {
				//ignore
			} else {
				return nil, p.SyntaxError()
			}
		}
	} else {
		p.UngetToken()
		return params, nil
	}
	
}

func (gql *Extension) Validate(p *parse.Parser) error {
//	return fmt.Errorf("graphql.Extension.Validate NYI")
	fmt.Println("validate:", parse.Pretty(gql.Model))
	return nil
}

/*
			case "graphql":
				err = p.parseGraphqlDirective(comment)

func (p *Parser) parseGraphqlDirective(comment string) error {
	pathTemplate, err := p.expectString()
	if err != nil {
		return err
	}
	options, err := p.parseOptions("graphql", []string{})
	if err != nil {
		return err
	}
	gql := &sadl.GraphQLDef{
		Path:        pathTemplate,
		Annotations: options.Annotations,
		Comment: comment,
	}
	//parse the entity bindings. xxx
	tok := p.getToken()
	if tok == nil {
		return p.endOfFileError()
	}
	if tok.Type == OPEN_BRACE {
		gql.Comment = p.parseTrailingComment(gql.Comment)
		comment = ""
		for {
			done, comment, err := p.isBlockDone(comment)
			if done {
				gql.Comment = p.mergeComment(gql.Comment, comment)
				break
			}
//   var bindings []*sadl.GraphQLEntityBinding
//
//			err = p.parseGraphqlBindingSpec(op, comment, false)
			if err != nil {
				return err
			}
		}
	} else {
		p.ungetToken()
	}
	gql.Comment, err = p.EndOfStatement(gql.Comment)
	p.schema.GraphQL = append(p.schema.GraphQL, gql)
	return err
}

*/

//how this works:
//Problem: I dont want graphql stuff in the main parser. Extensions solve that
//Problem: I don't want the sadl2java generator to *assume* graphql. Currently, -graphql solves that. The code *always* has it

//the extension gets called when the top level directive is encountered. When first pass parsing is completed, it gets called
//again to validate/finalize. In both cases, it is passed the parser as an argument, and itself is the additional state.
//this keeps the parser clean, while implementing the extension in another package.
//this allows the extension to exist in a completely independent repo, in fact. This is extensible.

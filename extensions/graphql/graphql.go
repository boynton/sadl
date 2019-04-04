package graphql

import (
	"fmt"

	"github.com/boynton/sadl"
	"github.com/boynton/sadl/parse"
)

type Model struct {
	Path        string            `json:"path"`
	Comment     string            `json:"comment,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Actions  []*Action      `json:"actions,omitempty"`
}

type Action struct {
	Name        string            `json:"name"`
	Params      []*Param          `json:"params"`
	Return      *sadl.TypeSpec    `json:"return"`
	Provider    string            `json:"provider"`
	Comment     string            `json:"comment,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type Param struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func ParseFile(path string) (*sadl.Model, error) {
	p := NewExtension()
	model, err := parse.File(path, p)
	if err != nil {
		return nil, err
	}
	return model, nil
}

func NewExtension() *Extension {
	return &Extension{
		Model: &Model{},
	}
}

type Extension struct {
	Model *Model
}

func (gql *Extension) Name() string {
	return "graphql"
}

func (gql *Extension) Result() interface{} {
	return gql.Model
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
	return err

}

func (gql *Extension) parseQuerySpec(p *parse.Parser, comment string) error {
	qName, err := p.ExpectIdentifier()
	if err != nil {
		return err
	}
	params, err := gql.parseParams(p, qName)
	if err != nil {
		return err
	}
	ts, _, qcomment, err := p.ParseTypeSpec(comment)
	if err != nil {
		return err
	}
	options, err := p.ParseOptions("graphql", []string{"action"})
	if err != nil {
		return err
	}
	qcomment, err = p.EndOfStatement(qcomment)
	op := &Action{
		Name:     qName,
		Params:   params,
		Return:   ts,
		Provider: options.Action,
		Comment:  qcomment,
	}
	gql.Model.Actions = append(gql.Model.Actions, op)
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

func (gql *Extension) IsAction(opname string, p *parse.Parser) bool {
	for _, op := range p.Model().Actions {
		if op.Name == opname {
			return true
		}
	}
	return false
}

func (gql *Extension) Validate(p *parse.Parser) error {
	for _, op := range gql.Model.Actions {
		if !gql.IsAction(op.Provider, p) {
			return fmt.Errorf("GraphQL query action '%s' has an undefined HTTP action: %q", op.Name, op.Provider)
		}
	}
	return nil
}

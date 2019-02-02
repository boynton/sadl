package sadl

import(
	"fmt"
)

type ExtensionHandler interface {
	ParseExtension(p *Parser, extension string) error
}

func (p *Parser) AddExtension(name string, handler ExtensionHandler) error {
	if p.extensions == nil {
		p.extensions = make(map[string]ExtensionHandler, 0)
	}
	if _, ok := p.extensions[name]; ok {
		return fmt.Errorf("Extension already exists: %s", name)
	}
	p.extensions[name] = handler
	return nil
}

func (p *Parser) expectedDirectiveError() error {
	msg := "Expected one of 'type', 'name', 'namespace', 'version'"
	if p.extensions != nil {
		for k, _ := range p.extensions {
			msg = msg + fmt.Sprintf(" '%s'", k)
		}
	}
	return p.Error(msg)
}

func (p *Parser) parseExtensionDirective(comment string, extension string) error {
	if p.extensions != nil {
		if handler, ok := p.extensions[extension]; ok {
			p.currentComment = comment
			return handler.ParseExtension(p, extension)
		}
	}
	return p.expectedDirectiveError()
}


// Resolver SDUI: valida uma screen definition contra o catálogo fechado e resolve
// os placeholders {{variavel}} com os valores do payload de originação, produzindo
// o JSON resolvido — o artefato canônico que vai pro bucket. Semântica completa em
// screens/FORMAT.md. Somente stdlib.
package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ---------- Catálogo ----------

type Catalog struct {
	CatalogVersion string               `json:"catalogVersion"`
	Components     map[string]Component `json:"components"`
}

type Component struct {
	Description    string              `json:"description"`
	AllowsChildren bool                `json:"allowsChildren"`
	Props          map[string]PropSpec `json:"props"`
}

type PropSpec struct {
	Type        string              `json:"type"` // string | boolean | array | object
	Required    bool                `json:"required"`
	Enum        []string            `json:"enum,omitempty"`
	Items       *PropSpec           `json:"items,omitempty"`      // quando Type == array
	Properties  map[string]PropSpec `json:"properties,omitempty"` // quando Type == object
	Description string              `json:"description,omitempty"`
}

// ---------- Screen definition ----------

type ScreenDefinition struct {
	ScreenID       string             `json:"screenId"`
	ScreenVersion  string             `json:"screenVersion"`
	CatalogVersion string             `json:"catalogVersion"`
	Title          string             `json:"title,omitempty"`
	Variables      map[string]VarSpec `json:"variables"`
	Root           *Node              `json:"root"`
}

type VarSpec struct {
	Type        string `json:"type"` // string | boolean | array
	Required    *bool  `json:"required,omitempty"`
	Description string `json:"description,omitempty"`
}

func (v VarSpec) isRequired() bool { return v.Required == nil || *v.Required }

type Node struct {
	Type     string         `json:"type"`
	Props    map[string]any `json:"props"`
	Children []*Node        `json:"children,omitempty"`
}

// ResolvedScreen é o artefato final imutável salvo no bucket.
type ResolvedScreen struct {
	ScreenID       string `json:"screenId"`
	ScreenVersion  string `json:"screenVersion"`
	CatalogVersion string `json:"catalogVersion"`
	ResolvedAt     string `json:"resolvedAt"`
	Root           *Node  `json:"root"`
}

// ---------- Erros ----------

type ResolveError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

func (e ResolveError) Error() string { return e.Path + ": " + e.Message }

type errList struct{ errs []ResolveError }

func (l *errList) add(path, format string, args ...any) {
	l.errs = append(l.errs, ResolveError{Path: path, Message: fmt.Sprintf(format, args...)})
}

// ---------- Resolução ----------

var placeholderRe = regexp.MustCompile(`\{\{([a-zA-Z][a-zA-Z0-9_]*)\}\}`)

// ResolveScreen valida e resolve. Retorna o resolvido ou a lista de erros (fail-fast:
// qualquer erro impede a publicação da tela — nunca renderizar formalização com buraco).
func ResolveScreen(def *ScreenDefinition, payload map[string]any, catalog *Catalog) (*ResolvedScreen, []ResolveError) {
	l := &errList{}

	checkCatalogCompat(def.CatalogVersion, catalog.CatalogVersion, l)
	checkPayload(def.Variables, payload, l)
	if def.Root == nil {
		l.add("root", "screen definition sem root")
		return nil, l.errs
	}

	root := resolveNode(def.Root, def, payload, catalog, "root", l)
	if len(l.errs) > 0 {
		return nil, l.errs
	}
	return &ResolvedScreen{
		ScreenID:       def.ScreenID,
		ScreenVersion:  def.ScreenVersion,
		CatalogVersion: catalog.CatalogVersion,
		ResolvedAt:     time.Now().UTC().Format(time.RFC3339),
		Root:           root,
	}, nil
}

func checkCatalogCompat(screenVer, catalogVer string, l *errList) {
	major := func(v string) string { s, _, _ := strings.Cut(v, "."); return s }
	if screenVer == "" {
		l.add("catalogVersion", "screen definition sem catalogVersion")
		return
	}
	if major(screenVer) != major(catalogVer) {
		l.add("catalogVersion", "screen autorada contra catálogo %s incompatível com catálogo vigente %s (major diferente)", screenVer, catalogVer)
	}
}

// checkPayload valida o payload contra o bloco variables: obrigatória ausente,
// tipo errado e variável não declarada são erros (contrato estrito).
func checkPayload(vars map[string]VarSpec, payload map[string]any, l *errList) {
	for name, spec := range vars {
		val, ok := payload[name]
		if !ok {
			if spec.isRequired() {
				l.add("payload."+name, "variável obrigatória ausente no payload")
			}
			continue
		}
		if !matchesVarType(val, spec.Type) {
			l.add("payload."+name, "esperado %s, recebido %T", spec.Type, val)
		}
	}
	for name := range payload {
		if _, ok := vars[name]; !ok {
			l.add("payload."+name, "variável não declarada em variables")
		}
	}
}

func matchesVarType(v any, t string) bool {
	switch t {
	case "string":
		_, ok := v.(string)
		return ok
	case "boolean":
		_, ok := v.(bool)
		return ok
	case "array":
		_, ok := v.([]any)
		return ok
	}
	return false
}

func resolveNode(n *Node, def *ScreenDefinition, payload map[string]any, catalog *Catalog, path string, l *errList) *Node {
	comp, ok := catalog.Components[n.Type]
	if !ok {
		l.add(path, "componente %q fora do catálogo", n.Type)
		return nil
	}
	if len(n.Children) > 0 && !comp.AllowsChildren {
		l.add(path, "componente %q não aceita children", n.Type)
	}

	out := &Node{Type: n.Type, Props: map[string]any{}}

	for name, raw := range n.Props {
		propPath := path + ".props." + name
		spec, known := comp.Props[name]
		if !known {
			l.add(propPath, "prop desconhecida para %q", n.Type)
			continue
		}
		val, omit := resolveValue(raw, def.Variables, payload, propPath, false, l)
		if omit {
			if spec.Required {
				l.add(propPath, "placeholder integral de variável opcional ausente, mas a prop é obrigatória")
			}
			continue
		}
		validatePropValue(val, spec, propPath, l)
		out.Props[name] = val
	}
	for name, spec := range comp.Props {
		if _, present := n.Props[name]; spec.Required && !present {
			l.add(path+".props."+name, "prop obrigatória ausente")
		}
	}

	for i, child := range n.Children {
		if c := resolveNode(child, def, payload, catalog, fmt.Sprintf("%s.children[%d]", path, i), l); c != nil {
			out.Children = append(out.Children, c)
		}
	}
	return out
}

// resolveValue aplica a semântica de placeholders (screens/FORMAT.md):
// string integral "{{x}}" -> valor com tipo original (omit=true se opcional ausente,
// permitido só no nível de prop — insideCollection marca posições aninhadas);
// interpolação parcial -> só variáveis string (opcional ausente vira "").
func resolveValue(v any, vars map[string]VarSpec, payload map[string]any, path string, insideCollection bool, l *errList) (out any, omit bool) {
	switch val := v.(type) {
	case string:
		return resolveString(val, vars, payload, path, insideCollection, l)
	case map[string]any:
		m := map[string]any{}
		for k, inner := range val {
			r, om := resolveValue(inner, vars, payload, path+"."+k, true, l)
			if !om {
				m[k] = r
			}
		}
		return m, false
	case []any:
		s := make([]any, 0, len(val))
		for i, inner := range val {
			r, om := resolveValue(inner, vars, payload, fmt.Sprintf("%s[%d]", path, i), true, l)
			if !om {
				s = append(s, r)
			}
		}
		return s, false
	default: // bool, float64, nil — literais passam intactos
		return v, false
	}
}

func resolveString(s string, vars map[string]VarSpec, payload map[string]any, path string, insideCollection bool, l *errList) (any, bool) {
	matches := placeholderRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return s, false
	}

	// Placeholder integral: a string é exatamente {{x}} — substitui com tipo original.
	if len(matches) == 1 && s == matches[0][0] {
		name := matches[0][1]
		spec, declared := vars[name]
		if !declared {
			l.add(path, "placeholder {{%s}} referencia variável não declarada", name)
			return s, false
		}
		val, present := payload[name]
		if !present {
			if spec.isRequired() {
				l.add(path, "variável obrigatória {{%s}} ausente no payload", name)
				return s, false
			}
			if insideCollection {
				l.add(path, "placeholder integral opcional {{%s}} ausente dentro de estrutura aninhada — omissão só é válida no nível de prop", name)
				return s, false
			}
			return nil, true
		}
		return val, false
	}

	// Interpolação parcial: toda variável referenciada deve ser string.
	result := placeholderRe.ReplaceAllStringFunc(s, func(m string) string {
		name := placeholderRe.FindStringSubmatch(m)[1]
		spec, declared := vars[name]
		if !declared {
			l.add(path, "placeholder {{%s}} referencia variável não declarada", name)
			return m
		}
		if spec.Type != "string" {
			l.add(path, "interpolação parcial exige variável string; {{%s}} é %s", name, spec.Type)
			return m
		}
		val, present := payload[name]
		if !present {
			if spec.isRequired() {
				l.add(path, "variável obrigatória {{%s}} ausente no payload", name)
				return m
			}
			return ""
		}
		str, _ := val.(string)
		return str
	})
	return result, false
}

// validatePropValue confere o valor resolvido contra o PropSpec do catálogo.
func validatePropValue(v any, spec PropSpec, path string, l *errList) {
	switch spec.Type {
	case "string":
		s, ok := v.(string)
		if !ok {
			l.add(path, "esperado string, recebido %T", v)
			return
		}
		if len(spec.Enum) > 0 && !contains(spec.Enum, s) {
			l.add(path, "valor %q fora do enum %v", s, spec.Enum)
		}
	case "boolean":
		if _, ok := v.(bool); !ok {
			l.add(path, "esperado boolean, recebido %T", v)
		}
	case "array":
		arr, ok := v.([]any)
		if !ok {
			l.add(path, "esperado array, recebido %T", v)
			return
		}
		if spec.Items != nil {
			for i, item := range arr {
				validatePropValue(item, *spec.Items, fmt.Sprintf("%s[%d]", path, i), l)
			}
		}
	case "object":
		obj, ok := v.(map[string]any)
		if !ok {
			l.add(path, "esperado object, recebido %T", v)
			return
		}
		for name, sub := range spec.Properties {
			val, present := obj[name]
			if !present {
				if sub.Required {
					l.add(path+"."+name, "campo obrigatório ausente")
				}
				continue
			}
			validatePropValue(val, sub, path+"."+name, l)
		}
		for name := range obj {
			if _, known := spec.Properties[name]; !known {
				l.add(path+"."+name, "campo desconhecido")
			}
		}
	default:
		l.add(path, "PropSpec com type desconhecido %q no catálogo", spec.Type)
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// ---------- Parsing ----------

func ParseCatalog(data []byte) (*Catalog, error) {
	var c Catalog
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("catálogo inválido: %w", err)
	}
	return &c, nil
}

func ParseScreenDefinition(data []byte) (*ScreenDefinition, error) {
	var d ScreenDefinition
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("screen definition inválida: %w", err)
	}
	return &d, nil
}

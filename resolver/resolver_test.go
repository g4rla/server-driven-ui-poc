package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func loadCatalog(t *testing.T) *Catalog {
	t.Helper()
	raw, err := os.ReadFile("../catalog/schema.json")
	if err != nil {
		t.Fatal(err)
	}
	c, err := ParseCatalog(raw)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func loadScreen(t *testing.T, path string) *ScreenDefinition {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	d, err := ParseScreenDefinition(raw)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func consorcioPayload() map[string]any {
	return map[string]any{
		"nomeCliente":     "Maria Silva",
		"grupoCota":       "Grupo 1234 / Cota 56",
		"valorCarta":      "R$ 80.000,00",
		"prazoMeses":      "180 meses",
		"parcelaMensal":   "R$ 612,34",
		"clausulasResumo": []any{"Cláusula 1 — objeto", "Cláusula 2 — contemplação"},
	}
}

func cartaoPayload() map[string]any {
	return map[string]any{
		"nomeCliente":            "João Souza",
		"nomeProduto":            "Cartão Platinum",
		"limiteAprovado":         "R$ 12.000,00",
		"anuidade":               "12x R$ 49,90",
		"cetAnual":               "312,4% a.a.",
		"beneficios":             []any{"Sala VIP", "Cashback 1%"},
		"aceitaDebitoAutomatico": true,
	}
}

func mustResolve(t *testing.T, def *ScreenDefinition, payload map[string]any) *ResolvedScreen {
	t.Helper()
	resolved, errs := ResolveScreen(def, payload, loadCatalog(t))
	if len(errs) > 0 {
		t.Fatalf("esperava sucesso, obteve erros: %v", errs)
	}
	return resolved
}

func expectError(t *testing.T, def *ScreenDefinition, payload map[string]any, fragment string) {
	t.Helper()
	_, errs := ResolveScreen(def, payload, loadCatalog(t))
	for _, e := range errs {
		if strings.Contains(e.Error(), fragment) {
			return
		}
	}
	t.Fatalf("esperava erro contendo %q, obteve: %v", fragment, errs)
}

// ---------- Casos felizes com as fixtures reais ----------

func TestResolveConsorcio(t *testing.T) {
	resolved := mustResolve(t, loadScreen(t, "../screens/example-consorcio.json"), consorcioPayload())

	text := resolved.Root.Children[0]
	if got := text.Props["content"]; got != "Olá, Maria Silva. Revise as condições abaixo antes de confirmar a sua adesão." {
		t.Errorf("interpolação parcial errada: %v", got)
	}
	// Placeholder integral de array dentro do Accordion > List.
	list := resolved.Root.Children[2].Children[0]
	items, ok := list.Props["items"].([]any)
	if !ok || len(items) != 2 || items[0] != "Cláusula 1 — objeto" {
		t.Errorf("placeholder integral de array errado: %v", list.Props["items"])
	}
	// Interpolação dentro de estrutura aninhada (rows da KeyValueTable).
	rows := resolved.Root.Children[1].Props["rows"].([]any)
	if rows[1].(map[string]any)["value"] != "R$ 80.000,00" {
		t.Errorf("placeholder em rows errado: %v", rows[1])
	}
}

func TestResolveConsorcioDetalhes(t *testing.T) {
	payload := map[string]any{
		"grupoCota":            "Grupo 1234 / Cota 56",
		"taxaAdministracao":    "18% total",
		"fundoReserva":         "2%",
		"condicoesLance":       []any{"Lance livre mensal"},
		"condicoesDesistencia": []any{"Devolução após encerramento do grupo"},
	}
	resolved := mustResolve(t, loadScreen(t, "../screens/example-consorcio-detalhes.json"), payload)
	if got := resolved.Root.Props["title"]; got != "Detalhes do contrato — grupo Grupo 1234 / Cota 56" {
		t.Errorf("título interpolado errado: %v", got)
	}
}

func TestResolveCartaoComBooleanIntegral(t *testing.T) {
	resolved := mustResolve(t, loadScreen(t, "../screens/example-cartao.json"), cartaoPayload())
	accordion := resolved.Root.Children[2]
	if got := accordion.Props["defaultOpen"]; got != true {
		t.Errorf("placeholder integral boolean errado: %v (%T)", got, got)
	}
}

func TestResolveCartaoVariavelOpcionalAusenteOmiteProp(t *testing.T) {
	payload := cartaoPayload()
	delete(payload, "aceitaDebitoAutomatico")
	resolved := mustResolve(t, loadScreen(t, "../screens/example-cartao.json"), payload)
	accordion := resolved.Root.Children[2]
	if _, present := accordion.Props["defaultOpen"]; present {
		t.Errorf("prop de variável opcional ausente deveria ser omitida: %v", accordion.Props)
	}
}

// ---------- Casos de erro ----------

func TestErroComponenteForaDoCatalogo(t *testing.T) {
	def := &ScreenDefinition{
		ScreenID: "x", ScreenVersion: "1.0.0", CatalogVersion: "1.1.0",
		Root: &Node{Type: "Iframe", Props: map[string]any{}},
	}
	expectError(t, def, map[string]any{}, `componente "Iframe" fora do catálogo`)
}

func TestErroVariavelObrigatoriaAusente(t *testing.T) {
	payload := consorcioPayload()
	delete(payload, "nomeCliente")
	expectError(t, loadScreen(t, "../screens/example-consorcio.json"), payload, "variável obrigatória ausente no payload")
}

func TestErroPropObrigatoriaFaltando(t *testing.T) {
	def := &ScreenDefinition{
		ScreenID: "x", ScreenVersion: "1.0.0", CatalogVersion: "1.1.0",
		Root: &Node{Type: "Text", Props: map[string]any{"variant": "body"}},
	}
	expectError(t, def, map[string]any{}, "prop obrigatória ausente")
}

func TestErroPropDesconhecida(t *testing.T) {
	def := &ScreenDefinition{
		ScreenID: "x", ScreenVersion: "1.0.0", CatalogVersion: "1.1.0",
		Root: &Node{Type: "Text", Props: map[string]any{"content": "oi", "onClick": "hack()"}},
	}
	expectError(t, def, map[string]any{}, "prop desconhecida")
}

func TestErroEnumInvalido(t *testing.T) {
	def := &ScreenDefinition{
		ScreenID: "x", ScreenVersion: "1.0.0", CatalogVersion: "1.1.0",
		Root: &Node{Type: "Text", Props: map[string]any{"content": "oi", "variant": "gigante"}},
	}
	expectError(t, def, map[string]any{}, "fora do enum")
}

func TestErroPlaceholderNaoDeclarado(t *testing.T) {
	def := &ScreenDefinition{
		ScreenID: "x", ScreenVersion: "1.0.0", CatalogVersion: "1.1.0",
		Root: &Node{Type: "Text", Props: map[string]any{"content": "{{fantasma}}"}},
	}
	expectError(t, def, map[string]any{}, "variável não declarada")
}

func TestErroPayloadComVariavelNaoDeclarada(t *testing.T) {
	payload := consorcioPayload()
	payload["extra"] = "x"
	expectError(t, loadScreen(t, "../screens/example-consorcio.json"), payload, "variável não declarada em variables")
}

func TestErroPayloadTipoErrado(t *testing.T) {
	payload := consorcioPayload()
	payload["clausulasResumo"] = "não sou array"
	expectError(t, loadScreen(t, "../screens/example-consorcio.json"), payload, "esperado array")
}

func TestErroInterpolacaoParcialComVariavelNaoString(t *testing.T) {
	def := &ScreenDefinition{
		ScreenID: "x", ScreenVersion: "1.0.0", CatalogVersion: "1.1.0",
		Variables: map[string]VarSpec{"lista": {Type: "array"}},
		Root:      &Node{Type: "Text", Props: map[string]any{"content": "itens: {{lista}}!"}},
	}
	expectError(t, def, map[string]any{"lista": []any{"a"}}, "interpolação parcial exige variável string")
}

func TestErroChildrenNaoPermitido(t *testing.T) {
	def := &ScreenDefinition{
		ScreenID: "x", ScreenVersion: "1.0.0", CatalogVersion: "1.1.0",
		Root: &Node{Type: "Text", Props: map[string]any{"content": "oi"},
			Children: []*Node{{Type: "Text", Props: map[string]any{"content": "filho"}}}},
	}
	expectError(t, def, map[string]any{}, "não aceita children")
}

func TestErroCatalogoMajorIncompativel(t *testing.T) {
	def := &ScreenDefinition{
		ScreenID: "x", ScreenVersion: "1.0.0", CatalogVersion: "2.0.0",
		Root: &Node{Type: "Section", Props: map[string]any{}},
	}
	expectError(t, def, map[string]any{}, "major diferente")
}

func TestErroActionMalformada(t *testing.T) {
	def := &ScreenDefinition{
		ScreenID: "x", ScreenVersion: "1.0.0", CatalogVersion: "1.1.0",
		Root: &Node{Type: "Button", Props: map[string]any{
			"label":  "Ir",
			"action": map[string]any{"type": "navigate", "url": "https://evil.example"},
		}},
	}
	expectError(t, def, map[string]any{}, "campo desconhecido")
}

// O JSON resolvido deve ser serializável e carregar o catalogVersion vigente.
func TestResolvedSerializaComContexto(t *testing.T) {
	resolved := mustResolve(t, loadScreen(t, "../screens/example-cartao.json"), cartaoPayload())
	out, err := json.Marshal(resolved)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"catalogVersion":"1.1.0"`) {
		t.Errorf("resolvido sem catalogVersion: %s", out)
	}
}

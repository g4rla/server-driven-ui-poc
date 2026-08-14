# Server-Driven UI — POC

POC de migração de telas de formalização/assinatura de contratos de **HTML renderizado no
backend** (frágil: XSS, drift, sem design system) para **Server-Driven UI**: o backend envia
um JSON declarativo (árvore de componentes) validado contra um **catálogo fechado**, e o
front tem um **renderer único** que interpreta esse JSON com componentes reais.

```
originação ──payload──▶ resolver (Go/Lambda) ──JSON resolvido──▶ bucket ──▶ renderer (React) ──eventos──▶ host ──▶ APIs
                              ▲                                                    ▲
                        catalog/schema.json                              mesmo catálogo (registry)
```

## Estrutura

| Caminho | O que é |
|---|---|
| `catalog/schema.json` | Catálogo fechado de componentes, props tipadas (fonte da verdade) |
| `screens/schema.json` | JSON Schema do envelope de screen definition |
| `screens/FORMAT.md` | Semântica de placeholders `{{var}}`, ações, versionamento |
| `screens/example-*.json` | Screen definitions de exemplo (2 produtos + tela secundária) |
| `docs/host-contract.md` | Contrato do host: gating de aceite, eventos `SduiEvent`, mock HTTP |
| `docs/adoption-guide.md` | **Insumos pra implementação real em ambiente corporativo** |
| `resolver/` | Resolver Go (stdlib only): valida + resolve placeholders + persiste |
| `renderer/` | Renderer React/TS puro + `DemoHost` (estado, navegação, HTTP) |
| `skills/sdui-authoring/` | Skill de autoria assistida por IA (template pra acoplar ao DS real) |

## Como rodar

Pré-requisitos: Go ≥ 1.22 e Node ≥ 18.

**Resolver (testes + CLI simulando o Lambda):**

```bash
cd resolver
go test ./...            # 17 testes: casos felizes com as fixtures + casos de erro
go run . -screen ../screens/example-cartao.json \
         -payload payload.json \            # objeto JSON com as variáveis declaradas
         -catalog ../catalog/schema.json -out ./out
# gera out/cartao-adesao@1.0.0.resolved.json — o artefato canônico
```

**Renderer (typecheck; é uma lib, não um app):**

```bash
cd renderer
npm install
npm run typecheck
```

Para ver na tela, monte o `DemoHost` em qualquer app React (Vite etc.):

```tsx
<DemoHost
  screens={{ "cartao-adesao": resolvedJson }}   // saída do resolver
  initialScreenId="cartao-adesao"
  eventsEndpoint="https://httpbin.org/post"     // mock; troque pela API real
/>
```

## Invariantes de segurança (não negociar)

1. O JSON nunca carrega HTML, URLs, expressões ou código — só tipos do catálogo + dados.
2. Resolver rejeita: tipo fora do catálogo, prop desconhecida, action com campo extra,
   variável ausente/não declarada/de tipo errado. Fail-fast: erro = tela não publica.
3. Renderer nunca usa `dangerouslySetInnerHTML`; conteúdo é sempre texto puro.
4. Comportamento (gating, navegação, HTTP) vive no host, nunca no JSON.

## Como plugar o catálogo/design system real

Resumo (detalhes em `docs/adoption-guide.md`):

1. Substitua os componentes de `renderer/components/` pelos componentes do design system
   real, mantendo a interface `{ node: ScreenNode }` e o dispatch do registry.
2. Evolua `catalog/schema.json` de forma aditiva (minor); breaking = major + migração.
3. Troque o entrypoint do resolver: `main.go` → handler `aws-lambda-go`; `LocalDirStore`
   → implementação S3 da interface `ScreenStore`. `resolver.go` não muda.
4. Troque `eventsEndpoint` do host pelo endpoint real de telemetria/formalização.
5. Alimente o agente de autoria com `skills/sdui-authoring/` + docs do design system.

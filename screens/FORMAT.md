# Formato de screen definition (SDUI)

Uma screen definition é um JSON declarativo validado por `screens/schema.json` (envelope/árvore)
e pelo resolver contra `catalog/schema.json` (tipos e props). Definições publicadas são
**imutáveis** — mudou, versiona (`screenVersion`).

## Envelope

```json
{
  "screenId": "consorcio-resumo",
  "screenVersion": "1.0.0",
  "catalogVersion": "1.1.0",
  "title": "Resumo da contratação — Consórcio",
  "variables": {
    "nomeCliente":   { "type": "string",  "description": "Nome completo do cliente" },
    "valorCarta":    { "type": "string",  "description": "Valor da carta de crédito, já formatado" },
    "clausulas":     { "type": "array",   "description": "Lista de cláusulas resumidas" },
    "aceiteDigital": { "type": "boolean", "required": false }
  },
  "root": { "type": "Section", "props": {}, "children": [ ... ] }
}
```

- `variables` declara o contrato com o payload de originação. O resolver rejeita payload com
  variável obrigatória ausente ou de tipo errado, e rejeita definição que usa placeholder de
  variável não declarada.
- `catalogVersion` fica registrado também no JSON **resolvido**, que é o artefato canônico
  congelado no bucket — garante reprodutibilidade (tela e evidência legal) anos depois.

## Semântica de placeholders

Placeholders só existem em **valores string** da árvore, na forma `{{nomeDaVariavel}}`.

1. **Interpolação parcial** — `"Olá, {{nomeCliente}}"`: a variável deve ser `string`;
   o valor é concatenado. Variável não-string aqui é **erro de resolução**.
2. **Placeholder integral** — a string inteira é exatamente `"{{clausulas}}"`: o valor da
   variável substitui o nó de valor **com seu tipo original** (string, boolean ou array).
   É assim que se injeta `items` de uma `List` ou `defaultOpen` de um `Accordion`.
3. **Variável ausente** — placeholder referenciando variável obrigatória sem valor no payload
   é **erro de resolução** (fail-fast; nunca renderizar tela de formalização com buraco).
   Variável declarada `required: false` ausente: interpolação parcial vira string vazia;
   placeholder integral vira omissão da prop (válido só se a prop for opcional no catálogo).
4. **Sem escape/HTML** — valores são dados, nunca markup. O renderer sempre trata `content`,
   `label` etc. como texto puro (nada de `dangerouslySetInnerHTML`). O resolver ainda valida
   tipo e tamanho dos valores do payload, mas não faz escaping — não há mais HTML no pipeline.
5. Não há expressões, condicionais ou loops — só substituição. Lógica de variação de tela é
   resolvida antes (o serviço de originação escolhe `screenId`/variáveis), não no template.

## Ações e navegação

Botões carregam um objeto `action` **semântico** — o SDUI descreve intenção; quem executa é o
host (app que embute o renderer). Nenhuma URL ou código trafega na definição.

| `action.type` | Campos            | Comportamento do host |
|---------------|-------------------|-----------------------|
| `navigate`    | `screenId`, `screenVersion?` | Busca a screen resolvida indicada e a renderiza, empilhando a atual na pilha de navegação. |
| `back`        | —                 | Volta para a screen anterior da pilha (estado de checkboxes preservado pelo host). |
| `submit`      | `intent`: `accept` \| `decline` | Finaliza a jornada. Antes de `accept`, o host valida que todo `Checkbox` com `required: true` da screen atual está marcado. |

Exemplo de rodapé de aceite:

```json
{
  "type": "Section",
  "props": {},
  "children": [
    { "type": "Checkbox", "props": { "id": "aceite-termos", "label": "Li e aceito os termos", "required": true } },
    { "type": "Button", "props": { "label": "Ver detalhes", "variant": "secondary",
        "action": { "type": "navigate", "screenId": "consorcio-detalhes" } } },
    { "type": "Button", "props": { "label": "Recusar", "variant": "danger",
        "action": { "type": "submit", "intent": "decline" } } },
    { "type": "Button", "props": { "label": "Aceitar e assinar", "variant": "primary",
        "action": { "type": "submit", "intent": "accept" } } }
  ]
}
```

Numa tela secundária (ex.: `consorcio-detalhes`), o retorno usa
`{ "type": "back" }` — a tela não precisa saber quem a chamou.

O comportamento do host (estado de checkboxes, gating do botão de aceite, shape dos eventos
emitidos pelo renderer e envio deles ao backend) está especificado em
[`docs/host-contract.md`](../docs/host-contract.md).

## Compatibilidade catálogo × screen

- Catálogo evolui de forma **aditiva** (novo componente, nova prop opcional) → minor bump;
  screens publicadas continuam válidas.
- Remoção/mudança de semântica → major bump do catálogo + migração explícita das screens.
- O resolver valida a screen contra o catálogo vigente e falha se `catalogVersion` da screen
  tiver major diferente do catálogo carregado.

# Guia do catálogo e dos schemas

Este documento explica **o que é cada arquivo de schema do repo, quem consome cada um,
e o passo a passo para evoluí-los** sem quebrar telas publicadas.

## Mapa: o que é o quê

| Arquivo | O que é | Quem consome |
|---|---|---|
| `catalog/schema.json` | **Catálogo de componentes**: a lista fechada (whitelist) de tipos que podem aparecer numa tela, com as props tipadas de cada um. É a fronteira de segurança do SDUI: tipo fora daqui é rejeitado. | Resolver (validação), renderer (1 componente React por tipo), skill de autoria (contexto pra IA). |
| `screens/schema.json` | **JSON Schema do envelope** de uma screen definition: campos obrigatórios (`screenId`, `screenVersion`, `catalogVersion`, `root`), formato do bloco `variables` e shape recursivo dos nós (`type`/`props`/`children`). Não sabe nada sobre componentes específicos. | Qualquer validador JSON Schema (CI, IDE, playground de autoria). |
| `screens/FORMAT.md` | Semântica que schema não expressa: placeholders `{{var}}`, ações de botão, regra de compatibilidade catálogo × screen. | Humanos e a skill de autoria. |
| `screens/example-*.json` | Fixtures: screen definitions reais usadas nos testes do resolver e no demo. | Testes, demo. |

A divisão de responsabilidade é em **duas camadas**:

1. `screens/schema.json` valida a **forma** do documento (envelope + árvore genérica).
2. O **resolver** (`resolver/resolver.go`) valida o **conteúdo** contra o catálogo:
   tipo existe, props conhecidas/obrigatórias/tipadas, `allowsChildren`, enums,
   variáveis declaradas × payload.

Ou seja: o catálogo **não** é um JSON Schema padrão — é um formato próprio, simples,
interpretado pelo resolver (structs `Catalog`/`Component`/`PropSpec` em `resolver/resolver.go`).

## Anatomia de uma entrada do catálogo

```json
"Checkbox": {
  "description": "…",
  "allowsChildren": false,
  "props": {
    "id":       { "type": "string",  "required": true },
    "label":    { "type": "string",  "required": true },
    "required": { "type": "boolean", "required": false }
  }
}
```

Campos de uma **PropSpec** (o que o resolver entende hoje):

| Campo | Significado |
|---|---|
| `type` | `string` \| `boolean` \| `array` \| `object`. |
| `required` | Se `true`, o resolver falha quando a prop está ausente na tela. |
| `enum` | (só `string`) valores permitidos; valor fora do enum é erro de resolução. |
| `items` | (só `array`) PropSpec de cada item — recursivo. |
| `properties` | (só `object`) PropSpec de cada campo — recursivo. Ex.: a `action` do Button. |
| `description` | Documentação para humanos e para a IA de autoria. Não é validada. |

`allowsChildren` diz se o nó pode ter `children` (contêineres como `Section`/`Accordion`).
Componente folha com `children` na tela é erro de resolução.

O resolver é **estrito nos dois sentidos**: prop desconhecida é erro (não é ignorada),
e prop obrigatória ausente é erro. Isso é proposital — tela de formalização não renderiza
"quase certa".

## Onde cada validação acontece (pipeline)

```
autoria ──▶ screens/schema.json (forma do envelope, opcional em CI/IDE)
        ──▶ resolver: checkCatalogCompat  (major do catalogVersion bate?)
        ──▶ resolver: checkPayload        (payload × bloco variables: ausente, tipo, não declarada)
        ──▶ resolver: resolveNode         (tipo no catálogo, props, enums, children, placeholders)
        ──▶ JSON resolvido imutável no bucket (carrega catalogVersion + resolvedAt)
front  ──▶ SduiRenderer: registry por type (renderer/SduiRenderer.tsx)
```

O renderer confia no JSON resolvido (já validado); o `registry` só precisa ter um
componente React para cada tipo do catálogo.

## Como evoluir

Regra geral de versionamento do `catalogVersion` (semver):

- **Aditivo** (novo componente, nova prop **opcional**, novo valor de enum) → **minor**.
  Telas publicadas continuam válidas sem tocar em nada.
- **Breaking** (remover componente/prop, tornar prop obrigatória, mudar tipo ou semântica,
  remover valor de enum) → **major** + migrar todas as screen definitions publicadas.
  O resolver rejeita tela cujo `catalogVersion` tenha major diferente do catálogo carregado.
- Patch: só correção de `description`/documentação.

Telas publicadas são **imutáveis**: qualquer mudança numa screen definition gera novo
`screenVersion`. O JSON resolvido no bucket nunca é reescrito (é evidência legal).

### Receita: adicionar um componente novo (ex.: `Image`)

1. **Catálogo**: nova entrada em `catalog/schema.json` com `description`, `allowsChildren`
   e props tipadas. Bump **minor** do `catalogVersion`.
   Atenção às decisões fixadas: nada de URL arbitrária/HTML/código em prop — se o componente
   precisa de recurso externo, referencie por id semântico que o host resolve
   (ex.: `assetId`, não `src`).
2. **Renderer**: criar `renderer/components/Image.tsx` e registrá-lo no `registry` de
   `renderer/SduiRenderer.tsx`. Componente puro: lê `node.props`, emite `onEvent` se for
   interativo, zero fetch/estado global.
3. **Testes do resolver**: normalmente **nenhuma mudança em `resolver.go`** — ele é genérico,
   dirigido pelo catálogo. Adicione um caso em `resolver/resolver_test.go` cobrindo o
   componente novo (feliz + prop obrigatória faltando) só se ele tiver forma nova
   (ex.: objeto aninhado tipo `action`).
4. **Fixture**: usar o componente em um `screens/example-*.json` (com bump de `screenVersion`)
   pra ele ser exercitado ponta a ponta.
5. **Skill de autoria**: atualizar `skills/sdui-authoring/` se ela lista componentes.
6. Só depois de tudo mergeado o componente pode aparecer em tela publicada — o deploy do
   catálogo novo (resolver) e do renderer novo precisa preceder a primeira tela que o usa.

### Receita: adicionar uma prop opcional (ex.: `Text.align`)

1. Adicionar a PropSpec com `"required": false` (e `enum` se for um conjunto fechado).
2. Bump minor do catálogo.
3. Tratar a prop no componente React correspondente, **com default sensato quando ausente**
   (telas antigas não têm a prop).

### Receita: mudança breaking

1. Bump **major** do catálogo.
2. Migrar todas as screen definitions publicadas (novo `screenVersion` + novo
   `catalogVersion` em cada uma) — o resolver com o catálogo novo rejeita as antigas por
   major incompatível, então a migração é forçada, não silenciosa.
3. Janela de transição: manter o resolver/catálogo antigos servindo as telas antigas até a
   migração completar (feature flag por jornada, como no rollout do passo 6 do roadmap).
4. Evite: quase toda breaking pode virar aditiva (novo componente `TextV2`/nova prop opcional
   com deprecação documentada) — major deve ser raro.

### Receita: evoluir o formato do envelope (`screens/schema.json`)

Muito mais raro. Campo novo **opcional** no envelope: adicionar no schema (`properties` +
manter `additionalProperties: false`) e, se o resolver precisar dele, no struct
`ScreenDefinition`. Campo obrigatório novo ou novo tipo em `variables.type` é breaking do
formato → tratar como major do ecossistema (schema + resolver + todas as telas).

### Novo tipo de `action` de Button (ex.: `openDocument`)

`action` é o ponto de contato mais sensível, porque envolve **três** lugares:

1. `catalog/schema.json`: novo valor no `enum` de `action.type` + campos novos opcionais
   (minor).
2. `docs/host-contract.md`: definir o que o host faz com a ação e o shape do `SduiEvent`.
3. Renderer/DemoHost: emitir e tratar o evento novo.

Host antigo que recebe ação desconhecida deve falhar de forma explícita (não engolir) —
por isso a mesma regra de deploy: host primeiro, tela depois.

## Checklist rápido de PR que toca o catálogo

- [ ] `catalogVersion` bumpado com a regra semver acima?
- [ ] `description` da prop/componente escrita pensando na IA de autoria (ela lê isso)?
- [ ] Nenhuma prop carrega URL, HTML, expressão ou código?
- [ ] Renderer atualizado (`registry` + componente) na mesma mudança?
- [ ] Fixture em `screens/example-*.json` exercita o que mudou?
- [ ] `go test ./...` no resolver passando?
- [ ] Se breaking: plano de migração das telas publicadas descrito no PR?

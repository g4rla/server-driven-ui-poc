# Guia de adoção: da POC ao ambiente corporativo

Insumos para implementar a solução real a partir desta POC. Organizado por frente de
trabalho; cada seção diz **o que a POC já resolve**, **o que falta** e **decisões a tomar
com o contexto da empresa**.

## 1. Catálogo × design system real

O que a POC dá: formato do catálogo (`catalog/schema.json`) com props tipadas, enums,
shapes aninhados; regra de evolução (aditivo = minor, breaking = major + migração).

Para o mundo real:
- Mapeie cada componente do catálogo para um componente **existente** do design system da
  empresa. O catálogo deve ser um **subconjunto** do DS — nunca componentes novos criados
  só pro SDUI (isso reintroduz drift visual).
- O catálogo é a fonte da verdade compartilhada entre resolver, renderer e agente de
  autoria. Publique-o como artefato versionado (pacote npm interno + módulo Go, gerados do
  mesmo JSON) para os três consumirem a mesma versão.
- Governança: mudança no catálogo passa por revisão do time de plataforma + design system.
  Um PR de catálogo que só adiciona prop opcional pode ter fluxo leve; breaking exige RFC.

## 2. Resolver → Lambda de produção

O que a POC dá: `resolver/resolver.go` completo (validação + placeholders + erros com
path) e a interface `ScreenStore` que isola a persistência.

Para o mundo real:
- Trocar `main.go` por handler `aws-lambda-go` (`lambda.Start`); o evento de entrada vem
  do serviço de originação (mesmo trigger de hoje). `resolver.go` não muda.
- Implementar `ScreenStore` com S3 `PutObject`, chave `{screenId}@{screenVersion}/{idempotencyKey}.resolved.json`.
  **O JSON resolvido é imutável** — write-once (bucket com object lock ou política de negação
  de overwrite) porque ele é a base da evidência legal.
- Screen definitions publicadas ficam num repositório versionado (git ou tabela com
  status `draft`/`published`); o Lambda só resolve definitions `published`.
- Observabilidade: os erros de resolução já saem estruturados (`path` + `message`) —
  emita-os como métricas/logs; erro de resolução em produção é alarme, não fallback.

## 3. Renderer × app real

O que a POC dá: renderer puro com registry fechado, contrato `onEvent(SduiEvent)`,
`DemoHost` de referência com gating/navegação/HTTP.

Para o mundo real:
- Substituir os componentes de `renderer/components/` pelos do design system, mantendo a
  interface `{ node: ScreenNode }` e o registry. O `SduiRenderer.tsx` não muda.
- Reescrever o `DemoHost` como host de produção: mesmas responsabilidades (estado de
  checkboxes, gating, pilha de navegação, tradução de eventos), trocando o `fetch` pro
  cliente HTTP/telemetria da empresa. `docs/host-contract.md` é a spec; o `DemoHost.tsx`
  é a implementação de referência.
- O playground de autoria usa o **mesmo pacote do renderer** com um host fake que só loga
  eventos — nunca um segundo renderer.

## 4. Migração incremental (rollout)

1. Escolher a jornada piloto mais simples e migrar o template manualmente (teste de
   expressividade do catálogo — se não couber, evoluir o catálogo antes de escalar).
2. Rodar em paralelo por feature flag por jornada: mesmo payload gera HTML legado e JSON
   SDUI; comparar visualmente/QA antes de cortar tráfego.
3. Migrar produto a produto. O legado só desliga quando a evidência legal nova (item 5)
   estiver aprovada por compliance.

## 5. Evidência legal

- Artefato canônico: JSON resolvido imutável no bucket + `screenId/screenVersion/catalogVersion`.
- O evento `submit` (`acceptances` + contexto) é o registro de "cliente viu a tela X@Y e
  marcou os aceites Z" — persistir com trilha auditável e correlacionar com o JSON.
- PDF/visualização para auditoria é **derivado** do JSON resolvido (renderer de PDF usando
  o mesmo catálogo), nunca um artefato paralelo mantido à mão.
- Validar com jurídico/compliance da empresa **antes** de desligar o HTML congelado legado.

## 6. Autoria assistida por IA

O que a POC dá: `skills/sdui-authoring/SKILL.md` — skill pronta pra acoplar a um agente
(Claude Code, agente interno etc.), com o loop autoria → validação → correção.

Para o mundo real:
- Alimentar o agente com: o catálogo vigente, `screens/FORMAT.md`, exemplos publicados e a
  **documentação do design system da empresa** (guidelines de conteúdo, tom de voz, quando
  usar cada componente) — a skill tem placeholders marcados com `TODO(empresa)` pra isso.
- O playground valida com o resolver real (mesmo binário/lib) e renderiza com o renderer
  real — o agente recebe os erros com `path` e corrige em loop.
- Gate humano obrigatório: agente produz `draft`; só compliance/jurídico promove a
  `published`. A validação de schema garante segurança estrutural, não correção jurídica.

## 7. Checklist de kickoff no ambiente corporativo

- [ ] Inventário: quais componentes do DS real cobrem o catálogo v1.1.0? Gaps?
- [ ] Definir repositório/registro de screen definitions e fluxo draft→published
- [ ] Bucket write-once + convenção de chaves do JSON resolvido
- [ ] Handler Lambda real + `ScreenStore` S3 (resolver.go inalterado)
- [ ] Host de produção implementando `docs/host-contract.md`
- [ ] Feature flag por jornada + plano de comparação legado × SDUI
- [ ] Aprovação de compliance do modelo de evidência (JSON resolvido + evento submit)
- [ ] Skill de autoria preenchida com docs do DS da empresa (`TODO(empresa)`)

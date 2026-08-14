# Contexto do projeto: POC de migração para SDUI (plataforma de assinatura de contratos)

## Quem sou eu (o usuário deste repo)
Tech Lead de uma squad de plataforma (foundation) de um produto de assinatura digital de contratos.
Stack principal: Java/Kotlin, Go, AWS (Lambda, EKS, Aurora PostgreSQL).

## Problema original
Arquitetura atual de renderização de telas de formalização (multi-jornada, multi-produto):
1. **Serviço de originação**: produto passa variáveis + valores.
2. **Lambda Go**: lê um `template.html`, faz replace de variáveis (string-based), salva o HTML final num bucket.
3. **Frontend**: busca o HTML no bucket, converte pra base64, injeta na tela (`dangerouslySetInnerHTML`-like).

Isso foi identificado como frágil: risco de XSS (HTML dinâmico no front), sem enforcement de design system,
autoria pesada pra times de produto (sintaxe de Go template sem IDE), acoplamento entre a tela e o artefato
legal imutável, drift entre `template.html`/`schema.json`/`meta.json`.

## Decisão tomada: migrar para Server-Driven UI (SDUI)
Em vez do backend montar HTML, o backend envia um **JSON declarativo** descrevendo a árvore de componentes
(tipo + props + children), validado contra um **catálogo fechado** de componentes do design system. O front
tem um **renderer único** que interpreta esse JSON e desenha com componentes reais (não HTML injetado).

Referência de mercado usada para validar a abordagem: Nubank tem um framework de SDUI em produção; o protocolo
aberto **A2UI** (Google, a2ui.org) resolve o mesmo problema para UIs geradas por agentes de IA, com o mesmo
princípio de segurança: **catálogo whitelisted + validação de schema = a IA/backend nunca produz algo que o
renderer não sabe desenhar**. Não estamos adotando o protocolo A2UI literal, só o padrão de design (catálogo +
validação + loop de correção).

## Roadmap de migração (7 passos, já alinhado)
1. Definir catálogo de componentes (JSON Schema por componente: tipo, props, se aceita children).
2. Definir formato de screen definition (JSON declarativo, versionado `screenId + screenVersion`, com
   `schema.json` ao lado pra validar payload).
3. Escolher uma jornada simples como piloto e migrar o template manualmente (teste de expressividade do
   catálogo).
4. Adaptar o Lambda: de "template render" pra "schema resolve" — valida payload contra schema, faz merge das
   variáveis nos placeholders da árvore JSON, salva o JSON resolvido (não HTML). Mais simples que hoje: não
   precisa mais sanitização de HTML/escaping.
5. Construir o SDUI renderer no front: um componente que recebe o JSON resolvido e despacha por `type` pros
   componentes reais do design system.
6. Rodar em paralelo (feature flag por jornada) comparando HTML antigo vs SDUI novo antes de cortar tráfego.
7. Migrar produto a produto + decidir separadamente a evidência legal (gerar PDF a partir da mesma screen
   definition + payload, em vez de reusar HTML congelado — mantém a evidência legal desacoplada do fluxo de
   tela).

## Extensão futura: autoria assistida por IA
Ideia: times de produto + um agente de IA (alimentado com os schemas do catálogo + docs do design system como
skill) montam a screen definition por conversa, com um "playground" que valida e renderiza com o **mesmo
renderer de produção** (pra não criar um segundo renderer e reintroduzir drift). Gate de conteúdo humano
(compliance/jurídico) continua obrigatório antes de `draft` virar `published`, já que são telas de
formalização com peso legal — validação de schema garante segurança estrutural, não correção jurídica do
conteúdo.

## Escopo da POC atual (o que este repo deve entregar)
Duas entregas:
1. **Base de código exportável** (este repo): catálogo, exemplos de screen definition, resolver em Go
   (stdlib only, sem dependências externas — mantém o POC compilável offline), renderer de referência em
   React/TypeScript.
2. **Demo interativo**: um artifact React (já gerado numa sessão anterior no claude.ai) simulando o pipeline
   completo (editor de screen definition + variáveis → botão "Resolver" simulando o Lambda → preview
   renderizado). Serve pra visualizar o conceito; este repo é o código "de verdade" que evolui a partir dele.

## Estrutura já criada
```
catalog/schema.json      <- catálogo v1.1.0, props tipadas (Section, Text, List, Accordion,
                            KeyValueTable, Checkbox, Button com action semântica)
screens/schema.json      <- JSON Schema do envelope de screen definition (screenId/screenVersion/
                            catalogVersion/variables/root)
screens/FORMAT.md        <- semântica de placeholders {{var}}, ações (navigate/back/submit),
                            regra de compatibilidade catálogo × screen
docs/host-contract.md    <- contrato do host: estado de checkboxes, gating do botão de aceite
                            (submit accept desabilitado até todos os Checkbox required marcados),
                            shape dos eventos SduiEvent, mock HTTP com httpbin.org
docs/adoption-guide.md   <- insumos de implantação corporativa (catálogo×DS, Lambda/S3,
                            rollout, evidência legal, checklist de kickoff)
screens/example-*.json   <- 3 fixtures (consórcio + detalhes com back, cartão)
resolver/                <- PRONTO: resolver.go (stdlib) + main.go (CLI simula Lambda via
                            ScreenStore) + 17 testes passando
renderer/                <- PRONTO: SduiRenderer puro + 7 componentes + DemoHost
                            (gating/navegação/HTTP), typecheck estrito ok
skills/sdui-authoring/   <- skill de autoria assistida por IA, placeholders TODO(empresa)
                            pra acoplar docs do design system real
```
Ambiente local: Go e Node NÃO estão instalados na máquina (`dnf install golang nodejs`);
nas sessões anteriores foram usados toolchains baixados no scratchpad.

## Decisões de arquitetura já fixadas (não rediscutir sem motivo novo)
- JSON declara **intenção e vínculos**; comportamento vive no **host** (app que embute o
  renderer). Nunca URLs, expressões/condicionais ou código na screen definition.
- Gating de aceite é convenção do catálogo implementada no host, não expressão no JSON.
- Renderer é puro: desenha e emite `onEvent(SduiEvent)`; host traduz eventos em chamadas de API.
- Evento `submit` carrega `screenId + screenVersion + acceptances` = registro de evidência
  legal, correlacionável com o JSON resolvido imutável no bucket.
- POC inclui conexão HTTP real mockada: host de demo envia eventos via POST para
  `https://httpbin.org/post` (config do host, trocável pela API real depois).

## Próximos passos concretos pra esta sessão do Claude Code
1. Criar `screens/example-consorcio.json` e `screens/example-cartao.json` — duas screen definitions de
   produtos diferentes, mesmos tipos de componente, pra provar que o resolver é agnóstico a produto.
   Devem exercitar Checkbox required + Buttons (navigate/back/submit accept/decline), incluindo uma
   tela secundária de detalhes com botão de voltar.
2. Implementar `resolver/resolver.go`: função `ResolveScreen(screenDef, vars, catalog) (resolved, errors)` que
   percorre a árvore recursivamente, valida `type` contra o catálogo, valida props obrigatórias, substitui
   placeholders `{{variavel}}` pelos valores do payload. Sem libs externas (stdlib `encoding/json` só).
3. `resolver/main.go`: handler de Lambda (`github.com/aws/aws-lambda-go/events` + `lambda.Start`) chamando
   `ResolveScreen` e salvando o resultado no bucket (pode simular o `S3 PutObject` com uma interface
   `ScreenStore` pra manter testável sem AWS real).
4. `resolver/resolver_test.go`: casos felizes + casos de erro (componente fora do catálogo, variável
   ausente, prop obrigatória faltando).
5. `renderer/types.ts`: interface `ScreenNode { type: string; props: Record<string, unknown>; children?:
   ScreenNode[] }`.
6. `renderer/SduiRenderer.tsx` + `renderer/components/{Text,List,Accordion,KeyValueTable,Section,
   Checkbox,Button}.tsx`: dispatch por `type`, renderer puro emitindo `onEvent(SduiEvent)` conforme
   `docs/host-contract.md`; estilização simples (placeholder pro design system real).
6b. `renderer/DemoHost.tsx`: host de demonstração — mantém estado de checkboxes, aplica gating do
   submit accept, pilha de navegação (navigate/back), e envia eventos via POST pra
   `https://httpbin.org/post` (checkbox-changed fire-and-forget; submit síncrono com tratamento
   de sucesso/erro). URL do endpoint em config do host, nunca na screen definition.
7. `README.md` explicando como rodar (`go test ./...` no resolver) e como plugar o catálogo/design system
   real.

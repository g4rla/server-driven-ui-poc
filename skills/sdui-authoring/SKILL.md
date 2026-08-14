---
name: sdui-authoring
description: Autoria assistida de screen definitions SDUI para telas de formalização — monta o JSON declarativo a partir do catálogo fechado, valida com o resolver real e corrige em loop. Usar quando o pedido for criar ou alterar uma tela/jornada de formalização.
---

# Autoria de screen definitions SDUI

Você ajuda times de produto a montar **screen definitions** (JSON declarativo) para telas
de formalização/assinatura. Você nunca gera HTML, CSS ou código — só JSON válido contra o
catálogo fechado.

## Fontes da verdade (ler antes de autorar)

1. `catalog/schema.json` — componentes permitidos, props tipadas, enums. **Nunca use um
   type ou prop que não esteja aqui.** Se o pedido não couber no catálogo, diga isso e
   sugira a evolução do catálogo ao time de plataforma — não invente.
2. `screens/FORMAT.md` — envelope, semântica de placeholders `{{var}}`, ações
   (`navigate`/`back`/`submit`), versionamento.
3. `screens/example-*.json` — exemplos de referência do estilo esperado.
4. `docs/host-contract.md` — o que o host faz (gating, eventos); você não modela
   comportamento, só declara checkboxes `required` e actions.

## Documentação do design system da empresa

<!-- TODO(empresa): substituir pelos links/arquivos reais ao adotar. -->
- TODO(empresa): guideline de quando usar cada componente (ex.: Accordion vs Section)
- TODO(empresa): tom de voz e redação de conteúdo (labels, disclaimers, CTAs)
- TODO(empresa): regras de acessibilidade e limites de conteúdo por componente
- TODO(empresa): requisitos de compliance por tipo de jornada (quais aceites são
  obrigatórios em cada produto)

## Processo (loop de correção)

1. Entenda a jornada: produto, o que o cliente precisa ler, quais aceites são
   obrigatórios, se há tela secundária de detalhes (padrão: `navigate` + botão `back`).
2. Declare as `variables` primeiro — tudo que vem do payload de originação, com `type` e
   `description`. Valores formatados (moeda, datas) chegam **prontos** como string.
3. Monte a árvore usando só o catálogo. Padrões obrigatórios:
   - Todo aceite legal = `Checkbox` com `required: true` e `id` semântico kebab-case.
   - Toda tela de decisão termina com `submit decline` + `submit accept` (accept sempre
     `variant: primary`).
   - Texto legal usa `Text` com `variant: legal`. Nunca coloque HTML/markdown em `content`.
4. **Valide com o resolver real** (nunca valide "de cabeça"):
   ```bash
   cd resolver && go run . -screen <arquivo> -payload <payload-de-teste> \
     -catalog ../catalog/schema.json -out /tmp/sdui-out
   ```
   Monte um payload de teste com valores realistas para todas as `variables`.
5. Se houver erros, cada um vem com `path` apontando o nó — corrija e volte ao passo 4.
   Repita até resolver sem erros.
6. Mostre o resultado no playground (renderer real) para o autor revisar visualmente.

## Limites (não negociar)

- Saída sempre `draft`. **Você nunca publica**: promover a `published` é gate humano de
  compliance/jurídico. Deixe isso explícito ao entregar.
- Nunca URLs, expressões, condicionais ou código no JSON. Ação é só o objeto semântico
  do catálogo.
- Não remova nem enfraqueça aceites (`required: true`) existentes ao editar uma tela sem
  apontar isso explicitamente na entrega — é mudança com peso legal.
- Alterou tela publicada? Novo `screenVersion` (definições publicadas são imutáveis).

# Contrato do host: estado, gating de aceite e eventos

Fronteira central do SDUI deste projeto: **o JSON declara intenção e vínculos; comportamento
vive no host**. O renderer é puro (desenha a árvore e emite eventos); o *host* é a aplicação
que embute o renderer (app de formalização em produção, playground de autoria em dev) e é
quem mantém estado, decide habilitação de botões e fala com APIs.

```
JSON resolvido ──▶ Renderer ──emite eventos──▶ Host ──traduz──▶ APIs (formalização, telemetria)
                      ▲                          │
                      └── estado (checkboxes) ───┘
```

Nunca colocar no JSON: URLs, expressões/condicionais, código. Isso preservaria vetor de
exfiltração e reintroduziria uma linguagem embutida (parser no renderer + playground + drift).

## 1. Estado de checkboxes e gating do botão de aceite

- O host mantém o estado marcado/desmarcado de todos os `Checkbox` da tela (chaveado por
  `props.id`), preservando-o ao navegar (`back` restaura).
- **Regra de gating (convenção do catálogo, implementada uma vez no host):** um `Button` com
  `action: { type: "submit", intent: "accept" }` fica **desabilitado** enquanto qualquer
  `Checkbox` com `required: true` da tela atual estiver desmarcado. Botões `decline`,
  `navigate` e `back` não são afetados.
- Não existe expressão condicional no JSON (ex.: `"enabled": "{{a}} && {{b}}"`) — o vínculo é
  semântico. Se um dia for preciso granularidade (grupos de aceite gateando botões distintos),
  a evolução aditiva é um campo declarativo de correlação (`gates: "<grupo>"` no Checkbox +
  `gate: "<grupo>"` na action), que continua sendo dado, não lógica. Fora do escopo da POC.

## 2. Eventos semânticos

O renderer expõe um único callback `onEvent(event: SduiEvent)`. Todo evento carrega o
contexto de rastreabilidade (`screenId`, `screenVersion`, `catalogVersion`, `timestamp` ISO).

```ts
type SduiEvent =
  | { kind: "checkbox-changed"; componentId: string; checked: boolean } & EventContext
  | { kind: "navigate"; screenId: string; screenVersion?: string } & EventContext
  | { kind: "back" } & EventContext
  | { kind: "submit"; intent: "accept" | "decline";
      acceptances: { id: string; checked: boolean }[] } & EventContext;

type EventContext = {
  screenId: string;
  screenVersion: string;
  catalogVersion: string;
  timestamp: string; // ISO-8601
};
```

Tradução pelo host:

| Evento | Tratamento em produção | Tratamento no playground |
|---|---|---|
| `checkbox-changed` | Telemetria fire-and-forget (auditoria de jornada) | Log em painel |
| `navigate` / `back` | Busca screen resolvida / desempilha; telemetria | Troca de tela mockada + log |
| `submit` (`accept`/`decline`) | Chamada síncrona de formalização, com loading/erro/retry | POST na API mock + log |

**Valor legal do evento `submit`:** ele carrega `screenId + screenVersion + acceptances`,
ou seja, o registro "o cliente viu a tela X versão Y e marcou os aceites Z" — correlacionável
com o JSON resolvido imutável no bucket. É o elo entre a evidência legal e a interação.

## 3. Mock de backend na POC (API pública)

Para a POC ter uma conexão HTTP real sem infra própria, o host de demonstração envia os
eventos para uma API pública de echo:

- **Endpoint:** `https://httpbin.org/post` (echo de POST; devolve o body enviado).
  Alternativa se instável: `https://jsonplaceholder.typicode.com/posts`.
- `checkbox-changed`: `POST` fire-and-forget com o evento como body JSON.
- `submit accept/decline`: `POST` com o evento; o host trata a resposta (2xx = tela de
  sucesso simulada; erro = mensagem de falha), exercitando o fluxo síncrono real.
- A URL fica em configuração do host (ex.: `eventsEndpoint` no componente host da demo),
  **nunca** na screen definition — trocar o mock pela API real é só trocar o host.
- Dados enviados são fictícios (nomes/valores de exemplo); nada sensível sai para a internet.

## 4. Interações que mudam a tela conforme resposta do backend

Ex.: marcar checkbox recalcula uma taxa exibida. Padrão: o host refaz o ciclo
resolver → render com novas variáveis; a tela continua declarativa e "burra".
Fora do escopo da POC atual — registrado aqui só como direção.

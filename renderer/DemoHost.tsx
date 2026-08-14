// Host de demonstração (docs/host-contract.md). É o dono do comportamento:
// estado dos checkboxes por tela, gating do submit/accept, pilha de navegação
// e tradução dos SduiEvent em chamadas HTTP para a API mock (httpbin echo).
// Em produção este componente é substituído pelo app real; o SduiRenderer não muda.
import React, { useMemo, useState } from "react";
import type { CheckboxState, ResolvedScreen, SduiEvent } from "./types";
import { collectCheckboxes } from "./types";
import { SduiRenderer } from "./SduiRenderer";

export interface DemoHostProps {
  /** Screens resolvidas disponíveis, chaveadas por screenId (saída do resolver Go). */
  screens: Record<string, ResolvedScreen>;
  /** screenId inicial da jornada. */
  initialScreenId: string;
  /** Endpoint que recebe os eventos. Config do host — nunca vem da screen definition. */
  eventsEndpoint?: string;
}

type SubmitStatus =
  | { phase: "idle" }
  | { phase: "sending" }
  | { phase: "done"; intent: "accept" | "decline" }
  | { phase: "error"; message: string };

export function DemoHost({
  screens,
  initialScreenId,
  eventsEndpoint = "https://httpbin.org/post",
}: DemoHostProps) {
  // Pilha de navegação: topo = tela atual. `back` desempilha.
  const [stack, setStack] = useState<string[]>([initialScreenId]);
  // Estado de checkboxes por screenId — preservado ao navegar e voltar.
  const [checkboxes, setCheckboxes] = useState<Record<string, CheckboxState>>({});
  const [status, setStatus] = useState<SubmitStatus>({ phase: "idle" });
  const [eventLog, setEventLog] = useState<SduiEvent[]>([]);

  const currentId = stack[stack.length - 1];
  const screen = screens[currentId];
  const state = checkboxes[currentId] ?? {};

  // Gating: submit/accept habilitado só com todos os Checkbox required marcados.
  const submitAcceptEnabled = useMemo(() => {
    if (!screen) return false;
    return collectCheckboxes(screen.root)
      .filter((c) => c.required)
      .every((c) => state[c.id] === true);
  }, [screen, state]);

  const postEvent = (event: SduiEvent) =>
    fetch(eventsEndpoint, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(event),
    });

  const onEvent = (event: SduiEvent) => {
    setEventLog((log) => [...log, event]);
    switch (event.kind) {
      case "checkbox-changed":
        setCheckboxes((all) => ({
          ...all,
          [currentId]: { ...all[currentId], [event.componentId]: event.checked },
        }));
        // Telemetria fire-and-forget: falha do mock não bloqueia a interação.
        postEvent(event).catch(() => {});
        break;

      case "navigate":
        if (screens[event.screenId]) {
          setStack((s) => [...s, event.screenId]);
        }
        postEvent(event).catch(() => {});
        break;

      case "back":
        setStack((s) => (s.length > 1 ? s.slice(0, -1) : s));
        postEvent(event).catch(() => {});
        break;

      case "submit":
        // Chamada síncrona de formalização: loading + tratamento de erro.
        setStatus({ phase: "sending" });
        postEvent(event)
          .then((res) => {
            if (!res.ok) throw new Error(`HTTP ${res.status}`);
            setStatus({ phase: "done", intent: event.intent });
          })
          .catch((err: Error) =>
            setStatus({ phase: "error", message: err.message })
          );
        break;
    }
  };

  if (!screen) {
    return <p>Screen não encontrada: {currentId}</p>;
  }

  if (status.phase === "done") {
    return (
      <div style={{ maxWidth: 480, margin: "0 auto", padding: 16, textAlign: "center" }}>
        <h2>{status.intent === "accept" ? "Contrato assinado ✔" : "Proposta recusada"}</h2>
        <p style={{ color: "#666" }}>
          Evento de {status.intent} registrado para {screen.screenId}@{screen.screenVersion}.
        </p>
      </div>
    );
  }

  return (
    <div style={{ maxWidth: 480, margin: "0 auto", padding: 16, display: "flex", flexDirection: "column", gap: 16 }}>
      <SduiRenderer
        screen={screen}
        checkboxState={state}
        submitAcceptEnabled={submitAcceptEnabled}
        onEvent={onEvent}
      />
      {status.phase === "sending" && <p>Enviando…</p>}
      {status.phase === "error" && (
        <p style={{ color: "#c00" }}>Falha ao enviar: {status.message}. Tente novamente.</p>
      )}
      <details>
        <summary style={{ cursor: "pointer", color: "#666" }}>
          Eventos emitidos ({eventLog.length})
        </summary>
        <pre style={{ fontSize: 11, overflowX: "auto" }}>
          {JSON.stringify(eventLog, null, 2)}
        </pre>
      </details>
    </div>
  );
}

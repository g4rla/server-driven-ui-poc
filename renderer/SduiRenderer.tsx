// Renderer SDUI de referência. Puro: recebe o JSON resolvido + estado vindo do host,
// despacha por `type` para componentes reais do design system e emite SduiEvent via
// onEvent. Não faz requests, não guarda estado, não interpreta HTML — conteúdo é
// sempre texto puro (nunca dangerouslySetInnerHTML).
import React, { createContext, useContext } from "react";
import type { CheckboxState, ResolvedScreen, ScreenNode, SduiEvent } from "./types";
import { Section } from "./components/Section";
import { Text } from "./components/Text";
import { List } from "./components/List";
import { Accordion } from "./components/Accordion";
import { KeyValueTable } from "./components/KeyValueTable";
import { Checkbox } from "./components/Checkbox";
import { Button } from "./components/Button";

export interface RendererApi {
  checkboxState: CheckboxState;
  /** Gating calculado pelo host: false desabilita botões submit/accept. */
  submitAcceptEnabled: boolean;
  emit: (event: SduiEvent) => void;
  screen: ResolvedScreen;
}

const RendererContext = createContext<RendererApi | null>(null);

export function useRenderer(): RendererApi {
  const ctx = useContext(RendererContext);
  if (!ctx) throw new Error("componente SDUI usado fora do SduiRenderer");
  return ctx;
}

// Catálogo do lado do front: dispatch fechado por type. Tipo desconhecido nunca chega
// aqui em produção (o resolver rejeita), mas falhamos visível em vez de silencioso.
const registry: Record<string, React.ComponentType<{ node: ScreenNode }>> = {
  Section,
  Text,
  List,
  Accordion,
  KeyValueTable,
  Checkbox,
  Button,
};

export function RenderNode({ node }: { node: ScreenNode }) {
  const Component = registry[node.type];
  if (!Component) {
    return (
      <div style={{ border: "2px solid #c00", color: "#c00", padding: 8 }}>
        Componente fora do catálogo: {node.type}
      </div>
    );
  }
  return <Component node={node} />;
}

export function RenderChildren({ node }: { node: ScreenNode }) {
  return (
    <>
      {node.children?.map((child, i) => (
        <RenderNode key={i} node={child} />
      ))}
    </>
  );
}

export interface SduiRendererProps {
  screen: ResolvedScreen;
  checkboxState: CheckboxState;
  submitAcceptEnabled: boolean;
  onEvent: (event: SduiEvent) => void;
}

export function SduiRenderer({
  screen,
  checkboxState,
  submitAcceptEnabled,
  onEvent,
}: SduiRendererProps) {
  const emit = (event: SduiEvent) => onEvent(event);
  return (
    <RendererContext.Provider
      value={{ checkboxState, submitAcceptEnabled, emit, screen }}
    >
      <RenderNode node={screen.root} />
    </RendererContext.Provider>
  );
}

/** Monta o EventContext de rastreabilidade a partir da screen atual. */
export function eventContext(screen: ResolvedScreen) {
  return {
    screenId: screen.screenId,
    screenVersion: screen.screenVersion,
    catalogVersion: screen.catalogVersion,
    timestamp: new Date().toISOString(),
  };
}

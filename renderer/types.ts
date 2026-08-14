// Tipos do protocolo SDUI do lado do front. Espelham o JSON resolvido produzido
// pelo resolver Go e os eventos do contrato do host (docs/host-contract.md).

export interface ScreenNode {
  type: string;
  props: Record<string, unknown>;
  children?: ScreenNode[];
}

export interface ResolvedScreen {
  screenId: string;
  screenVersion: string;
  catalogVersion: string;
  resolvedAt: string;
  root: ScreenNode;
}

// ---- Ações declarativas (props.action de Button) ----

export type ButtonAction =
  | { type: "navigate"; screenId: string; screenVersion?: string }
  | { type: "back" }
  | { type: "submit"; intent: "accept" | "decline" };

// ---- Eventos semânticos emitidos pelo renderer ----

export interface EventContext {
  screenId: string;
  screenVersion: string;
  catalogVersion: string;
  timestamp: string; // ISO-8601
}

export type SduiEvent = EventContext &
  (
    | { kind: "checkbox-changed"; componentId: string; checked: boolean }
    | { kind: "navigate"; screenId: string; screenVersion?: string }
    | { kind: "back" }
    | {
        kind: "submit";
        intent: "accept" | "decline";
        acceptances: { id: string; checked: boolean }[];
      }
  );

export type CheckboxState = Record<string, boolean>;

// Percorre a árvore coletando os Checkbox; usado pelo host para gating e acceptances.
export function collectCheckboxes(
  node: ScreenNode
): { id: string; required: boolean }[] {
  const found: { id: string; required: boolean }[] = [];
  const walk = (n: ScreenNode) => {
    if (n.type === "Checkbox") {
      found.push({
        id: String(n.props.id),
        required: n.props.required === true,
      });
    }
    n.children?.forEach(walk);
  };
  walk(node);
  return found;
}

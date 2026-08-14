import React from "react";
import { collectCheckboxes, type ButtonAction, type ScreenNode } from "../types";
import { eventContext, useRenderer } from "../SduiRenderer";

const variants: Record<string, React.CSSProperties> = {
  primary: { background: "#1a56db", color: "#fff", border: "none" },
  secondary: { background: "#fff", color: "#1a56db", border: "1px solid #1a56db" },
  danger: { background: "#fff", color: "#c00", border: "1px solid #c00" },
};

export function Button({ node }: { node: ScreenNode }) {
  const { emit, screen, checkboxState, submitAcceptEnabled } = useRenderer();
  const action = node.props.action as ButtonAction;
  const variant = (node.props.variant as string) ?? "primary";

  // Gating (contrato do host): só submit/accept é desabilitável pelos required checkboxes.
  const disabled =
    action.type === "submit" && action.intent === "accept" && !submitAcceptEnabled;

  const onClick = () => {
    const ctx = eventContext(screen);
    switch (action.type) {
      case "navigate":
        emit({
          ...ctx,
          kind: "navigate",
          screenId: action.screenId,
          ...(action.screenVersion ? { screenVersion: action.screenVersion } : {}),
        });
        break;
      case "back":
        emit({ ...ctx, kind: "back" });
        break;
      case "submit":
        emit({
          ...ctx,
          kind: "submit",
          intent: action.intent,
          acceptances: collectCheckboxes(screen.root).map(({ id }) => ({
            id,
            checked: checkboxState[id] === true,
          })),
        });
        break;
    }
  };

  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      style={{
        padding: "10px 16px",
        borderRadius: 6,
        fontSize: 15,
        fontWeight: 600,
        cursor: disabled ? "not-allowed" : "pointer",
        opacity: disabled ? 0.5 : 1,
        ...variants[variant],
      }}
    >
      {String(node.props.label)}
    </button>
  );
}

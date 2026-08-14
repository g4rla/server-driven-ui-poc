import React from "react";
import type { ScreenNode } from "../types";
import { eventContext, useRenderer } from "../SduiRenderer";

export function Checkbox({ node }: { node: ScreenNode }) {
  const { checkboxState, emit, screen } = useRenderer();
  const id = String(node.props.id);
  const checked = checkboxState[id] === true;
  return (
    <label style={{ display: "flex", alignItems: "flex-start", gap: 8, cursor: "pointer" }}>
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) =>
          emit({
            ...eventContext(screen),
            kind: "checkbox-changed",
            componentId: id,
            checked: e.target.checked,
          })
        }
      />
      <span>
        {String(node.props.label)}
        {node.props.required === true && <span style={{ color: "#c00" }}> *</span>}
      </span>
    </label>
  );
}

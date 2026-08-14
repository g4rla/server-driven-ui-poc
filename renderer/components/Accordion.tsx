import React, { useState } from "react";
import type { ScreenNode } from "../types";
import { RenderChildren } from "../SduiRenderer";

export function Accordion({ node }: { node: ScreenNode }) {
  const [open, setOpen] = useState(node.props.defaultOpen === true);
  return (
    <div style={{ border: "1px solid #ddd", borderRadius: 6 }}>
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        aria-expanded={open}
        style={{
          width: "100%",
          textAlign: "left",
          padding: "10px 12px",
          background: "none",
          border: "none",
          fontWeight: 600,
          cursor: "pointer",
        }}
      >
        {open ? "▾" : "▸"} {String(node.props.title)}
      </button>
      {open && (
        <div style={{ padding: "0 12px 12px", display: "flex", flexDirection: "column", gap: 8 }}>
          <RenderChildren node={node} />
        </div>
      )}
    </div>
  );
}

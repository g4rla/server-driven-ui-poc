import React from "react";
import type { ScreenNode } from "../types";

export function List({ node }: { node: ScreenNode }) {
  const items = node.props.items as string[];
  return (
    <ul style={{ margin: 0, paddingLeft: 20, display: "flex", flexDirection: "column", gap: 4 }}>
      {items.map((item, i) => (
        <li key={i}>{item}</li>
      ))}
    </ul>
  );
}

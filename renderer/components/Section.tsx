import React from "react";
import type { ScreenNode } from "../types";
import { RenderChildren } from "../SduiRenderer";

export function Section({ node }: { node: ScreenNode }) {
  const title = node.props.title as string | undefined;
  return (
    <section style={{ display: "flex", flexDirection: "column", gap: 12, margin: "8px 0" }}>
      {title && <h2 style={{ fontSize: 18, margin: 0 }}>{title}</h2>}
      <RenderChildren node={node} />
    </section>
  );
}

import React from "react";
import type { ScreenNode } from "../types";

const variants: Record<string, React.CSSProperties> = {
  body: { fontSize: 15 },
  caption: { fontSize: 13, color: "#666" },
  legal: { fontSize: 12, color: "#666", fontStyle: "italic" },
  heading: { fontSize: 17, fontWeight: 600 },
};

export function Text({ node }: { node: ScreenNode }) {
  const variant = (node.props.variant as string) ?? "body";
  // content é sempre texto puro — nunca HTML/markdown.
  return <p style={{ margin: 0, ...variants[variant] }}>{String(node.props.content)}</p>;
}

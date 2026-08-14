import React from "react";
import type { ScreenNode } from "../types";

export function KeyValueTable({ node }: { node: ScreenNode }) {
  const rows = node.props.rows as { label: string; value: string }[];
  return (
    <table style={{ borderCollapse: "collapse", width: "100%" }}>
      <tbody>
        {rows.map((row, i) => (
          <tr key={i} style={{ borderBottom: "1px solid #eee" }}>
            <th scope="row" style={{ textAlign: "left", padding: "6px 8px", color: "#666", fontWeight: 500 }}>
              {row.label}
            </th>
            <td style={{ padding: "6px 8px", textAlign: "right", fontWeight: 600 }}>{row.value}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

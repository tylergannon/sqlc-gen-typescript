import { describe, expect, it } from "bun:test";
import {
  EmitHint,
  NewLineKind,
  Node,
  ScriptKind,
  ScriptTarget,
  createPrinter,
  createSourceFile,
} from "typescript";

import { Column, Identifier, Parameter } from "../gen/plugin/codegen_pb";
import * as postgres from "./postgres";

function render(node: Node): string {
  const sourceFile = createSourceFile("file.ts", "", ScriptTarget.Latest, false, ScriptKind.TS);
  return createPrinter({ newLine: NewLineKind.LineFeed }).printNode(
    EmitHint.Unspecified,
    node,
    sourceFile,
  );
}

function column(name: string, typeName: string, notNull: boolean): Column {
  return new Column({
    name,
    notNull,
    type: new Identifier({ name: typeName }),
  });
}

describe("postgres driver JSON types", () => {
  it("maps jsonb to JsonValue", () => {
    expect(render(postgres.columnType(column("profile", "jsonb", true)))).toBe("JsonValue");
  });

  it("preserves SQL nullability for nullable jsonb", () => {
    expect(render(postgres.columnType(column("notes", "jsonb", false)))).toBe("JsonValue | null");
  });

  it("throws instead of emitting any when type metadata is missing", () => {
    expect(() => postgres.columnType(new Column({ name: "payload" }))).toThrow(
      /Missing PostgreSQL type metadata/,
    );
  });

  it("wraps scalar JSON parameters with sql.json", () => {
    const node = postgres.execDecl(
      "updateAuthor",
      "UPDATE authors SET profile = $1, notes = $2",
      "UpdateAuthorArgs",
      [
        new Parameter({ column: column("profile", "jsonb", true) }),
        new Parameter({ column: column("notes", "jsonb", false) }),
      ],
    );

    const output = render(node);
    expect(output).toContain("${sql.json(args.profile)}");
    expect(output).toContain("${args.notes === null ? null : sql.json(args.notes)}");
  });
});

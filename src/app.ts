import { readFileSync, writeFileSync, STDIO } from "javy/fs";
import {
  EmitHint,
  NewLineKind,
  ScriptKind,
  ScriptTarget,
  SyntaxKind,
  Node,
  createPrinter,
  createSourceFile,
  factory,
} from "typescript";

import { GenerateRequest, GenerateResponse, File, Enum } from "./gen/plugin/codegen_pb";

import { argName, colName } from "./drivers/utils";
import { assertUniqueNames } from "./validate";
import * as postgres from "./drivers/postgres";

// Read input from stdin
const input = readInput();
// Call the function with the input
const result = codegen(input);
// Write the result to stdout
writeOutput(result);

/**
 * Build a map of enum names to their values from the catalog.
 * This allows us to recognize enum types and generate appropriate TypeScript types.
 */
function buildEnumMap(input: GenerateRequest): Map<string, Enum> {
  const enumMap = new Map<string, Enum>();
  const defaultSchema = input.catalog?.defaultSchema ?? "public";

  for (const schema of input.catalog?.schemas ?? []) {
    if (schema.name === "pg_catalog" || schema.name === "information_schema") {
      continue;
    }

    for (const enumDef of schema.enums) {
      // Store with both qualified and unqualified names
      enumMap.set(enumDef.name, enumDef);
      if (schema.name !== defaultSchema) {
        enumMap.set(`${schema.name}.${enumDef.name}`, enumDef);
      }
    }
  }

  return enumMap;
}

/**
 * Generate TypeScript union type for an enum.
 * e.g., type EventSource = 'user' | 'runner' | 'system';
 */
function enumTypeDecl(name: string, enumDef: Enum): Node {
  const unionType = factory.createUnionTypeNode(
    enumDef.vals.map((val) => factory.createLiteralTypeNode(factory.createStringLiteral(val))),
  );

  return factory.createTypeAliasDeclaration(
    [factory.createToken(SyntaxKind.ExportKeyword)],
    factory.createIdentifier(pascalCase(name)),
    undefined,
    unionType,
  );
}

function jsonTypeDecls(): Node[] {
  return [
    factory.createTypeAliasDeclaration(
      [factory.createToken(SyntaxKind.ExportKeyword)],
      factory.createIdentifier("JsonPrimitive"),
      undefined,
      factory.createUnionTypeNode([
        factory.createKeywordTypeNode(SyntaxKind.StringKeyword),
        factory.createKeywordTypeNode(SyntaxKind.NumberKeyword),
        factory.createKeywordTypeNode(SyntaxKind.BooleanKeyword),
        factory.createLiteralTypeNode(factory.createNull()),
      ]),
    ),
    factory.createTypeAliasDeclaration(
      [factory.createToken(SyntaxKind.ExportKeyword)],
      factory.createIdentifier("JsonValue"),
      undefined,
      factory.createUnionTypeNode([
        factory.createTypeReferenceNode(factory.createIdentifier("JsonPrimitive"), undefined),
        factory.createTypeOperatorNode(
          SyntaxKind.ReadonlyKeyword,
          factory.createArrayTypeNode(
            factory.createTypeReferenceNode(factory.createIdentifier("JsonValue"), undefined),
          ),
        ),
        factory.createTypeLiteralNode([
          factory.createIndexSignature(
            [factory.createToken(SyntaxKind.ReadonlyKeyword)],
            [
              factory.createParameterDeclaration(
                undefined,
                undefined,
                factory.createIdentifier("key"),
                undefined,
                factory.createKeywordTypeNode(SyntaxKind.StringKeyword),
                undefined,
              ),
            ],
            factory.createUnionTypeNode([
              factory.createTypeReferenceNode(factory.createIdentifier("JsonValue"), undefined),
              factory.createKeywordTypeNode(SyntaxKind.UndefinedKeyword),
            ]),
          ),
        ]),
      ]),
    ),
  ];
}

/**
 * Convert snake_case to PascalCase
 */
function pascalCase(str: string): string {
  return str
    .split("_")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1).toLowerCase())
    .join("");
}

function codegen(input: GenerateRequest): GenerateResponse {
  const files = [];
  const enumMap = buildEnumMap(input);

  // Set the enum map in the postgres driver so columnType can use it
  postgres.setEnumMap(enumMap);

  const querymap = new Map<string, typeof input.queries>();

  for (const query of input.queries) {
    if (!querymap.has(query.filename)) {
      querymap.set(query.filename, []);
    }
    const qs = querymap.get(query.filename);
    qs?.push(query);
  }

  // Track which enums are used across all files
  const usedEnums = new Set<string>();

  for (const [filename, queries] of querymap.entries()) {
    const nodes: Node[] = [...postgres.preamble()];

    // Track enums used in this file
    const fileEnums = new Set<string>();
    let fileUsesJson = false;

    for (const query of queries) {
      const lowerName = query.name[0].toLowerCase() + query.name.slice(1);

      let argIface = undefined;
      let returnIface = undefined;

      if (query.params.length > 0) {
        argIface = `${query.name}Args`;
        const names = query.params.map((param, i) => argName(i, param.column));
        assertUniqueNames({
          kind: "argument",
          queryName: query.name,
          fileName: filename,
          names,
        });

        // Check for enum usage in params
        for (const param of query.params) {
          const enumName = postgres.getEnumName(param.column);
          if (enumName) {
            fileEnums.add(enumName);
            usedEnums.add(enumName);
          }
          if (postgres.isJsonColumn(param.column)) {
            fileUsesJson = true;
          }
        }

        try {
          nodes.push(
            factory.createInterfaceDeclaration(
              [factory.createToken(SyntaxKind.ExportKeyword)],
              factory.createIdentifier(argIface),
              undefined,
              undefined,
              query.params.map((param, i) =>
                factory.createPropertySignature(
                  undefined,
                  factory.createIdentifier(argName(i, param.column)),
                  undefined,
                  postgres.columnType(param.column),
                ),
              ),
            ),
          );
        } catch (err) {
          throw new Error(
            `Error in query "${query.name}" (${filename}): ${err instanceof Error ? err.message : String(err)}`,
          );
        }
      }

      if (query.columns.length > 0) {
        returnIface = `${query.name}Row`;
        const names = query.columns.map((column, i) => colName(i, column));
        assertUniqueNames({
          kind: "column",
          queryName: query.name,
          fileName: filename,
          names,
        });

        // Check for enum usage in columns
        for (const col of query.columns) {
          const enumName = postgres.getEnumName(col);
          if (enumName) {
            fileEnums.add(enumName);
            usedEnums.add(enumName);
          }
          if (postgres.isJsonColumn(col)) {
            fileUsesJson = true;
          }
        }

        try {
          nodes.push(
            factory.createInterfaceDeclaration(
              [factory.createToken(SyntaxKind.ExportKeyword)],
              factory.createIdentifier(returnIface),
              undefined,
              undefined,
              query.columns.map((column, i) =>
                factory.createPropertySignature(
                  undefined,
                  factory.createIdentifier(colName(i, column)),
                  undefined,
                  postgres.columnType(column),
                ),
              ),
            ),
          );
        } catch (err) {
          throw new Error(
            `Error in query "${query.name}" (${filename}): ${err instanceof Error ? err.message : String(err)}`,
          );
        }
      }

      switch (query.cmd) {
        case ":exec": {
          nodes.push(postgres.execDecl(lowerName, query.text, argIface, query.params));
          break;
        }
        case ":execlastid": {
          nodes.push(postgres.execlastidDecl(lowerName, query.text, argIface, query.params));
          break;
        }
        case ":one": {
          nodes.push(
            postgres.oneDecl(
              lowerName,
              query.text,
              argIface,
              returnIface ?? "void",
              query.params,
              query.columns,
            ),
          );
          break;
        }
        case ":many": {
          nodes.push(
            postgres.manyDecl(
              lowerName,
              query.text,
              argIface,
              returnIface ?? "void",
              query.params,
              query.columns,
            ),
          );
          break;
        }
      }
    }

    // Add shared JSON and enum type declarations at the beginning of the file (after imports)
    const sharedTypeNodes: Node[] = [];
    if (fileUsesJson) {
      sharedTypeNodes.push(...jsonTypeDecls());
    }

    const enumNodes: Node[] = [];
    for (const enumName of fileEnums) {
      const enumDef = enumMap.get(enumName);
      if (enumDef) {
        enumNodes.push(enumTypeDecl(enumName, enumDef));
      }
    }

    // Insert enum declarations after the preamble (imports)
    const preambleLength = postgres.preamble().length;
    nodes.splice(preambleLength, 0, ...sharedTypeNodes, ...enumNodes);

    files.push(
      new File({
        name: `${filename.replace(".", "_")}.ts`,
        contents: new TextEncoder().encode(printNode(nodes)),
      }),
    );
  }

  return new GenerateResponse({
    files: files,
  });
}

// Read input from stdin
function readInput(): GenerateRequest {
  const buffer = readFileSync(STDIO.Stdin);
  return GenerateRequest.fromBinary(buffer);
}

function printNode(nodes: Node[]): string {
  const resultFile = createSourceFile(
    "file.ts",
    "",
    ScriptTarget.Latest,
    /*setParentNodes*/ false,
    ScriptKind.TS,
  );
  const printer = createPrinter({ newLine: NewLineKind.LineFeed });
  let output = "// Code generated by sqlc. DO NOT EDIT.\n\n";
  for (const node of nodes) {
    output += printer.printNode(EmitHint.Unspecified, node, resultFile);
    output += "\n\n";
  }
  return output;
}

// Write output to stdout
function writeOutput(output: GenerateResponse) {
  const encodedOutput = output.toBinary();
  const buffer = new Uint8Array(encodedOutput);
  writeFileSync(STDIO.Stdout, buffer);
}

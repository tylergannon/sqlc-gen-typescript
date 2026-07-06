/**
 * postgres.js driver for sqlc-gen-typescript
 *
 * Generates code using tagged template literals for the postgres npm package.
 * Assumes the connection is configured with:
 *   - transform: postgres.camel (for snake_case -> camelCase)
 *   - types.bigint.parse for bigint -> number conversion
 */

import {
  SyntaxKind,
  NodeFlags,
  TypeNode,
  Expression,
  factory,
  FunctionDeclaration,
} from "typescript";

import { Parameter, Column, Enum } from "../gen/plugin/codegen_pb";
import { argName } from "./utils";

// Map of enum names to their definitions, set by app.ts
let enumMap: Map<string, Enum> = new Map();

/**
 * Set the enum map from the catalog. Called by app.ts before generating code.
 */
export function setEnumMap(map: Map<string, Enum>): void {
  enumMap = map;
}

/**
 * Check if a column type is an enum and return the enum name if so.
 */
export function getEnumName(column?: Column): string | null {
  if (column === undefined || column.type === undefined) {
    return null;
  }
  const typeName = normalizedTypeName(column);
  if (typeName === null) {
    return null;
  }
  if (enumMap.has(typeName)) {
    return typeName;
  }
  return null;
}

function normalizedTypeName(column?: Column): string | null {
  if (column === undefined || column.type === undefined) {
    return null;
  }
  let typeName = column.type.name;
  const pgCatalog = "pg_catalog.";
  if (typeName.startsWith(pgCatalog)) {
    typeName = typeName.slice(pgCatalog.length);
  }
  return typeName.toLowerCase();
}

export function isJsonColumn(column?: Column): boolean {
  const typeName = normalizedTypeName(column);
  return typeName === "json" || typeName === "jsonb";
}

function isScalarJsonColumn(column?: Column): boolean {
  return isJsonColumn(column) && !column?.isArray && (column?.arrayDims ?? 0) === 0;
}

/**
 * Convert snake_case to PascalCase for enum type names
 */
function pascalCase(str: string): string {
  return str
    .split("_")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1).toLowerCase())
    .join("");
}

export function columnType(column?: Column): TypeNode {
  if (column === undefined || column.type === undefined) {
    throw new Error(
      `Missing PostgreSQL type metadata for column "${column?.name || "unknown"}". ` +
        `Try adding an explicit cast or named parameter in your query.`,
    );
  }
  const originalTypeName = column.type.name;
  const lowerTypeName = normalizedTypeName(column);
  if (lowerTypeName === null) {
    throw new Error(
      `Missing PostgreSQL type metadata for column "${column.name || "unknown"}". ` +
        `Try adding an explicit cast or named parameter in your query.`,
    );
  }

  // Check if it's an enum type
  if (enumMap.has(lowerTypeName)) {
    const typ = factory.createTypeReferenceNode(
      factory.createIdentifier(pascalCase(lowerTypeName)),
      undefined,
    );
    if (column.isArray || column.arrayDims > 0) {
      let arrayType: TypeNode = typ;
      const dims = Math.max(column.arrayDims || 1);
      for (let i = 0; i < dims; i++) {
        arrayType = factory.createArrayTypeNode(arrayType);
      }
      if (column.notNull) {
        return arrayType;
      }
      return factory.createUnionTypeNode([
        arrayType,
        factory.createLiteralTypeNode(factory.createNull()),
      ]);
    }
    if (column.notNull) {
      return typ;
    }
    return factory.createUnionTypeNode([typ, factory.createLiteralTypeNode(factory.createNull())]);
  }

  let typ: TypeNode;
  switch (lowerTypeName) {
    // Boolean types
    case "bool":
    case "boolean":
      typ = factory.createKeywordTypeNode(SyntaxKind.BooleanKeyword);
      break;
    // Binary types
    case "bytea":
      typ = factory.createTypeReferenceNode(factory.createIdentifier("Buffer"), undefined);
      break;
    // Date/time types
    case "date":
    case "timestamp":
    case "timestamp without time zone":
    case "timestamptz":
    case "timestamp with time zone":
      typ = factory.createTypeReferenceNode(factory.createIdentifier("Date"), undefined);
      break;
    // Numeric types
    case "float4":
    case "real":
    case "float8":
    case "float":
    case "double precision":
    case "int2":
    case "smallint":
    case "int4":
    case "int":
    case "integer":
    case "int8":
    case "bigint":
    case "bigserial":
    case "serial":
    case "serial2":
    case "serial4":
    case "serial8":
    case "smallserial":
    case "oid":
      typ = factory.createKeywordTypeNode(SyntaxKind.NumberKeyword);
      break;
    // JSON types
    case "json":
    case "jsonb":
      typ = factory.createTypeReferenceNode(factory.createIdentifier("JsonValue"), undefined);
      break;
    // Void type (from functions like pg_advisory_xact_lock)
    case "void":
      typ = factory.createKeywordTypeNode(SyntaxKind.VoidKeyword);
      break;
    // String types - explicitly listed (unambiguously representable as string)
    case "text":
    case "varchar":
    case "character varying":
    case "char":
    case "character":
    case "bpchar":
    case "name":
    case "uuid":
    case "citext":
    case "inet":
    case "cidr":
    case "macaddr":
    case "macaddr8":
    case "money":
    case "numeric":
    case "decimal":
    case "xml":
    case "bit":
    case "varbit":
    case "bit varying":
    case "interval":
    case "time":
    case "time without time zone":
    case "timetz":
    case "time with time zone":
    case "tsvector":
    case "tsquery":
      typ = factory.createKeywordTypeNode(SyntaxKind.StringKeyword);
      break;
    // Geometric types - postgres.js returns these as objects, not strings
    case "point":
    case "line":
    case "lseg":
    case "box":
    case "path":
    case "polygon":
    case "circle":
      throw new Error(
        `Unrecognized PostgreSQL type: "${originalTypeName}". ` +
          `Please add support for this type in sqlc-gen-typescript/src/drivers/postgres.ts`,
      );
    default:
      throw new Error(
        `Unrecognized PostgreSQL type: "${originalTypeName}" for column "${column.name || "unknown"}". ` +
          `This usually means sqlc couldn't infer the type. ` +
          `Try adding an explicit cast like "sqlc.arg(${column.name})::text" or "sqlc.narg('${column.name}')" in your query. ` +
          `If this is a valid PostgreSQL type that needs support, please add it to sqlc-gen-typescript/src/drivers/postgres.ts`,
      );
  }

  if (column.isArray || column.arrayDims > 0) {
    let dims = Math.max(column.arrayDims || 1);
    for (let i = 0; i < dims; i++) {
      typ = factory.createArrayTypeNode(typ);
    }
  }

  if (column.notNull) {
    return typ;
  }
  return factory.createUnionTypeNode([typ, factory.createLiteralTypeNode(factory.createNull())]);
}

export function preamble() {
  return [
    factory.createImportDeclaration(
      undefined,
      factory.createImportClause(
        true, // type-only import
        undefined,
        factory.createNamedImports([
          factory.createImportSpecifier(false, undefined, factory.createIdentifier("Sql")),
        ]),
      ),
      factory.createStringLiteral("postgres"),
      undefined,
    ),
  ];
}

function funcParamsDecl(iface: string | undefined, params: Parameter[]) {
  let funcParams = [
    factory.createParameterDeclaration(
      undefined,
      undefined,
      factory.createIdentifier("sql"),
      undefined,
      factory.createTypeReferenceNode(factory.createIdentifier("Sql"), undefined),
      undefined,
    ),
  ];

  if (iface && params.length > 0) {
    funcParams.push(
      factory.createParameterDeclaration(
        undefined,
        undefined,
        factory.createIdentifier("args"),
        undefined,
        factory.createTypeReferenceNode(factory.createIdentifier(iface), undefined),
        undefined,
      ),
    );
  }

  return funcParams;
}

/**
 * Builds a tagged template literal from SQL text and parameters.
 *
 * Takes SQL like "SELECT * FROM foo WHERE id = $1 AND name = $2"
 * and params [{column: {name: "id"}}, {column: {name: "name"}}]
 * and produces: sql`SELECT * FROM foo WHERE id = ${args.id} AND name = ${args.name}`
 */
function buildTaggedTemplate(queryText: string, params: Parameter[]) {
  // Parse the SQL to find $1, $2, etc. and split into parts
  const parts: string[] = [];
  const expressions: Expression[] = [];

  // Regex to match $1, $2, etc.
  const paramRegex = /\$(\d+)/g;
  let lastIndex = 0;

  for (const match of queryText.matchAll(paramRegex)) {
    // Add the text before this parameter
    parts.push(queryText.slice(lastIndex, match.index));

    // Get the parameter index (1-based in SQL, 0-based in array)
    const paramIndex = parseInt(match[1], 10) - 1;
    const param = params[paramIndex];

    if (param) {
      const arg = factory.createPropertyAccessExpression(
        factory.createIdentifier("args"),
        factory.createIdentifier(argName(paramIndex, param.column)),
      );
      if (isScalarJsonColumn(param.column)) {
        const jsonArg = factory.createCallExpression(
          factory.createPropertyAccessExpression(
            factory.createIdentifier("sql"),
            factory.createIdentifier("json"),
          ),
          undefined,
          [arg],
        );
        if (param.column?.notNull) {
          expressions.push(jsonArg);
        } else {
          expressions.push(
            factory.createConditionalExpression(
              factory.createBinaryExpression(
                arg,
                factory.createToken(SyntaxKind.EqualsEqualsEqualsToken),
                factory.createNull(),
              ),
              factory.createToken(SyntaxKind.QuestionToken),
              factory.createNull(),
              factory.createToken(SyntaxKind.ColonToken),
              jsonArg,
            ),
          );
        }
      } else {
        expressions.push(arg);
      }
    } else {
      // Fallback if param not found (shouldn't happen)
      expressions.push(
        factory.createPropertyAccessExpression(
          factory.createIdentifier("args"),
          factory.createIdentifier(`arg${paramIndex}`),
        ),
      );
    }

    lastIndex = (match.index ?? 0) + match[0].length;
  }

  // Add any remaining text after the last parameter
  parts.push(queryText.slice(lastIndex));

  // Build the tagged template
  // sql`part0${expr0}part1${expr1}part2`
  const head = factory.createTemplateHead(parts[0], parts[0]);
  const spans = expressions.map((expr, i) => {
    const isLast = i === expressions.length - 1;
    const text = parts[i + 1];
    const literal = isLast
      ? factory.createTemplateTail(text, text)
      : factory.createTemplateMiddle(text, text);
    return factory.createTemplateSpan(expr, literal);
  });

  // If no parameters, use a no-substitution template
  if (expressions.length === 0) {
    return factory.createTaggedTemplateExpression(
      factory.createIdentifier("sql"),
      undefined,
      factory.createNoSubstitutionTemplateLiteral(queryText, queryText),
    );
  }

  return factory.createTaggedTemplateExpression(
    factory.createIdentifier("sql"),
    undefined,
    factory.createTemplateExpression(head, spans),
  );
}

export function execDecl(
  funcName: string,
  queryText: string,
  argIface: string | undefined,
  params: Parameter[],
) {
  const funcParams = funcParamsDecl(argIface, params);

  return factory.createFunctionDeclaration(
    [factory.createToken(SyntaxKind.ExportKeyword), factory.createToken(SyntaxKind.AsyncKeyword)],
    undefined,
    factory.createIdentifier(funcName),
    undefined,
    funcParams,
    factory.createTypeReferenceNode(factory.createIdentifier("Promise"), [
      factory.createKeywordTypeNode(SyntaxKind.VoidKeyword),
    ]),
    factory.createBlock(
      [
        factory.createExpressionStatement(
          factory.createAwaitExpression(buildTaggedTemplate(queryText, params)),
        ),
      ],
      true,
    ),
  );
}

export function manyDecl(
  funcName: string,
  queryText: string,
  argIface: string | undefined,
  returnIface: string,
  params: Parameter[],
  _columns: Column[],
) {
  const funcParams = funcParamsDecl(argIface, params);

  // Generate: return await sql<ReturnRow[]>`SELECT ...`
  return factory.createFunctionDeclaration(
    [factory.createToken(SyntaxKind.ExportKeyword), factory.createToken(SyntaxKind.AsyncKeyword)],
    undefined,
    factory.createIdentifier(funcName),
    undefined,
    funcParams,
    factory.createTypeReferenceNode(factory.createIdentifier("Promise"), [
      factory.createArrayTypeNode(
        factory.createTypeReferenceNode(factory.createIdentifier(returnIface), undefined),
      ),
    ]),
    factory.createBlock(
      [
        factory.createReturnStatement(
          factory.createAwaitExpression(
            addTypeArgument(buildTaggedTemplate(queryText, params), returnIface, true),
          ),
        ),
      ],
      true,
    ),
  );
}

export function oneDecl(
  funcName: string,
  queryText: string,
  argIface: string | undefined,
  returnIface: string,
  params: Parameter[],
  _columns: Column[],
) {
  const funcParams = funcParamsDecl(argIface, params);

  // Generate:
  //   const rows = await sql<ReturnRow[]>`SELECT ...`
  //   return rows[0] ?? null
  return factory.createFunctionDeclaration(
    [factory.createToken(SyntaxKind.ExportKeyword), factory.createToken(SyntaxKind.AsyncKeyword)],
    undefined,
    factory.createIdentifier(funcName),
    undefined,
    funcParams,
    factory.createTypeReferenceNode(factory.createIdentifier("Promise"), [
      factory.createUnionTypeNode([
        factory.createTypeReferenceNode(factory.createIdentifier(returnIface), undefined),
        factory.createLiteralTypeNode(factory.createNull()),
      ]),
    ]),
    factory.createBlock(
      [
        // const rows = await sql<ReturnRow[]>`...`
        factory.createVariableStatement(
          undefined,
          factory.createVariableDeclarationList(
            [
              factory.createVariableDeclaration(
                factory.createIdentifier("rows"),
                undefined,
                undefined,
                factory.createAwaitExpression(
                  addTypeArgument(buildTaggedTemplate(queryText, params), returnIface, true),
                ),
              ),
            ],
            NodeFlags.Const,
          ),
        ),
        // return rows[0] ?? null
        factory.createReturnStatement(
          factory.createBinaryExpression(
            factory.createElementAccessExpression(
              factory.createIdentifier("rows"),
              factory.createNumericLiteral("0"),
            ),
            factory.createToken(SyntaxKind.QuestionQuestionToken),
            factory.createNull(),
          ),
        ),
      ],
      true,
    ),
  );
}

/**
 * Add a type argument to a tagged template expression.
 * Transforms: sql`...` into sql<Type[]>`...`
 */
function addTypeArgument(
  taggedTemplate: ReturnType<typeof factory.createTaggedTemplateExpression>,
  typeName: string,
  isArray: boolean,
) {
  let typeArg: TypeNode = factory.createTypeReferenceNode(
    factory.createIdentifier(typeName),
    undefined,
  );
  if (isArray) {
    typeArg = factory.createArrayTypeNode(typeArg);
  }

  return factory.createTaggedTemplateExpression(
    taggedTemplate.tag,
    [typeArg],
    taggedTemplate.template,
  );
}

export function execlastidDecl(
  _funcName: string,
  _queryText: string,
  _argIface: string | undefined,
  _params: Parameter[],
): FunctionDeclaration {
  throw new Error("postgres driver does not support :execlastid");
}

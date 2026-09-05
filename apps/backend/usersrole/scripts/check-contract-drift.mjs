#!/usr/bin/env node
import { readFileSync, existsSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { parse as parseYaml } from 'yaml';

const USERSROLE_ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const CONTRACTS_DIR = resolve(USERSROLE_ROOT, 'docs/contracts');
const LEDGER = resolve(CONTRACTS_DIR, 'served-document-deviations.json');
const DEFAULT_SERVED = resolve(
  USERSROLE_ROOT,
  'build/openapi/served-api-docs.json',
);
const EXTRACT_COMMAND =
  'bash gradlew test --tests com.jdw.usersrole.contracts.ServedOpenApiDocumentDumpTests';

const SPECS = [
  {
    service: 'identity-service',
    file: 'identity-service.openapi.yaml',
    expectedOperations: 18,
  },
  {
    service: 'profile-service',
    file: 'profile-service.openapi.yaml',
    expectedOperations: 15,
  },
];

const METHODS = [
  'get',
  'put',
  'post',
  'delete',
  'patch',
  'head',
  'options',
  'trace',
];

// springdoc emits `*/*` for every handler that declares no `produces`. The
// application configures no content negotiation beyond Jackson, so that is
// application/json on the wire; comparing the two literally would flag all 33.
const SERVED_WILDCARD_MEDIA_TYPE = '*/*';
const RESOLVED_WILDCARD_MEDIA_TYPE = 'application/json';

const REQUIRED_LEDGER_KEYS = ['globalRules', 'operations', 'schemas'];

const HELP = `check-contract-drift — compare the frozen service contracts to the served springdoc document

  node apps/backend/usersrole/scripts/check-contract-drift.mjs [--served <path>]

Fails when the frozen contracts and the served document differ in a way that
docs/contracts/served-document-deviations.json does not account for, when an
entry in that file no longer applies, and when a frozen operation's error-status
set has drifted from the set pinned there. Exit 0 clean, 1 on drift.

  --served <path>   served document JSON (default: build/openapi/served-api-docs.json)
  --help            this text

Produce the served document with:
  ${EXTRACT_COMMAND}
`;

function fail(message, help = []) {
  process.stdout.write(`FAIL ${message}\n`);
  for (const line of help) process.stdout.write(`  next: ${line}\n`);
  process.exit(1);
}

function operationKey(method, path) {
  return `${method.toUpperCase()} ${path}`;
}

function collectOperations(document) {
  const operations = new Map();
  for (const [path, item] of Object.entries(document.paths ?? {})) {
    for (const method of METHODS) {
      if (item[method])
        operations.set(operationKey(method, path), item[method]);
    }
  }
  return operations;
}

function statusesIn(operation, predicate) {
  return Object.keys(operation.responses ?? {})
    .filter((code) => predicate(Number(code)))
    .sort();
}

const isSuccess = (code) => code >= 200 && code < 300;
const isError = (code) => code >= 400;

/**
 * A stable one-line identity for a schema, so a response whose type is swapped
 * for another is caught without diffing whole subtrees. `$ref` names carry the
 * signal; primitives fall back to type and format.
 */
function schemaSignature(schema) {
  if (!schema) return 'none';
  if (schema.$ref) return `ref:${schema.$ref.split('/').pop()}`;
  if (schema.type === 'array') return `array:${schemaSignature(schema.items)}`;
  if (schema.oneOf)
    return `oneOf:${schema.oneOf.map(schemaSignature).sort().join('|')}`;
  const type = Array.isArray(schema.type)
    ? schema.type.slice().sort().join('|')
    : (schema.type ?? 'object');
  return schema.format ? `${type}/${schema.format}` : type;
}

/** Resolves one level of local `$ref`, which is all these documents use. */
function deref(node, document) {
  if (!node?.$ref) return node;
  return node.$ref
    .replace(/^#\//, '')
    .split('/')
    .reduce((current, segment) => current?.[segment], document);
}

/**
 * Media type to schema signature for one response, with springdoc's wildcard
 * media type normalized to what the service actually serves.
 */
function responseShape(response, document) {
  const resolved = deref(response, document) ?? {};
  const shape = new Map();
  for (const [mediaType, content] of Object.entries(resolved.content ?? {})) {
    const key =
      mediaType === SERVED_WILDCARD_MEDIA_TYPE
        ? RESOLVED_WILDCARD_MEDIA_TYPE
        : mediaType;
    shape.set(key, schemaSignature(content.schema));
  }
  return shape;
}

function formatShape(shape) {
  if (shape.size === 0) return 'no content';
  return [...shape.entries()]
    .map(([mediaType, signature]) => `${mediaType}=${signature}`)
    .sort()
    .join(', ');
}

function requestMediaTypes(operation) {
  return Object.keys(operation.requestBody?.content ?? {}).sort();
}

/**
 * Resolves the local $refs the frozen specs use for shared parameters. The
 * served document inlines everything, so without this every $ref would read as
 * a removed parameter.
 */
function parameterNames(operation, document) {
  return (operation.parameters ?? [])
    .map((parameter) => deref(parameter, document)?.name)
    .filter(Boolean)
    .sort();
}

function difference(left, right) {
  return left.filter((entry) => !right.includes(entry));
}

const argv = process.argv.slice(2);
if (argv.includes('--help') || argv.includes('-h')) {
  process.stdout.write(HELP);
  process.exit(0);
}
const servedIndex = argv.indexOf('--served');
const servedPath = servedIndex === -1 ? DEFAULT_SERVED : argv[servedIndex + 1];

if (!servedPath || !existsSync(servedPath)) {
  fail(`no served document at ${servedPath}`, [
    `cd apps/backend/usersrole && ${EXTRACT_COMMAND}`,
  ]);
}

const served = JSON.parse(readFileSync(servedPath, 'utf8'));
const deviations = JSON.parse(readFileSync(LEDGER, 'utf8'));

for (const key of REQUIRED_LEDGER_KEYS) {
  const value = deviations[key];
  if (!value || typeof value !== 'object') {
    fail(`the deviation ledger has no usable "${key}"`, [
      `add a "${key}" object to docs/contracts/served-document-deviations.json`,
    ]);
  }
}

const frozen = SPECS.map((spec) => ({
  ...spec,
  document: parseYaml(readFileSync(resolve(CONTRACTS_DIR, spec.file), 'utf8')),
}));

const servedOperations = collectOperations(served);
const problems = [];
const unusedDeviations = new Set(Object.keys(deviations.operations));
const unusedSchemaDeviations = new Set(Object.keys(deviations.schemas));

// 1. Coverage: every served operation frozen exactly once, and nothing invented.
const owners = new Map();
for (const spec of frozen) {
  const operations = collectOperations(spec.document);
  if (operations.size !== spec.expectedOperations) {
    problems.push(
      `${spec.service} declares ${operations.size} operations, expected ${spec.expectedOperations}`,
    );
  }
  for (const key of operations.keys()) {
    if (owners.has(key)) {
      problems.push(
        `${key} appears in both ${owners.get(key).service} and ${spec.service}`,
      );
    }
    owners.set(key, {
      service: spec.service,
      operation: operations.get(key),
      document: spec.document,
    });
  }
}

for (const key of servedOperations.keys()) {
  if (!owners.has(key))
    problems.push(`${key} is served but frozen in neither contract`);
}
for (const key of owners.keys()) {
  if (!servedOperations.has(key))
    problems.push(`${key} is frozen but not served`);
}

// 2. Per-operation: the ledger pins what the served document cannot show, and
//    accounts for every difference from what it can.
for (const [key, { operation, document }] of owners) {
  const servedOperation = servedOperations.get(key);
  const recorded = deviations.operations[key];
  if (recorded) unusedDeviations.delete(key);

  // 2a. The error contract. springdoc documents no error response at all, so
  //     there is nothing to diff against — it is pinned in the ledger instead,
  //     which makes an edit to either file have to be mirrored in the other.
  const errorsFrozen = statusesIn(operation, isError);
  if (!recorded) {
    problems.push(
      `${key} has no ledger entry, so its error contract [${errorsFrozen.join(', ')}] is unpinned`,
    );
  } else {
    const errorsRecorded = (recorded.errorStatuses ?? []).slice().sort();
    if (errorsFrozen.join(',') !== errorsRecorded.join(',')) {
      problems.push(
        `${key} declares error statuses [${errorsFrozen.join(', ')}], ledger pins [${errorsRecorded.join(', ')}]`,
      );
    }
  }

  if (!servedOperation) continue;

  // 2b. Success status.
  const successFrozen = statusesIn(operation, isSuccess);
  const successServed = statusesIn(servedOperation, isSuccess);
  if (successFrozen.join(',') !== successServed.join(',')) {
    const expected = recorded?.successStatus;
    if (
      !expected ||
      expected.served !== successServed.join(',') ||
      expected.frozen !== successFrozen.join(',')
    ) {
      problems.push(
        `${key} success status ${successServed.join(',')} served vs ${successFrozen.join(',')} frozen, not recorded`,
      );
    }
  } else if (recorded?.successStatus) {
    problems.push(
      `${key} records a success-status deviation that no longer applies`,
    );
  }

  // 2c. Success response media types and schema identity. Without this a
  //     response retyped from one model to another passes every other check.
  const shapeFrozen = responseShape(
    operation.responses?.[successFrozen[0]],
    document,
  );
  const shapeServed = responseShape(
    servedOperation.responses?.[successServed[0]],
    served,
  );
  if (formatShape(shapeFrozen) !== formatShape(shapeServed)) {
    const expected = recorded?.responseShape;
    if (
      !expected ||
      expected.served !== formatShape(shapeServed) ||
      expected.frozen !== formatShape(shapeFrozen)
    ) {
      problems.push(
        `${key} success response ${formatShape(shapeServed)} served vs ${formatShape(shapeFrozen)} frozen, not recorded`,
      );
    }
  } else if (recorded?.responseShape) {
    problems.push(
      `${key} records a response-shape deviation that no longer applies`,
    );
  }

  // 2d. Request media types.
  const requestFrozen = requestMediaTypes(operation);
  const requestServed = requestMediaTypes(servedOperation);
  if (requestFrozen.join(',') !== requestServed.join(',')) {
    const expected = recorded?.requestMediaType;
    if (
      !expected ||
      expected.served !== requestServed.join(',') ||
      expected.frozen !== requestFrozen.join(',')
    ) {
      problems.push(
        `${key} request media type ${requestServed.join(',') || 'none'} served vs ${requestFrozen.join(',') || 'none'} frozen, not recorded`,
      );
    }
  } else if (recorded?.requestMediaType) {
    problems.push(
      `${key} records a request-media-type deviation that no longer applies`,
    );
  }

  // 2e. Parameters.
  const parametersFrozen = parameterNames(operation, document);
  const parametersServed = parameterNames(servedOperation, served);
  const added = difference(parametersFrozen, parametersServed);
  const removed = difference(parametersServed, parametersFrozen);
  const recordedAdded = (recorded?.parametersAdded ?? []).slice().sort();
  const recordedRemoved = (recorded?.parametersRemoved ?? []).slice().sort();
  if (added.join(',') !== recordedAdded.join(',')) {
    problems.push(
      `${key} adds parameters [${added.join(', ')}], recorded [${recordedAdded.join(', ')}]`,
    );
  }
  if (removed.join(',') !== recordedRemoved.join(',')) {
    problems.push(
      `${key} drops parameters [${removed.join(', ')}], recorded [${recordedRemoved.join(', ')}]`,
    );
  }
}

for (const key of unusedDeviations) {
  problems.push(
    `the ledger records ${key}, which is not an operation in either contract`,
  );
}

// 3. Schema property sets, for the schemas both documents name.
for (const spec of frozen) {
  const frozenSchemas = spec.document.components?.schemas ?? {};
  const servedSchemas = served.components?.schemas ?? {};
  for (const [name, schema] of Object.entries(frozenSchemas)) {
    const key = `${spec.service}:${name}`;
    if (!servedSchemas[name]) {
      // Defined by the contract but absent from the served document — nothing to
      // diff, so a ledger entry here would assert nothing.
      if (deviations.schemas[key]) {
        unusedSchemaDeviations.delete(key);
        problems.push(
          `the ledger records schema ${key}, but the served document does not define ${name}, so the entry asserts nothing`,
        );
      }
      continue;
    }
    const recorded = deviations.schemas[key];
    if (recorded) unusedSchemaDeviations.delete(key);
    const frozenProperties = Object.keys(schema.properties ?? {}).sort();
    const servedProperties = Object.keys(
      servedSchemas[name].properties ?? {},
    ).sort();
    const added = difference(frozenProperties, servedProperties);
    const removed = difference(servedProperties, frozenProperties);
    const recordedAdded = (recorded?.propertiesAdded ?? []).slice().sort();
    const recordedRemoved = (recorded?.propertiesRemoved ?? []).slice().sort();
    if (added.join(',') !== recordedAdded.join(',')) {
      problems.push(
        `${key} adds properties [${added.join(', ')}], recorded [${recordedAdded.join(', ')}]`,
      );
    }
    if (removed.join(',') !== recordedRemoved.join(',')) {
      problems.push(
        `${key} drops properties [${removed.join(', ')}], recorded [${recordedRemoved.join(', ')}]`,
      );
    }
  }
}

for (const key of unusedSchemaDeviations) {
  problems.push(
    `the ledger records schema ${key}, which no contract defines under that name`,
  );
}

// 4. Every ledger entry has to assert something. An entry carrying only a
//    reason reads as accounted-for while checking nothing.
const ASSERTING_OPERATION_KEYS = [
  'errorStatuses',
  'successStatus',
  'responseShape',
  'requestMediaType',
  'parametersAdded',
  'parametersRemoved',
];
const ASSERTING_SCHEMA_KEYS = ['propertiesAdded', 'propertiesRemoved'];

for (const [key, entry] of Object.entries(deviations.operations)) {
  if (!ASSERTING_OPERATION_KEYS.some((field) => field in entry)) {
    problems.push(`the ledger entry for ${key} asserts nothing`);
  }
}
for (const [key, entry] of Object.entries(deviations.schemas)) {
  if (!ASSERTING_SCHEMA_KEYS.some((field) => field in entry)) {
    problems.push(`the ledger entry for schema ${key} asserts nothing`);
  }
}

// 5. The password contract, asserted against both documents independently of
//    the ledger — this is the one property no recorded deviation may excuse.
function findPasswordCarriers(schemas) {
  const carriers = [];
  for (const [name, schema] of Object.entries(schemas ?? {})) {
    const password = schema.properties?.password;
    if (password && password.writeOnly !== true) carriers.push(name);
  }
  return carriers;
}

for (const spec of frozen) {
  const leaks = findPasswordCarriers(spec.document.components?.schemas);
  if (leaks.length) {
    problems.push(
      `${spec.service} exposes a readable password on [${leaks.join(', ')}]`,
    );
  }
}
const servedLeaks = findPasswordCarriers(served.components?.schemas);
if (servedLeaks.length) {
  problems.push(
    `the served document exposes a readable password on [${servedLeaks.join(', ')}]`,
  );
}

if (problems.length) {
  process.stdout.write(
    `FAIL ${problems.length} unaccounted difference(s) between the frozen contracts and the served document\n`,
  );
  for (const problem of problems) process.stdout.write(`  - ${problem}\n`);
  process.stdout.write(
    '  next: fix the contract, or record the difference in docs/contracts/served-document-deviations.json with a reason\n',
  );
  process.exit(1);
}

const counts = frozen.map(
  (spec) => `${spec.service}=${collectOperations(spec.document).size}`,
);
const pinnedErrors = Object.values(deviations.operations).reduce(
  (total, entry) => total + (entry.errorStatuses?.length ?? 0),
  0,
);
process.stdout.write(
  `OK ${servedOperations.size} served operations, all frozen exactly once (${counts.join(', ')})\n`,
);
process.stdout.write(
  `OK success status, response media type and schema, request media type and parameters match the served document or a recorded deviation\n`,
);
process.stdout.write(
  `OK ${pinnedErrors} error responses across ${Object.keys(deviations.operations).length} operations match the pinned contract\n`,
);
process.stdout.write(
  'OK no schema in either contract exposes a readable password\n',
);

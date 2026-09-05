#!/usr/bin/env node
import { readFileSync, existsSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { parse as parseYaml } from 'yaml';

const USERSROLE_ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const CONTRACTS_DIR = resolve(USERSROLE_ROOT, 'docs/contracts');
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

const HELP = `check-contract-drift — compare the frozen service contracts to the served springdoc document

  node apps/backend/usersrole/scripts/check-contract-drift.mjs [--served <path>]

Fails when the frozen contracts and the served document differ in a way that
docs/contracts/served-document-deviations.json does not account for, and when an
entry in that file no longer applies. Exit 0 clean, 1 on drift.

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

function statusesOf(operation, predicate) {
  return Object.keys(operation.responses ?? {})
    .filter((code) => predicate(Number(code)))
    .sort();
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
    .map((parameter) => {
      if (!parameter.$ref) return parameter.name;
      const target = parameter.$ref.replace(/^#\//, '').split('/');
      return target.reduce((node, segment) => node?.[segment], document)?.name;
    })
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
const deviations = JSON.parse(
  readFileSync(
    resolve(CONTRACTS_DIR, 'served-document-deviations.json'),
    'utf8',
  ),
);

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

// 2. Per-operation deviations, each accounted for or reported.
for (const [key, { operation, document }] of owners) {
  const servedOperation = servedOperations.get(key);
  if (!servedOperation) continue;
  const recorded = deviations.operations[key];
  if (recorded) unusedDeviations.delete(key);

  const successFrozen = statusesOf(
    operation,
    (code) => code >= 200 && code < 300,
  );
  const successServed = statusesOf(
    servedOperation,
    (code) => code >= 200 && code < 300,
  );
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
    `served-document-deviations.json records ${key}, which is not an operation`,
  );
}

// 3. Schema property sets, for the schemas both documents name.
for (const spec of frozen) {
  const frozenSchemas = spec.document.components?.schemas ?? {};
  const servedSchemas = served.components?.schemas ?? {};
  for (const [name, schema] of Object.entries(frozenSchemas)) {
    if (!servedSchemas[name]) continue;
    const key = `${spec.service}:${name}`;
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
    `served-document-deviations.json records schema ${key}, which neither contract defines`,
  );
}

// 4. The password contract, asserted against both documents independently of
//    the deviation ledger — this is the one property no recorded deviation may
//    ever excuse.
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
process.stdout.write(
  `OK ${servedOperations.size} served operations, all frozen exactly once (${counts.join(', ')})\n`,
);
process.stdout.write(
  `OK ${Object.keys(deviations.operations).length} operation and ${Object.keys(deviations.schemas).length} schema deviation(s) recorded, all still applicable\n`,
);
process.stdout.write(
  'OK no schema in either contract exposes a readable password\n',
);

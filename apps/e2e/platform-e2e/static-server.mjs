import { createReadStream } from 'node:fs';
import { realpath, stat } from 'node:fs/promises';
import { createServer } from 'node:http';
import {
  extname,
  isAbsolute,
  join,
  normalize,
  relative,
  resolve,
} from 'node:path';

/**
 * Origins the suite addresses. The host is listed last on purpose: the servers
 * are bound in order, so the host port accepting a connection means every
 * remote is already accepting one too. That lets a single readiness probe stand
 * in for all four and keeps the test runner from starting against a half-open
 * platform.
 *
 * The ports are a contract, not a preference — the host's federation manifest
 * and the suite's service-discovery mock both address the remotes by absolute
 * origin, so a server that lands anywhere else is unreachable to the tests.
 */
const ORIGINS = [
  { port: 4201, root: 'dist/apps/frontend/authui' },
  { port: 4202, root: 'dist/apps/frontend/usersui' },
  { port: 4203, root: 'dist/apps/frontend/rolesui' },
  { port: 4200, root: 'dist/apps/frontend/container' },
];

const CONTENT_TYPES = new Map([
  ['.css', 'text/css; charset=utf-8'],
  ['.html', 'text/html; charset=utf-8'],
  ['.ico', 'image/x-icon'],
  ['.jpg', 'image/jpeg'],
  ['.js', 'text/javascript; charset=utf-8'],
  ['.json', 'application/json; charset=utf-8'],
  ['.map', 'application/json; charset=utf-8'],
  ['.mjs', 'text/javascript; charset=utf-8'],
  ['.png', 'image/png'],
  ['.svg', 'image/svg+xml'],
  ['.txt', 'text/plain; charset=utf-8'],
  ['.webp', 'image/webp'],
  ['.woff2', 'font/woff2'],
]);

/**
 * Absolute candidate path for a request, or null if it cannot be decoded.
 *
 * Containment is deliberately *not* decided here. The check is spelled out at
 * each filesystem call instead, because a containment test is only worth
 * something where it dominates the call it protects — reading it three lines
 * above a `stat` says more, to a reader and to static analysis both, than
 * reading it as the return value of a function defined elsewhere.
 */
function candidatePath(root, urlPath) {
  let decoded;
  try {
    decoded = decodeURIComponent(urlPath.split('?')[0]);
  } catch {
    // A stray `%` is a client mistake, not a server one. Left to propagate it
    // rejects out of an async handler, which has terminated the process since
    // Node 15 and would take every origin down mid-run.
    return null;
  }
  return resolve(join(root, normalize(decoded)));
}

async function fileAt(root, path) {
  const escapes = relative(root, path);
  if (escapes.startsWith('..') || isAbsolute(escapes)) {
    return null;
  }
  try {
    const stats = await stat(path);
    if (!stats.isFile()) {
      return null;
    }
    // Containment of the requested path says nothing about where a symlink
    // inside the tree points, so the resolved target has to satisfy it too.
    const linked = relative(root, await realpath(path));
    if (linked.startsWith('..') || isAbsolute(linked)) {
      return null;
    }
    return stats;
  } catch {
    return null;
  }
}

function send(response, status, headers, body) {
  response.writeHead(status, headers);
  if (body === undefined) {
    response.end();
  } else if (typeof body === 'string') {
    response.end(body);
  } else {
    body.on('error', () => response.destroy());
    body.pipe(response);
  }
}

function handler(root) {
  return async (request, response) => {
    if (request.method !== 'GET' && request.method !== 'HEAD') {
      send(response, 405, { allow: 'GET, HEAD' });
      return;
    }

    const requested = candidatePath(root, request.url ?? '/');
    if (requested === null) {
      send(response, 400, { 'content-type': 'text/plain' }, 'Bad request\n');
      return;
    }
    let target = requested;
    let stats = await fileAt(root, target);
    if (!stats) {
      target = join(requested, 'index.html');
      stats = await fileAt(root, target);
    }
    // Client-side routes carry no extension, so they fall back to the shell.
    // Anything that names a file does not: a missing bundle or manifest has to
    // surface as a 404 rather than as HTML that the caller then fails to parse
    // somewhere further downstream.
    if (!stats && !extname(requested)) {
      target = join(root, 'index.html');
      stats = await fileAt(root, target);
    }
    if (!stats) {
      send(response, 404, { 'content-type': 'text/plain' }, 'Not found\n');
      return;
    }

    const headers = {
      'content-type':
        CONTENT_TYPES.get(extname(target).toLowerCase()) ??
        'application/octet-stream',
      'content-length': stats.size,
      // The host loads every remote entry and chunk cross-origin.
      'access-control-allow-origin': '*',
      // The output directory is rewritten between runs while the ports stay
      // the same, so anything cached across runs is a stale build.
      'cache-control': 'no-store',
    };

    if (request.method === 'HEAD') {
      send(response, 200, headers);
      return;
    }
    // Nothing that reaches here should be able to fail this — every path above
    // was contained before it was stat'd. It is restated at the point the file
    // is actually opened because that is the call that hands bytes back to the
    // caller, and it costs a string compare to say so at the call itself.
    const escapes = relative(root, target);
    if (escapes.startsWith('..') || isAbsolute(escapes)) {
      send(response, 403, { 'content-type': 'text/plain' }, 'Forbidden\n');
      return;
    }
    send(response, 200, headers, createReadStream(target));
  };
}

function listen(port, root) {
  return new Promise((resolvePort, reject) => {
    const server = createServer(handler(resolve(root)));
    // A client that navigates away mid-download resets its socket; that is
    // routine under a parallel test run and must not take the server with it.
    server.on('clientError', (_error, socket) => socket.destroy());
    // Node closes an idle keep-alive socket after 5s, which is short enough to
    // land on a client writing its next request onto that socket — exactly the
    // ECONNRESET this server exists to avoid. The header timeout has to stay
    // above the keep-alive one, or a socket is torn down while the headers it
    // was reopened for are still arriving.
    server.keepAliveTimeout = 65_000;
    server.headersTimeout = 70_000;
    const rejectOnStartupError = (error) => reject(error);
    server.once('error', rejectOnStartupError);
    // Bound without a host so both loopback families answer — callers reach
    // these by name, and `localhost` resolves to ::1 first on some platforms
    // and 127.0.0.1 first on others.
    server.listen(port, () => {
      // Past this point the promise is settled, so an error routed to `reject`
      // would be discarded in silence and leave the suite talking to an origin
      // that has stopped answering.
      server.off('error', rejectOnStartupError);
      server.on('error', (error) => {
        process.stderr.write(
          `server for ${root} on port ${port} failed: ${error.stack ?? error.message}\n`,
        );
        process.exit(1);
      });
      process.stdout.write(`serving ${root} on http://localhost:${port}\n`);
      resolvePort(server);
    });
  });
}

const servers = [];
for (const { port, root } of ORIGINS) {
  try {
    servers.push(await listen(port, root));
  } catch (error) {
    // Relocating to a free port would leave the tests addressing an origin
    // nothing is listening on, which surfaces as unrelated navigation failures
    // deep in the run instead of here.
    process.stderr.write(
      `cannot serve ${root} on port ${port}: ${error.message}\n`,
    );
    process.exit(1);
  }
}

for (const signal of ['SIGINT', 'SIGTERM']) {
  process.on(signal, () => {
    for (const server of servers) server.close();
    process.exit(0);
  });
}

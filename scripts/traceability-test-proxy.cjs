// Disposable loopback HTTPS ingress for real Keycloak and Web acceptance tests.
// No TLS verification override, external targets, or identity headers.
const fs = require('node:fs');
const http = require('node:http');
const https = require('node:https');
const config = JSON.parse(fs.readFileSync(process.argv[2], 'utf8'));
const server = https.createServer({ key: fs.readFileSync(config.key), cert: fs.readFileSync(config.cert) }, (request, response) => {
  const storage = /^\/om-(?:catalog|traceability|fulfillment|documents|private)(?:\/|\?)/.test(request.url);
  const port = storage ? config.storage : request.url.startsWith('/realms/') || request.url.startsWith('/resources/') ? config.keycloak : config.web;
  const upstream = http.request({ hostname: '127.0.0.1', port, path: request.url, method: request.method,
    headers: { ...request.headers, host: storage ? request.headers.host : `127.0.0.1:${port}`, 'x-forwarded-proto': 'https', 'x-forwarded-host': request.headers.host } }, (reply) => {
    response.writeHead(reply.statusCode, reply.headers); reply.pipe(response);
  });
  upstream.setTimeout(15000, () => upstream.destroy());
  upstream.on('error', () => { if (!response.headersSent) response.writeHead(503); response.end(); });
  request.pipe(upstream);
});
server.listen(config.port, '127.0.0.1');

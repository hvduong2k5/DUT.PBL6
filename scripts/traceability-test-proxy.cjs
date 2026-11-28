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
// Next development bootstrap also needs its HMR WebSocket through HTTPS.
server.on('upgrade', (request, client, head) => {
  const upstream = http.request({ hostname: '127.0.0.1', port: config.web,
    path: request.url, method: request.method,
    headers: { ...request.headers, host: `127.0.0.1:${config.web}`,
      'x-forwarded-proto': 'https', 'x-forwarded-host': request.headers.host } });
  upstream.on('upgrade', (reply, socket, upstreamHead) => {
    let headers = `HTTP/1.1 ${reply.statusCode} ${reply.statusMessage}\r\n`;
    for (let index = 0; index < reply.rawHeaders.length; index += 2) headers += `${reply.rawHeaders[index]}: ${reply.rawHeaders[index + 1]}\r\n`;
    client.write(headers + '\r\n');
    if (head.length) socket.write(head);
    if (upstreamHead.length) client.write(upstreamHead);
    socket.on('error', () => client.destroy());
    client.on('error', () => socket.destroy());
    client.pipe(socket).pipe(client);
  });
  upstream.on('response', () => client.destroy());
  upstream.on('error', () => client.destroy());
  upstream.end();
});
server.listen(config.port, '127.0.0.1');

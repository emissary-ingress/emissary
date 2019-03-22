'use strict';

const http = require('http');
const PORT = process.env.PORT || '3000';

let state = 0;

const requestHandler = (request, response) => {
  if (state === 0) {
    response.writeHead(503, {'Content-Type': 'text/plain'});
    response.end('ERROR');
  } else {
    response.writeHead(200, {'Content-Type': 'text/plain'});
    response.end('OK');
  }
  state = (state + 1) % 2;
};

const server = http.createServer(requestHandler);

server.listen(PORT, () => {
  console.log(`server is listening on ${PORT}`)
});

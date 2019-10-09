const {EchoRequest} = require('./echo_pb.js');
const {EchoServiceClient} = require('./echo_grpc_web_pb.js');

const grpc = {};
grpc.web = require('grpc-web');

var echoService = new EchoServiceClient('http://localhost:7080', null, null);
const request = new EchoRequest();

function logMapElements(value, key, map) {
    console.log(` [ ${key}  :  ${value} ]`);
}

echoService.echo(request, {'requested-status': 0}, function(err, response) {
    if (err) {
        console.log("Response error code:", err.code);
        console.log("Response error message:", err.message);
    } else {
        console.log("\nRequest header map:");
        response.getRequest().getHeadersMap().forEach(logMapElements);
        
        console.log("\nResponse header map:");
        response.getResponse().getHeadersMap().forEach(logMapElements);
    }
}).on('status', function(status) {
    console.log("\nExpected code:", 0);
    console.log("\nEcho service responded: ", status.code);
});

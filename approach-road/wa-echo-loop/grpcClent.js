const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');
const path = require('path');

// Go from 'approach-road/discord/' 
// Up to 'approach-road/' (1)
// Up to 'indieclaw/' (2)
// Down to 'gateway-service/proto/gateway.proto'
const PROTO_PATH = path.resolve(__dirname, '../../gateway-service/proto/gateway/v1/gateway.proto');

console.log("Looking for proto at:", PROTO_PATH);

const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
    keepCase: true,
    longs: String,
    enums: String,
    defaults: true, // Crucial for missing fields
    oneofs: true,
    includeDirs: [path.resolve(__dirname, '../../gateway-service/proto')] // Allows imports
});

const gatewayProto = grpc.loadPackageDefinition(packageDefinition);
// Access the service correctly based on your package 'gateway.v1'
const client = new gatewayProto.gateway.v1.GatewayService(
    '127.0.0.1:8080', 
    grpc.credentials.createInsecure()
);

// Export the client directly with a helper for readability
module.exports = {
    chat: (request, callback) => {
        client.Chat(request, callback);
    }
};
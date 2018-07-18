# grpc

# ConnLogger
ConnLogger is a utility for asynchronously logging the gRPC client connection. This can be helpful especially in local development environments.

### Usage
```
	// dial trainee grpc address with the context async. this method does not block. we set an exponential backoff if the client
	// connection cannot be made with max set to CLIENT_BACKOFF env variable.
	conn, err := grpc.DialContext(context.Background(), "localhost:8001".WithInsecure(), grpc.WithBackoffMaxDelay(time.Duration(10) * time.Second))

	// init client connect logger
	cl := gt.NewConnLogger()
	cl.AddConn("Service", conn)
```

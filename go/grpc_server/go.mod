module grpc_server

go 1.26

require (
	github.com/matsuridayo/libneko v1.0.0 // replaced
	google.golang.org/grpc v1.81.1
	google.golang.org/protobuf v1.36.11
)

require github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.3

require (
	golang.org/x/net v0.54.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260511170946-3700d4141b60 // indirect
)

replace github.com/matsuridayo/libneko v1.0.0 => ../../../libneko

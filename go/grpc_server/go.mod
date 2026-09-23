module grpc_server

go 1.26.0

require (
	github.com/matsuridayo/libneko v1.0.0 // replaced
	google.golang.org/grpc v1.85.0-dev.0.20260825072537-93e31b48545e
	google.golang.org/protobuf v1.36.12
)

require github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.3

require (
	golang.org/x/net v0.59.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260817212433-ac3dfec99bb1 // indirect
)

replace github.com/matsuridayo/libneko v1.0.0 => ../../../libneko

module grpc_server

go 1.24.7

require (
	github.com/matsuridayo/libneko v1.0.0 // replaced
	google.golang.org/grpc v1.79.1
	google.golang.org/protobuf v1.36.11
)

require github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.3

require (
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto v0.0.0-20211223182754-3ac035c7e7cb // indirect
)

replace github.com/matsuridayo/libneko v1.0.0 => ../../../libneko

# Identity demo using hydra and grpc

Usage:

start hydra server:

```
FORCE_ROOT_CLIENT_CREDENTIALS="tthanh:secret" hydra host
```

start grpc server:

```
go run cmd/server/main.go
```

create new clients:

```
go run cmd/client/main.go register username password
```

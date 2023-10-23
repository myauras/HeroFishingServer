module matchgame

replace herofishingGoModule => ../herofishingGoModule // for local

// replace herofishingGoModule => /home/herofishingGoModule // for docker

go 1.21.0

require (
	agones.dev/agones v1.35.0
	github.com/sirupsen/logrus v1.9.3
	herofishingGoModule v0.0.0-00010101000000-000000000000
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.18.0 // indirect
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/montanaflynn/stats v0.0.0-20171201202039-1bf9dbcd8cbe // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	go.mongodb.org/mongo-driver v1.12.1 // indirect
	golang.org/x/crypto v0.14.0 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sync v0.3.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/genproto v0.0.0-20231009173412-8bfb1ae86b6c // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20231009173412-8bfb1ae86b6c // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231009173412-8bfb1ae86b6c // indirect
	google.golang.org/grpc v1.58.3 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)

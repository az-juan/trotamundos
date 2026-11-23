module trotamundos/filius

go 1.24.0

require (
	github.com/go-chi/chi/v5 v5.2.1
	trotamundos/shared v0.0.0
)

require (
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/nats-io/nats.go v1.39.0 // indirect
	github.com/nats-io/nkeys v0.4.9 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
)

replace trotamundos/shared => ../../shared

FROM golang:1.26-bookworm AS builder

RUN apt-get update && apt-get install -y \
    clang \
    llvm \
    libbpf-dev \
    linux-headers-generic \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install github.com/a-h/templ/cmd/templ@latest
RUN go install github.com/cilium/ebpf/cmd/bpf2go@latest

RUN templ generate
RUN go generate ./...

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /zeptor ./cmd/zeptor

FROM gcr.io/distroless/static-debian12

COPY --from=builder /zeptor /zeptor

EXPOSE 3000

ENTRYPOINT ["/zeptor"]

FROM golang:1.24.1-alpine3.21

WORKDIR /mitm_proxy

RUN apk add --no-cache openssl ca-certificates

COPY go.mod ./
RUN go mod download

COPY cert.key ca.key ca.crt ./
COPY gen_cert.sh ./

COPY . ./

RUN mkdir -p /usr/local/share/ca-certificates && \
    cp ca.crt /usr/local/share/ca-certificates/mitm-proxy-ca.crt && \
    update-ca-certificates --fresh && \
    go build -o mitm_proxy ./cmd

EXPOSE 8080
CMD ["./mitm_proxy"]
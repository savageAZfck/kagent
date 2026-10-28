FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go/ ./go/
WORKDIR /src/go
RUN go build -o /out/badapple-shim ./core/test/byobadapple

FROM alpine:3.20
COPY --from=build /out/badapple-shim /badapple-shim
ENV BADAPPLE_RPC_URL=http://host.docker.internal:9879/rpc
EXPOSE 8080
ENTRYPOINT ["/badapple-shim"]

FROM alpine:3.20
ARG TARGETARCH
COPY badapple/out/badapple-shim-${TARGETARCH} /badapple-shim
ENV BADAPPLE_RPC_URL=http://host.docker.internal:9879/rpc
EXPOSE 8080
ENTRYPOINT ["/badapple-shim"]

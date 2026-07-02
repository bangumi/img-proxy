FROM powerman/dockerize@sha256:ef4239a9d48d3f8120e7244661445f039f912201daf3d6281e5c06c874db4538 AS dockerize

FROM gcr.io/distroless/base-debian12@sha256:e7e678c88c59e70e105a46549bb3fbfb3d732ee3b4afd3a19fdab2e15afaa6b3

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy

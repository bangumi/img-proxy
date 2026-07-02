FROM powerman/dockerize@sha256:e2c34e00f1f8a8886aea2508ef4d680343f90302d3dbf4d3c08cab43351b07cf AS dockerize

FROM gcr.io/distroless/base-debian12@sha256:e7e678c88c59e70e105a46549bb3fbfb3d732ee3b4afd3a19fdab2e15afaa6b3

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy

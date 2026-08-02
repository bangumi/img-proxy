FROM powerman/dockerize@sha256:ef4239a9d48d3f8120e7244661445f039f912201daf3d6281e5c06c874db4538 AS dockerize

FROM gcr.io/distroless/base-debian13

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy

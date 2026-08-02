FROM powerman/dockerize@sha256:ef4239a9d48d3f8120e7244661445f039f912201daf3d6281e5c06c874db4538 AS dockerize

FROM gcr.io/distroless/base-debian13@sha256:f4a335ca209e1d2ee873102c17c389ad0142e3d5b21aee2817e9cc9c01d87d20

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy

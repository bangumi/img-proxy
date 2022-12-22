# need to use github ci ubuntu-20.04
FROM gcr.io/distroless/base-debian11

ENTRYPOINT ["/app/img-proxy"]

COPY /dist/img-proxy /app/img-proxy

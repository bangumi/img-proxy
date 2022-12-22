FROM gcr.io/distroless/base

ENTRYPOINT ["/app/img-proxy"]

COPY /dist/img-proxy /app/img-proxy

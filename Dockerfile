FROM gcr.io/distroless/base-debian12

ENTRYPOINT ["/app/img-proxy"]

COPY /dist/img-proxy /app/img-proxy

FROM gcr.io/distroless/static

ENTRYPOINT ["/app/img-proxy"]

COPY /dist/img-proxy /app/img-proxy

FROM powerman/dockerize@sha256:e645b37f160acfc20d49f545a8b917e402a1a10a31839912945fa78e4a35416b AS dockerize

FROM gcr.io/distroless/base-debian12@sha256:9e9b50d2048db3741f86a48d939b4e4cc775f5889b3496439343301ff54cdba8

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy

FROM powerman/dockerize@sha256:e645b37f160acfc20d49f545a8b917e402a1a10a31839912945fa78e4a35416b AS dockerize

FROM gcr.io/distroless/base-debian12@sha256:4f6e739881403e7d50f52a4e574c4e3c88266031fd555303ee2f1ba262523d6a

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy

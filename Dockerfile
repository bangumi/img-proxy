FROM powerman/dockerize@sha256:e645b37f160acfc20d49f545a8b917e402a1a10a31839912945fa78e4a35416b AS dockerize

FROM gcr.io/distroless/base-debian12@sha256:f5a3067027c2b322cd71b844f3d84ad3deada45ceb8a30f301260a602455070e

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy

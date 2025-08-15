FROM powerman/dockerize@sha256:f3ecfd5ac0f74eed3990782309ac6bf8b700f4eca0ea9e9ef507b11742c19cc6 AS dockerize

FROM gcr.io/distroless/base-debian12@sha256:4f6e739881403e7d50f52a4e574c4e3c88266031fd555303ee2f1ba262523d6a

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy

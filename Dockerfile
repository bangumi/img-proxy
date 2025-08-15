FROM powerman/dockerize@sha256:c9d039dea473ac380db66156693130ae88aaf7d349d4315c7d86bb4a38771a39 AS dockerize

FROM gcr.io/distroless/base-debian12@sha256:4f6e739881403e7d50f52a4e574c4e3c88266031fd555303ee2f1ba262523d6a

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy

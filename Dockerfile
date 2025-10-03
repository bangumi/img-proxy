FROM powerman/dockerize@sha256:c9d039dea473ac380db66156693130ae88aaf7d349d4315c7d86bb4a38771a39 AS dockerize

FROM gcr.io/distroless/base-debian12@sha256:9e9b50d2048db3741f86a48d939b4e4cc775f5889b3496439343301ff54cdba8

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy

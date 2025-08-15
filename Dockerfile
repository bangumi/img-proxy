FROM powerman/dockerize@sha256:c9d039dea473ac380db66156693130ae88aaf7d349d4315c7d86bb4a38771a39 AS dockerize

FROM gcr.io/distroless/base-debian12@sha256:201ef9125ff3f55fda8e0697eff0b3ce9078366503ef066653635a3ac3ed9c26

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy

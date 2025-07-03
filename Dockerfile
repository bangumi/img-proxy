FROM powerman/dockerize@sha256:f3ecfd5ac0f74eed3990782309ac6bf8b700f4eca0ea9e9ef507b11742c19cc6 AS dockerize

FROM gcr.io/distroless/base-debian12@sha256:201ef9125ff3f55fda8e0697eff0b3ce9078366503ef066653635a3ac3ed9c26

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy

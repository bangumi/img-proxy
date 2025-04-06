FROM powerman/dockerize@sha256:f3ecfd5ac0f74eed3990782309ac6bf8b700f4eca0ea9e9ef507b11742c19cc6 AS dockerize

FROM gcr.io/distroless/base-debian12@sha256:27769871031f67460f1545a52dfacead6d18a9f197db77110cfc649ca2a91f44

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy

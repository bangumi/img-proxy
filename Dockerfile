FROM powerman/dockerize@sha256:aea7a9d7fea00b3c7e5f000b56adb33c19e7ac0ceb22037addfdee89a3921346 AS dockerize

FROM gcr.io/distroless/base-debian12@sha256:937c7eaaf6f3f2d38a1f8c4aeff326f0c56e4593ea152e9e8f74d976dde52f56

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy

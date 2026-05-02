FROM powerman/dockerize@sha256:aea7a9d7fea00b3c7e5f000b56adb33c19e7ac0ceb22037addfdee89a3921346 AS dockerize

FROM gcr.io/distroless/base-debian12@sha256:9dce90e688a57e59ce473ff7bc4c80bc8fe52d2303b4d99b44f297310bbd2210

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy

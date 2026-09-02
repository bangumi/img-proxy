FROM powerman/dockerize@sha256:717d6a9be2bc0a8420e2a59855b5eabf6f52bbabf3bd14d101392a42229e9954 AS dockerize

FROM gcr.io/distroless/base-debian13@sha256:9ef50bca108839d5986e4d84b7f7b2d79024c9293b7c35b162c6c55485bd5868

ENTRYPOINT ["/app/img-proxy"]

COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/

COPY /dist/img-proxy /app/img-proxy

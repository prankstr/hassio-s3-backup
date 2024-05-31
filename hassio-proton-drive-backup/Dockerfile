ARG BUILD_FROM
FROM docker.io/golang:1.22-alpine AS build

WORKDIR /app
COPY . /app

RUN apk --no-cache add nodejs yarn --repository=http://dl-cdn.alpinelinux.org/alpine/edge/community && \
  cd web && \
  yarn && \
  yarn build

RUN GO111MODULE=on CGO_ENABLED=0 go build -o hassio_s3_backup cmd/hassio_s3_backup/main.go 

##
FROM $BUILD_FROM

COPY --from=build /app/hassio_s3_backup/ .
COPY run.sh /
RUN chmod a+x /run.sh

CMD [ "/run.sh" ]

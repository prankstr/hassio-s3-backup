ARG BUILD_FROM
FROM docker.io/golang:1.21-rc-alpine AS build

WORKDIR /app
COPY . /app

RUN apk --no-cache add nodejs yarn --repository=http://dl-cdn.alpinelinux.org/alpine/edge/community && \
    cd ui && \
    yarn && \
    yarn build

RUN GO111MODULE=on CGO_ENABLED=0 go build -o hassio_proton_drive_backup

##
FROM $BUILD_FROM

COPY --from=build /app/hassio_proton_drive_backup .
COPY run.sh /
RUN chmod a+x /run.sh

CMD [ "/run.sh" ]

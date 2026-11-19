FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -o /out/vinyl-api .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/vinyl-api /usr/local/bin/vinyl-api
ENV DATA_FILE=/data/records.json
VOLUME /data
EXPOSE 8080
CMD ["vinyl-api"]

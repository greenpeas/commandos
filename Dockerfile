FROM golang:1.21.4-alpine3.18 as builder
RUN apk --no-cache add tzdata git
RUN mkdir /app
WORKDIR /app
COPY go.mod go.sum ./
ARG access_token
RUN git config --global url."https://token:$access_token@gitlab.kvant.online".insteadOf "https://gitlab.kvant.online"
RUN --mount=type=cache,target=/var/cache/apt go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

FROM scratch as production
COPY --from=builder /app/main .
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
ENV TZ=Europe/Moscow
CMD ["/main"]

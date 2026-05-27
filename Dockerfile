ARG BINARY=server

FROM golang:1.25-alpine AS builder
ARG BINARY
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/app ./cmd/${BINARY}

FROM gcr.io/distroless/static-debian12
ARG BINARY
WORKDIR /app
COPY --from=builder /out/app /app/app
COPY seeds /app/seeds
ENV BINARY=${BINARY}
ENTRYPOINT ["/app/app"]

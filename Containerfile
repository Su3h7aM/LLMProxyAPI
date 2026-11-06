FROM oven/bun:1.4.4-debian AS panel

WORKDIR /panel

COPY apps/web/package.json apps/web/bun.lock ./

RUN bun install --frozen-lockfile

COPY apps/web/ ./

RUN bun run build && cp dist/index.html dist/management.html

FROM golang:1.27-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./

RUN go mod download

COPY . .

COPY --from=panel /panel/dist ./apps/web/dist

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -buildvcs=false -ldflags="-s -w -X 'main.Version=${VERSION}' -X 'main.Commit=${COMMIT}' -X 'main.BuildDate=${BUILD_DATE}'" -o ./llmproxy ./cmd/server/

FROM alpine:3

RUN apk add --no-cache ca-certificates tzdata \
 && addgroup -S llmproxy && adduser -S -G llmproxy -h /home/llmproxy -s /sbin/nologin llmproxy \
 && mkdir /LLMProxyAPI \
 && chown llmproxy:llmproxy /LLMProxyAPI

COPY --from=builder ./app/llmproxy /LLMProxyAPI/llmproxy

COPY config.example.yaml /LLMProxyAPI/config.example.yaml

WORKDIR /LLMProxyAPI

EXPOSE 8317

ENV TZ=Asia/Shanghai
ENV HOME=/home/llmproxy

RUN cp /usr/share/zoneinfo/${TZ} /etc/localtime && echo "${TZ}" > /etc/timezone

USER llmproxy

CMD ["./llmproxy"]

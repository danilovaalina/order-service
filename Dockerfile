FROM golang:1.24-alpine AS build
WORKDIR /src
COPY . .
RUN go build -o order-service cmd/main.go


FROM alpine
WORKDIR /etc/order-service
ENV PATH=/etc/order-service:${PATH}
COPY --from=build /src/order-service .
COPY config.yaml ./config.yaml

ENTRYPOINT ["order-service"]

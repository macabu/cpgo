FROM golang:1.20 AS build

WORKDIR /app

COPY . .

RUN go mod download
RUN CGO_ENABLED=0 go build -o cpgo ./cmd/cpgo/main.go


FROM gcr.io/distroless/static-debian11

ARG GITHUB_TOKEN
ENV GITHUB_TOKEN=$GITHUB_TOKEN

COPY --from=build /app/cpgo /
## Change config sample
COPY --from=build /app/config.sample.yaml /

CMD ["/cpgo", "-githubToken=${GITHUB_TOKEN}"]

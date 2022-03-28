# build stage
FROM golang:1.17 AS build-env
WORKDIR /go/src/github.com/probably-not/eks-image-updater
COPY . .

## Get Dependencies
COPY go.mod go.sum ./
RUN go mod download && go get -d -v ./...

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -tags netgo,osusergo -ldflags '-extldflags "-static"' -o eks-image-updater

# final stage
FROM gcr.io/distroless/static:latest
WORKDIR /app

COPY --from=build-env /go/src/github.com/probably-not/eks-image-updater/configs /app/configs
COPY --from=build-env /go/src/github.com/probably-not/eks-image-updater /app/

ENTRYPOINT ["./eks-image-updater"]

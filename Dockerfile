##
## BUILD
## 
FROM golang:1.17-bullseye AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download
COPY *.go ./
COPY models/*.go ./models/
COPY handlers/*.go ./handlers/
COPY host/*.go ./host/
COPY templates/*.html ./templates/

RUN go build -o /mdbssg

##
## Deploy
##
FROM golang:1.17-bullseye AS deploy

WORKDIR /app

COPY --from=build /mdbssg ./mdbssg
COPY templates/*.html ./templates/
COPY creds-to-file.sh /creds-to-file.sh

USER nonroot:nonroot

ENTRYPOINT [ "/app/mdbssg" ]

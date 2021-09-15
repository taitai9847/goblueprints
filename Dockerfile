FROM golang
# ENV ROOT=/go/src/app
WORKDIR /go/src/app
ADD . /go/src/app/
EXPOSE 8080



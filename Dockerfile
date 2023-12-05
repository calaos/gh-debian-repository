ARG ARCH
FROM ${ARCH}golang:alpine as build

COPY . $GOPATH/src/github.com/calaos/gh-debian-repository
RUN cd $GOPATH/src/github.com/calaos/gh-debian-repository && \
  go install -v ./...

ARG ARCH
FROM ${ARCH}alpine as release
COPY --from=build /go/bin/gh-debian-repository /

VOLUME ["/cache"]

ENV REPOSITORY_CACHE=/cache

ENTRYPOINT ["/gh-debian-repository"]

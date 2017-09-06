FROM alpine:3.6

ENV VERSION 0.1.0
ADD ./bundles/${VERSION}/binary/zlb-api /usr/bin

ENTRYPOINT ["/usr/bin/zlb-api"]

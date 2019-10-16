FROM registry.access.redhat.com/ubi8/ubi-minimal:8.0-213

ARG VCS_REF
ARG VCS_URL
ARG IMAGE_NAME
ARG IMAGE_DESCRIPTION
ARG SUMMARY
ARG GOARCH

RUN microdnf update && \
    microdnf install shadow-utils procps && \
    adduser -r -u 10000 watcher && \
    microdnf clean all

ADD icp-config-map-watcher_$GOARCH /usr/bin/watcher

RUN chmod a+x /usr/bin/watcher

RUN mkdir /licenses

COPY packages.yaml /licenses

user 10000

ENTRYPOINT ["/usr/bin/watcher"]

# http://label-schema.org/rc1/
LABEL org.label-schema.vendor="IBM" \
      org.label-schema.name="$IMAGE_NAME" \
      org.label-schema.description="$IMAGE_DESCRIPTION" \
      org.label-schema.vcs-ref=$VCS_REF \
      org.label-schema.vcs-url=$VCS_URL \
      org.label-schema.license="Licensed Materials - Property of IBM" \
      org.label-schema.schema-version="1.0"

LABEL name="$IMAGE_NAME"
LABEL vendor="IBM"
LABEL version="1.0"
LABEL release="$VCS_REF"
LABEL summary="$SUMMARY"
LABEL description="$IMAGE_DESCRIPTION"

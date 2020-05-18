FROM registry.access.redhat.com/ubi8/ubi-minimal:8.2-267

ARG VCS_REF
ARG VCS_URL
ARG IMAGE_NAME
ARG IMAGE_DESCRIPTION
ARG SUMMARY

RUN microdnf update && \
    microdnf install shadow-utils procps && \
    adduser -r -l -u 100101 watcher && \
    microdnf clean all

ADD configmap-watcher /usr/bin/watcher

RUN chmod a+x /usr/bin/watcher

RUN mkdir /licenses

user 100101

ENTRYPOINT ["/usr/bin/watcher"]

# http://label-schema.org/rc1/
LABEL org.label-schema.vendor="Red Hat" \
      org.label-schema.name="$IMAGE_NAME" \
      org.label-schema.description="$IMAGE_DESCRIPTION" \
      org.label-schema.vcs-ref=$VCS_REF \
      org.label-schema.vcs-url=$VCS_URL \
      org.label-schema.license="Red Hat Advanced Cluster Management for Kubernetes EULA" \
      org.label-schema.schema-version="1.0"

LABEL name="$IMAGE_NAME"
LABEL vendor="IBM"
LABEL version="1.0"
LABEL release="$VCS_REF"
LABEL summary="$SUMMARY"
LABEL description="$IMAGE_DESCRIPTION"

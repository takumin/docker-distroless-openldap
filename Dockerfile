################################################################################
# Build Args
################################################################################

ARG DEBIAN_IMAGE_DOMAIN="docker.io/library/debian"
ARG DEBIAN_IMAGE_BRANCH="10-slim"
ARG GOLANG_IMAGE_DOMAIN="docker.io/library/golang"
ARG GOLANG_IMAGE_BRANCH="buster"
ARG DISTROLESS_IMAGE_DOMAIN="gcr.io/distroless/base-debian10"
ARG DISTROLESS_IMAGE_BRANCH="latest"

################################################################################
# Build OpenLDAP stage
################################################################################

FROM ${DEBIAN_IMAGE_DOMAIN}:${DEBIAN_IMAGE_BRANCH} AS openldap

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

ARG OPENLDAP_VERSION
ENV OPENLDAP_VERSION=${OPENLDAP_VERSION:-"2.4.57"}
ARG OPENLDAP_TESTING
ENV OPENLDAP_TESTING=${OPENLDAP_TESTING:-"false"}

RUN apt-get update \
 && apt-get -y --no-install-recommends install \
      gnupg \
      wget \
      ca-certificates \
      build-essential \
      pkg-config \
      libtool \
      autoconf \
      automake \
      libssl-dev \
      groff-base \
      upx-ucl \
 && apt-get clean

WORKDIR /source
COPY openldap-signing.asc openldap-signing.asc
RUN wget "https://www.openldap.org/software/download/OpenLDAP/openldap-release/openldap-${OPENLDAP_VERSION}.tgz"
RUN wget "https://www.openldap.org/software/download/OpenLDAP/openldap-release/openldap-${OPENLDAP_VERSION}.tgz.asc"
RUN gpg --import "openldap-signing.asc"
RUN gpg --verify "openldap-${OPENLDAP_VERSION}.tgz.asc"

WORKDIR /build
RUN tar -xvf "/source/openldap-${OPENLDAP_VERSION}.tgz" --strip-components=1
RUN ./configure \
      --prefix=/usr \
      --sysconfdir=/etc \
      --localstatedir=/var \
      --enable-backends=no \
      --enable-dnssrv=yes \
      --enable-ldap=yes \
      --enable-mdb=yes \
      --enable-meta=yes \
      --enable-monitor=yes \
      --enable-null=yes \
      --enable-relay=yes \
      --enable-sock=yes \
      --enable-overlays=yes \
    | tee configure.log
RUN make -j $(nproc) depend
RUN make -j $(nproc)
RUN test "${OPENLDAP_TESTING}" = "true" && make test || true
RUN make DESTDIR=/opt/openldap install
RUN find /opt/openldap/usr/bin -type f -exec upx {} \;
RUN find /opt/openldap/usr/sbin -type f -exec upx {} \;
RUN find /opt/openldap/usr/libexec -type f -exec upx {} \;

################################################################################
# Build Entrypoint stage
################################################################################

FROM ${GOLANG_IMAGE_DOMAIN}:${GOLANG_IMAGE_BRANCH} AS entrypoint

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

RUN apt-get update \
 && apt-get -y --no-install-recommends install \
      upx-ucl \
 && apt-get clean

RUN go env -w GO111MODULE="on"

WORKDIR /go/src/github.com/takumin/docker-distroless-openldap
COPY go.* .
RUN go mod download
COPY *.go .
RUN go build -a -ldflags "-s -w" -o /usr/local/bin/entrypoint .
RUN upx /usr/local/bin/entrypoint

################################################################################
# Build Service stage
################################################################################

FROM ${DISTROLESS_IMAGE_DOMAIN}:${DISTROLESS_IMAGE_BRANCH} AS service

COPY --from=openldap /opt/openldap/etc/ /etc/
COPY --from=openldap /opt/openldap/usr/libexec/ /usr/libexec/
COPY --from=entrypoint /usr/local/bin/entrypoint /

ENTRYPOINT ["/entrypoint"]
CMD ["slapd"]

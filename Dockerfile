FROM ubuntu:16.04
MAINTAINER Bartek Kryza <bkryza@gmail.com>

# Build arguments
ARG RELEASE=devel
ARG VERSION="18.02.0-rc1"

# Get the image up to date and install utility tools
RUN apt-get -y update && \
    apt-get -y upgrade && \
    apt-get -y install bash-completion ca-certificates curl && \
    apt-get clean

WORKDIR /tmp

# Install oneclient package
RUN case ${RELEASE} in \
        production) \
            curl -O http://get.onedata.org/oneclient.sh; \
            ;; \
        *) \
            curl -O http://onedata-dev-packages.cloud.plgrid.pl/oneclient.sh; \
            ;; \
        esac && \
        sh oneclient.sh && \
        apt clean -y

RUN mkdir -p /run/docker/plugins /mnt/state /mnt/volumes /go/bin

COPY docker-volume-onedata /go/bin/docker-volume-onedata
RUN chmod 0775 /go/bin/docker-volume-onedata

CMD ["/go/bin/docker-volume-onedata"]

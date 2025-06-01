FROM ubuntu:18.04

# Install dependencies
RUN apt-get -y update && \
    apt-get -y install build-essential libncursesw5-dev bison flex liblua5.1-0-dev \
    libsqlite3-dev libz-dev pkg-config python3-yaml binutils-gold git python3-pretty-yaml \
    python3-pip python3-ipython-genutils python python-yaml golang-1.18 wget

# Create directories
RUN mkdir -p /opt/seeker/bin /crawl/trunk

# Clone the repository
RUN git clone https://github.com/crawl/crawl.git /crawl/trunk/crawl

# Define versions to build
ENV CRAWL_VERSIONS="0.24.1 0.25.1 0.26.1 0.27.1 0.28.0 0.29.1 0.30.2 0.31.2 0.32.1 0.33.0"

# Build each version of crawl
RUN for version in $CRAWL_VERSIONS; do \
        mkdir -p /crawl/$version && \
        cp -r /crawl/trunk/crawl /crawl/$version/ && \
        cd /crawl/$version/crawl/crawl-ref/source && \
        FULLDEBUG=yes make -j 24 && \
        cd util && sed -i 's/alarm/\/\/alarm/g' fake_pty.c && \
        gcc fake_pty.c -o fake_pty -lutil && \
        cd /crawl/$version/crawl/crawl-ref/source && \
        ./crawl -builddb; \
    done

# Build the latest version in trunk
RUN cd /crawl/trunk/crawl/crawl-ref/source && \
    FULLDEBUG=yes make -j 24 && \
    cd util && \
    sed -i 's/alarm/\/\/alarm/g' fake_pty.c && \
    gcc fake_pty.c -o fake_pty -lutil && \
    cd /crawl/trunk/crawl/crawl-ref/source && \
    ./crawl -builddb

# Install IPFS
RUN wget https://dist.ipfs.io/go-ipfs/v0.29.0/go-ipfs_v0.29.0_linux-amd64.tar.gz && \
    tar zxvf go-ipfs_v0.29.0_linux-amd64.tar.gz && \
    cd go-ipfs && ./install.sh

# Copy and build the seeker application
COPY seeker.go /root/
COPY go.mod /root/
RUN mkdir /root/seeker && \
    cp /root/go.mod /root/seeker && \
    cp /root/seeker.go /root/seeker && \
    cd /root/seeker && \
    /usr/lib/go-1.18/bin/go get github.com/ipfs/go-ipfs-api && \
    /usr/lib/go-1.18/bin/go build seeker.go && \
    cp seeker /opt/seeker/bin/

# Copy the launch script
COPY launch.sh /bin/
RUN chmod +x /bin/launch.sh

# Set entrypoint
ENTRYPOINT [ "/bin/launch.sh" ]

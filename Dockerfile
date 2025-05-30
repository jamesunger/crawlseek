FROM ubuntu:18.04
RUN mkdir -p /opt/frontend/bin/

RUN apt-get update
RUN apt-get -y install wget golang-1.18 
RUN wget https://dist.ipfs.io/go-ipfs/v0.29.0/go-ipfs_v0.29.0_linux-amd64.tar.gz
RUN tar zxvf go-ipfs_v0.29.0_linux-amd64.tar.gz
RUN cd go-ipfs; ./install.sh

COPY launch.sh /bin
COPY form.html /opt/frontend
COPY style.css /opt/frontend
RUN chmod +x /bin/launch.sh



# Copy and build the frontend application
COPY frontend.go /root/
COPY go.mod /root/
RUN mkdir /root/frontend && \
    cp /root/go.mod /root/frontend && \
    cp /root/frontend.go /root/frontend && \
    cd /root/frontend && \
    /usr/lib/go-1.18/bin/go get github.com/ipfs/go-ipfs-api && \
    /usr/lib/go-1.18/bin/go get github.com/google/uuid && \
    /usr/lib/go-1.18/bin/go build frontend.go && \
    cp frontend /opt/frontend/bin/


#RUN apt-get -y install nodejs npm
RUN mkdir /root/dungeon-crawl-seed-finder
COPY dungeon-crawl-seed-finder /root/dungeon-crawl-seed-finder/
RUN mkdir /opt/frontend/html && \
    cd /root/dungeon-crawl-seed-finder && \
#    npm run build && \
    cp -r build/* /opt/frontend/html/


ENTRYPOINT [ "/bin/launch.sh" ]

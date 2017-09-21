#!/bin/bash -e

# Software versions
GO_VERSION=1.9
GLIDE_VERSION=0.12.3

# Update and install stuff
apt-get update
apt-get install --no-install-recommends -y git openssh-client libxslt-dev libicu-dev unzip sqlite3

# Install docker
curl -fsSL https://get.docker.com/ | sh

# Install golang
curl -O https://storage.googleapis.com/golang/go${GO_VERSION}.linux-amd64.tar.gz
tar -xf go${GO_VERSION}.linux-amd64.tar.gz
mv go /usr/local
rm go${GO_VERSION}.linux-amd64.tar.gz
ln -s /usr/local/go/bin/go /usr/local/bin/go

# Install glide
curl -L https://github.com/Masterminds/glide/releases/download/v${GLIDE_VERSION}/glide-v${GLIDE_VERSION}-linux-amd64.tar.gz -o /tmp/glide-v${GLIDE_VERSION}-linux-amd64.tar.gz
tar -xf /tmp/glide-v${GLIDE_VERSION}-linux-amd64.tar.gz -C /tmp
cp /tmp/linux-amd64/glide /usr/local/bin
rm -rf /tmp/linux-amd64

# Set GOPATH to global environments
echo 'GOPATH="/go"' >> /etc/environment
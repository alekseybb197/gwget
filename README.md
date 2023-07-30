# gwget
A simple replacement for curl

## Linux build

```bash
apt-get update
apt-get install curl git make build-essential pkg-config libcurl4-openssl-dev -y
curl -LO https://go.dev/dl/go1.20.6.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.20.6.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
git clone https://github.com/alekseybb197/gwget.git
cd gwget
make build
```
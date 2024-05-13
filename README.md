# ServFailD

A little DNS server that will reject all queries with SERVFAIL

## Usage

```sh
docker run -d -p 53:5353/udp -p 53:5353/tcp --name servfaild ghcr.io/noisysockets/servfaild:latest
```
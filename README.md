### Source
- https://github.com/chenhw2/dnspod-http-dns
  
### Thanks to
- https://www.dnspod.cn/httpdns/guide
  
### Docker
- https://hub.docker.com/r/chenhw2/dnspod-http-dns
  
### TODO
- No caching is implemented, and probably never will
  
### Usage
```
$ docker pull chenhw2/dnspod-http-dns

$ docker run -d \
    -e "Args=-T -U --fallbackedns 119.29.29.29" \
    -p "5300:5300/udp" \
    -p "5300:5300/tcp" \
    chenhw2/dnspod-http-dns

```
### Help
```
$ docker run --rm chenhw2/dnspod-http-dns -h
NAME:
   dnspod-http-dns - A DNS-protocol proxy for DNSPOD's DNS-over-HTTP service.

USAGE:
   dnspod-http-dns [global options] command [command options] [arguments...]

VERSION:
   MISSING build version [git hash]

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --listen value, -l value        Serve address (default: ":5300")
   --fallbackedns value, -e value  Extension mechanisms for DNS (EDNS) is parameters of the Domain Name System (DNS) protocol.
   --udp, -U                       Listen on UDP
   --tcp, -T                       Listen on TCP
   --help, -h                      show help
   --version, -v                   print the version

```

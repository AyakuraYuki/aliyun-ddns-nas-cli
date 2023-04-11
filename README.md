# aliyun-ddns-nas-cli

> 致敬：原 repo [honwen/aliyun-ddns-cli](https://github.com/honwen/aliyun-ddns-cli)
> 
> 本项目改编自 `honwen/aliyun-ddns-cli` 项目，升级了阿里云 DNS SDK，并针对我本人的需求改变了部分代码和 API

本项目是一个提供给群晖 NAS 配置阿里云 DNS 服务实现 DDNS 功能的工具，目的是运行在 Docker 中，定时、自动的向阿里云注册 DNS 解析。

## Docker usage

```shell
$ docker pull ayakurayuki/aliyun-ddns-nas-cli

$ docker run -d \
    -e "AccessKeyId=123456" \
    -e "AccessKeySecret=123456" \
    -e "DOMAIN=ddns.example.com" \
    -e "REDO=600" \
    ayakurayuki/aliyun-ddns-nas-cli
```

## Environments

* `AccessKeyId`: 阿里云 Access Key ID
* `AccessKeySecret`: 阿里云 Access Key Secret
* `DOMAIN`: 自动刷新的解析地址
* `REDO`: 自动刷新的间隔时间，单位：秒。支持在特定时间上下增加随机间隔，在秒数后面接上字母 `R` 即可。`0` 表示不自动刷新，行为类似于命令 `update`。
* `IPAPI`: 自定的获取 IP 地址的 API Url

## Help

```shell
$ docker run --rm ayakurayuki/aliyun-ddns-nas-cli -h
NAME:
   ali-ddns - aliyun-ddns-nas-cli

USAGE:
   ali-ddns [global options] command [command options] [arguments...]

VERSION:
   Git:[MISSING VERSION [GIT HASH]] (go1.20.3)

AUTHOR:
   Ayakura Yuki

COMMANDS:
   help, h  Shows a list of commands or help for one command
   DDNS:
     list, l                     Describe all domain records by giving domain name
     update, up, u               Update domain record, or create record if not exists
     delete                      Delete domain record
     auto-update, auto-up, auto  Update domain record automatically
   Maintain:
     get-ip   Get IP address combines multiple Web-API
     resolve  Resolve DNS-IPv4 combines multiple DNS upstreams

GLOBAL OPTIONS:
   --access-key-id value, --id value                              Aliyun OpenAPI Access Key ID
   --access-key-secret value, --secret value                      Aliyun OpenAPI Access Key Secret
   --ip-api value, --api value                                    Specify API to fetch ip address, e.g. https://v6r.ipip.net/
   --ip2location-api-key value, --ip2loc-key value, --ip2l value  Specify API key for using IP2Location API to get IP GeoLocation
   --ipv6, -6                                                     IPv6 (default: false)
   --help, -h                                                     show help
   --version, -v                                                  print the version

```

## CLI Examples

```shell
# 自动刷新
aliddns --id ${AccessKeyID} --secret ${AccessKeySecret} auto-update --domain ddns.example.win

# 手动刷新
aliddns --id ${AccessKeyID} --secret ${AccessKeySecret} update --domain ddns.example.win --ipaddr $(ifconfig pppoe-wan | sed -n '2{s/[^0-9]*://;s/[^0-9.].*//p}')

```

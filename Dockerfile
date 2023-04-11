FROM ayakurayuki/yukibuntu:20.04-slim
MAINTAINER AyakuraYuki

ENV AccessKeyId=123456
ENV AccessKeySecret=123456
ENV DOMAIN=ddns.example.com
ENV REDO=600R
ENV IPAPI=[IPAPI-GROUP]

RUN apt update \
    && apt install -y curl \
    && mkdir -p /usr/bin/ \
    && cd /usr/bin/ \
    && curl -skSL $(curl -skSL 'https://api.github.com/repos/AyakuraYuki/aliyun-ddns-nas-cli/releases/latest' | sed -n 's/.*\(https:.*.tar.gz\).*/\1/p' | grep 'linux-amd64') | tar --strip-components=1 -zx linux-amd64/aliddns \
    && ln -sf aliddns aliyun-ddns-nas-cli \
    && aliyun-ddns-nas-cli -v

CMD aliyun-ddns-nas-cli --ip-api ${IPAPI} auto-update --domain ${DOMAIN} --redo ${REDO}

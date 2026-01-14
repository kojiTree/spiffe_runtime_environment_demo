FROM busybox:1.36.1-uclibc AS bb

FROM ghcr.io/spiffe/spire-server:1.9.5
# busybox 本体を置く
COPY --from=bb /bin/busybox /bin/busybox

# 必要コマンドを busybox に向ける（bootstrap.sh が使う分）
# sh
COPY --from=bb /bin/busybox /bin/sh
# coreutils 相当
COPY --from=bb /bin/busybox /usr/bin/awk
COPY --from=bb /bin/busybox /usr/bin/tail
COPY --from=bb /bin/busybox /bin/sleep

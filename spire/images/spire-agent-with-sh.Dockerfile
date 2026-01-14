FROM busybox:1.36.1-uclibc AS bb
FROM ghcr.io/spiffe/spire-agent:1.9.5

COPY --from=bb /bin/busybox /bin/busybox
COPY --from=bb /bin/busybox /bin/sh

COPY spire/scripts/entrypoint-agent.sh /entrypoint-agent.sh

ENTRYPOINT ["/bin/sh", "/entrypoint-agent.sh"]

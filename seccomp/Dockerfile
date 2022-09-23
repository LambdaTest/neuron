FROM alpine:latest

WORKDIR /home/profiles

COPY profiles/* /home/profiles/
COPY scripts/move_seccomp.sh /home/profiles/move_seccomp.sh
RUN chmod +x ./move_seccomp.sh

ENTRYPOINT ["./move_seccomp.sh"]
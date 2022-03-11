FROM alpine:3.11

COPY ./_build/drbd-installer /

COPY ./kernel-mods /kernel-mods

ENTRYPOINT [ "/drbd-installer"]

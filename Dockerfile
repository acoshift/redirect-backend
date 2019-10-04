FROM scratch

ADD redirect-backend /server

ENTRYPOINT ["/server"]

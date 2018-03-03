FROM scratch

ADD redirect-backend /server
EXPOSE 8080

ENTRYPOINT ["/server"]

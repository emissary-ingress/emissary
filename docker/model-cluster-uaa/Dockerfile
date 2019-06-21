FROM cfidentity/uaa-postgresql:74.0.0
RUN git clone -b 4.31.0 git://github.com/cloudfoundry/uaa.git /root/uaa
WORKDIR /root/uaa
RUN ./gradlew --no-daemon assemble
CMD ["./gradlew", "--no-daemon", "--exclude-task=assemble", "run"]
EXPOSE 8080

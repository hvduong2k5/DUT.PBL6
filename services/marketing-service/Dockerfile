FROM eclipse-temurin:21-jdk AS build
WORKDIR /src
COPY . .
RUN sh gradlew --no-daemon bootJar

FROM eclipse-temurin:21-jre
WORKDIR /app
COPY --from=build /src/build/libs/marketing-service.jar app.jar
USER 10001:10001
EXPOSE 8014
ENTRYPOINT ["java", "-jar", "/app/app.jar"]

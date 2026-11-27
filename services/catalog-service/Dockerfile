# Development / Local Dockerfile for catalog-service (runs prebuilt executable JAR)
FROM eclipse-temurin:21-jre-alpine

LABEL maintainer="Hoang Van Duong <duong@omama.vn>"
LABEL description="DUT.PBL6 Mè Xửng O Mạ - Catalog Service"

WORKDIR /app

# Add a non-privileged user to run the app
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy prebuilt executable JAR
COPY build/libs/catalog-service-*.jar app.jar

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8006 9005

ENV JAVA_OPTS="-XX:+UseG1GC -XX:MaxRAMPercentage=75.0"

ENTRYPOINT ["sh", "-c", "java $JAVA_OPTS -jar app.jar"]

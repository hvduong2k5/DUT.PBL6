# Production Dockerfile for identity-service
FROM eclipse-temurin:21-jre-alpine

LABEL maintainer="Hoang Van Duong <duong@omama.vn>"
LABEL description="DUT.PBL6 Mè Xửng O Mạ - Identity Service"

WORKDIR /app

# Add a non-privileged user to run the app
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Copy prebuilt executable JAR
COPY build/libs/identity-service-*.jar app.jar

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8016 9016

ENV JAVA_OPTS="-XX:+UseG1GC -XX:MaxRAMPercentage=75.0"

ENTRYPOINT ["sh", "-c", "java $JAVA_OPTS -jar app.jar"]


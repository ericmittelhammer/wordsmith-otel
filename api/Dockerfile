# Build stage
FROM --platform=${BUILDPLATFORM} maven:3-amazoncorretto-17 as build
WORKDIR /home/lab

COPY pom.xml .
RUN mvn verify -DskipTests --fail-never

COPY src ./src
RUN mvn verify

WORKDIR /home/lab/nr
ADD https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/v1.26.0/opentelemetry-javaagent.jar opentelemetry-javaagent.jar

# Run stage
FROM --platform=${TARGETPLATFORM} amazoncorretto:17
WORKDIR /app
COPY --from=build /home/lab/target .
COPY --from=build /home/lab/nr/opentelemetry-javaagent.jar opentelemetry-javaagent.jar
ENTRYPOINT ["java", "-jar", "/app/words.jar"]
EXPOSE 8080

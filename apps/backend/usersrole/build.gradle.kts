import com.google.cloud.tools.jib.api.buildplan.ImageFormat
import java.time.Instant
import org.springframework.boot.gradle.tasks.bundling.BootBuildImage

plugins {
	java
	id("org.springframework.boot") version "4.1.0"
	id("io.spring.dependency-management") version "1.1.5"
	id("maven-publish")
	id("com.google.cloud.tools.jib") version "3.4.5"
	id("dev.nx.gradle.project-graph") version "0.1.20"
}

group = "com.jdw"
version = file("VERSION").readText().trim()

java {
	sourceCompatibility = JavaVersion.VERSION_21
	targetCompatibility = JavaVersion.VERSION_21
}

configurations {
	compileOnly {
		extendsFrom(configurations.annotationProcessor.get())
	}
}

repositories {
	mavenCentral()
}

dependencies {
	implementation("org.springframework.boot:spring-boot-starter-actuator")
	implementation("org.springframework.boot:spring-boot-starter-jdbc")
	implementation("org.springframework.boot:spring-boot-starter-security")
	implementation("org.springframework.boot:spring-boot-starter-web")
	implementation("org.springframework.boot:spring-boot-starter-validation")
	implementation("org.springframework.boot:spring-boot-starter-aop")
	implementation("io.jsonwebtoken:jjwt-api:0.12.5")
	implementation("io.jsonwebtoken:jjwt-impl:0.12.5")
	implementation("io.jsonwebtoken:jjwt-jackson:0.12.5")
	implementation("org.springdoc:springdoc-openapi-starter-webmvc-ui:2.6.0")
	compileOnly("org.projectlombok:lombok")
	runtimeOnly("org.postgresql:postgresql")
	annotationProcessor("org.projectlombok:lombok")
	testImplementation("org.springframework.boot:spring-boot-starter-test")
	testImplementation("org.springframework.boot:spring-boot-testcontainers")
	testImplementation("org.springframework.security:spring-security-test")
	testImplementation("org.testcontainers:junit-jupiter")
	testImplementation("org.testcontainers:postgresql")
}

tasks.withType<Test> {
	useJUnitPlatform()
}

springBoot {
	buildInfo()
}

publishing {
	publications {
		create<MavenPublication>("mavenJava") {
			artifact(tasks.getByName("bootJar"))
		}
	}
}

tasks.named<BootBuildImage>("bootBuildImage") {
	imageName.set("docker.io/jdwlabs/${project.name}:${version}")
	publish.set(true)
	tags.set(listOf("docker.io/jdwlabs/${project.name}:latest"))
	docker {
		publishRegistry {
			username.set(System.getenv("DOCKERHUB_USERNAME"))
			password.set(System.getenv("DOCKERHUB_PASSWORD"))
		}
	}
}

// Resolved at configuration time for the OCI image.revision label. A missing git
// context (e.g. a source-tarball build) is non-fatal — fall back rather than fail.
val gitRevision: String = runCatching {
	ProcessBuilder("git", "rev-parse", "--short", "HEAD")
		.redirectErrorStream(true)
		.start()
		.inputStream.bufferedReader().readText().trim()
}.getOrNull()?.takeIf { it.isNotEmpty() } ?: "unknown"

val imageCreated: String = Instant.now().toString()

jib {
	to {
		image = "docker.io/jdwlabs/${project.name}:${version}"
		tags = setOf("latest")
		auth {
			username = System.getenv("DOCKERHUB_USERNAME")
			password = System.getenv("DOCKERHUB_PASSWORD")
		}
	}
	from {
		image = "eclipse-temurin:21@sha256:6634936b2e8d90ee16eeb94420d71cd5e36ca677a4cf795a9ee1ee6e94379988"
		auth {
			username = System.getenv("DOCKERHUB_USERNAME")
			password = System.getenv("DOCKERHUB_PASSWORD")
		}
	}
	container {
		format = ImageFormat.OCI
		ports = listOf("8080")
		// Mirrors scripts/build-image.sh so the Jib-built image carries the same
		// org.opencontainers.image.* set as the six buildx images.
		labels.set(
			mapOf(
				"org.opencontainers.image.title" to "usersrole",
				"org.opencontainers.image.description" to "Users & roles Spring Boot backend service for the jdwlabs platform",
				"org.opencontainers.image.vendor" to "jdwlabs",
				"org.opencontainers.image.licenses" to "PolyForm-Noncommercial-1.0.0",
				"org.opencontainers.image.url" to "https://github.com/jdwlabs/apps",
				"org.opencontainers.image.source" to "https://github.com/jdwlabs/apps",
				"org.opencontainers.image.documentation" to "https://hub.docker.com/r/jdwlabs/usersrole",
				"org.opencontainers.image.version" to version.toString(),
				"org.opencontainers.image.revision" to gitRevision,
				"org.opencontainers.image.created" to imageCreated,
			),
		)
	}
}

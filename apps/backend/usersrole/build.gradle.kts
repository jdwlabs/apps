import com.google.cloud.tools.jib.api.buildplan.ImageFormat
import java.time.Instant
import org.springframework.boot.gradle.tasks.bundling.BootBuildImage

plugins {
	java
	id("org.springframework.boot") version "4.1.0"
	id("io.spring.dependency-management") version "1.1.7"
	id("maven-publish")
	id("com.google.cloud.tools.jib") version "3.4.5"
	id("dev.nx.gradle.project-graph") version "0.1.24"
}

group = "com.jdw"
// The released version comes from the git tag, which only exists after the
// commit being built — so no file in the tree can carry it. The image build
// passes it in; anything else is a local build and says so in the tag.
//
// A property that is present but empty is a failed version lookup upstream, not
// a request for the development default: the caller substitutes the version into
// -PreleaseVersion, and a substitution that fails still produces a well-formed
// flag. Falling back would publish the placeholder tag to Docker Hub, so refuse
// instead. An absent property means no caller tried, which is a local build.
val releaseVersionProperty = findProperty("releaseVersion") as String?
require(releaseVersionProperty == null || releaseVersionProperty.isNotBlank()) {
	"-PreleaseVersion was passed but is empty — the version lookup that supplies it failed"
}
version =
	releaseVersionProperty
		?: System.getenv("RELEASE_VERSION")?.ifBlank { null }
		?: "0.0.0-dev"

java {
	// A toolchain rather than source/targetCompatibility: those two only assert
	// the language level, leaving Gradle to compile with whatever JVM happened to
	// launch it. Two machines on different JDKs then produce different bytecode
	// from identical sources, which a build cache keyed on sources alone will
	// happily treat as interchangeable.
	toolchain {
		languageVersion = JavaLanguageVersion.of(21)
	}
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
	implementation("org.springframework.boot:spring-boot-starter-aspectj")
	implementation("io.jsonwebtoken:jjwt-api:0.12.5")
	implementation("io.jsonwebtoken:jjwt-impl:0.12.5")
	implementation("io.jsonwebtoken:jjwt-jackson:0.12.5")
	implementation("org.springdoc:springdoc-openapi-starter-webmvc-ui:3.0.3")
	compileOnly("org.projectlombok:lombok")
	runtimeOnly("org.postgresql:postgresql")
	annotationProcessor("org.projectlombok:lombok")
	testImplementation("org.springframework.boot:spring-boot-starter-test")
	testImplementation("org.springframework.boot:spring-boot-testcontainers")
	testImplementation("org.springframework.security:spring-security-test")
	testImplementation("org.testcontainers:testcontainers-junit-jupiter")
	testImplementation("org.testcontainers:testcontainers-postgresql")
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

allprojects {
    apply {
        plugin("dev.nx.gradle.project-graph")
    }
}
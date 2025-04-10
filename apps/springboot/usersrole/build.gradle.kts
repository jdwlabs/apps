import com.google.cloud.tools.jib.api.buildplan.ImageFormat
import org.springframework.boot.gradle.tasks.bundling.BootBuildImage

plugins {
	java
	id("org.springframework.boot") version "3.3.2"
	id("io.spring.dependency-management") version "1.1.5"
	id("maven-publish")
	id("com.google.cloud.tools.jib") version "3.4.5"
}

group = "com.jdw"
version = "0.10.7"

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
	imageName.set("docker.io/jdwillmsen/jdw-${project.name}:${version}")
	publish.set(true)
	tags.set(listOf("docker.io/jdwillmsen/jdw-${project.name}:latest"))
	docker {
		publishRegistry {
			username.set(System.getenv("DOCKERHUB_USERNAME"))
			password.set(System.getenv("DOCKERHUB_PASSWORD"))
		}
	}
}

jib {
	to {
		image = "docker.io/jdwillmsen/jdw-${project.name}:${version}"
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
	}
}

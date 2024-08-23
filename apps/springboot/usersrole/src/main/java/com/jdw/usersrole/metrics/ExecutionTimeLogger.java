package com.jdw.usersrole.metrics;

import lombok.AllArgsConstructor;
import lombok.Getter;

import java.lang.annotation.ElementType;
import java.lang.annotation.Retention;
import java.lang.annotation.RetentionPolicy;
import java.lang.annotation.Target;

@Target(ElementType.METHOD)
@Retention(RetentionPolicy.RUNTIME)
public @interface ExecutionTimeLogger {
    double maxThresholdTime() default 0.0d;

    ExecutionTimeUnit executionTimeUnit() default ExecutionTimeUnit.MSEC;

    @Getter
    @AllArgsConstructor
    enum ExecutionTimeUnit {
        NSEC("nanoseconds", "nanos", "ns", 1L),
        USEC("microseconds", "micros", "µs", 1_000L),
        MSEC("milliseconds", "millis", "ms", 1_000_000L),
        SEC("seconds", "secs", "s", 1_000_000_000L),
        MIN("minutes", "mins","m", 60_000_000_000L),
        HOUR("hours", "hrs", "h", 3_600_000_000_000L),
        DAY("days", "days", "d", 86_400_000_000_000L),
        WEEK("weeks", "weeks", "w", 604_800_000_000_000L);
        private final String longName;
        private final String shortName;
        private final String abbreviation;
        private final long nanos;
    }
}

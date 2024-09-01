package com.jdw.usersrole.metrics;

import lombok.extern.slf4j.Slf4j;
import org.aspectj.lang.ProceedingJoinPoint;
import org.aspectj.lang.annotation.Around;
import org.aspectj.lang.annotation.Aspect;
import org.springframework.stereotype.Component;

import java.text.DecimalFormat;

@Aspect
@Component
@Slf4j
public class ExecutionTimeLoggerAspect {
    @Around("@annotation(executionTimeLogger)")
    public Object executionTimeLogger(ProceedingJoinPoint joinPoint, ExecutionTimeLogger executionTimeLogger) throws Throwable {
        long startTime = System.nanoTime();
        try {
            return joinPoint.proceed();
        } catch (Throwable t) {
            log.warn("Error occurred while executing {}. Error={}", joinPoint.getSignature(), t.toString());
            throw t;
        } finally {
            logExecutionTime(joinPoint, startTime, executionTimeLogger);
        }
    }

    private void logExecutionTime(ProceedingJoinPoint joinPoint, long startTime, ExecutionTimeLogger executionTimeLogger) {
        try {
            long executionTimeInNanos = System.nanoTime() - startTime;
            ExecutionTimeLogger.ExecutionTimeUnit unit = executionTimeLogger.executionTimeUnit();
            double executionTime = executionTimeInNanos / (double) unit.getNanos();
            double maxThresholdTime = executionTimeLogger.maxThresholdTime();
            String methodName = joinPoint.getSignature().getName();
            String className = joinPoint.getSignature()
                    .getDeclaringTypeName()
                    .substring(joinPoint.getSignature().getDeclaringTypeName().lastIndexOf(".") + 1)
                    .trim();
            DecimalFormat decimalFormat = new DecimalFormat(executionTimeLogger.decimalFormat());

            String timeUnit = switch (executionTimeLogger.timeUnitFormat()) {
                case LONG_NAME -> unit.getLongName();
                case SHORT_NAME -> unit.getShortName();
                case ABBREVIATION -> unit.getAbbreviation();
            };

            if (maxThresholdTime > 0.0d && executionTime > maxThresholdTime) {
                log.warn("{}: {} was executed in {} {} which was higher than expected max threshold time of {} {}",
                        className, methodName, decimalFormat.format(executionTime), timeUnit, maxThresholdTime, timeUnit);
            } else {
                log.info("{}: {} was executed in {} {}", className, methodName, decimalFormat.format(executionTime), timeUnit);
            }
        } catch (Throwable t) {
            log.warn("Error occurred while logging execution time. Error={}", t.toString());
        }
    }
}

package com.jdw.usersrole.metrics;

import org.aspectj.lang.ProceedingJoinPoint;
import org.aspectj.lang.reflect.MethodSignature;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.boot.test.system.CapturedOutput;
import org.springframework.boot.test.system.OutputCaptureExtension;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
@Tag("unit")
@Tag("fast")
class ExecutionTimeLoggerAspectTests {
    @Mock
    private ProceedingJoinPoint joinPoint;
    @Mock
    private ExecutionTimeLogger executionTimeLogger;
    @InjectMocks
    private ExecutionTimeLoggerAspect aspect;

    @Test
    @ExtendWith(OutputCaptureExtension.class)
    void whenProceedExecutesSuccessfully_thenExecutionTimeIsLogged(CapturedOutput capturedOutput) throws Throwable {
        when(joinPoint.proceed()).thenReturn(new Object());
        when(executionTimeLogger.executionTimeUnit()).thenReturn(ExecutionTimeLogger.ExecutionTimeUnit.MSEC);
        when(executionTimeLogger.maxThresholdTime()).thenReturn(0.0d);
        MethodSignature signature = mock(MethodSignature.class);
        when(joinPoint.getSignature()).thenReturn(signature);
        when(signature.getName()).thenReturn("testMethod");
        when(signature.getDeclaringTypeName()).thenReturn("package.test.class");

        aspect.executionTimeLogger(joinPoint, executionTimeLogger);

        verify(joinPoint, times(1)).proceed();
        assertTrue(capturedOutput.toString().contains("class: testMethod was executed in"));
    }

    @Test
    @ExtendWith(OutputCaptureExtension.class)
    void whenProceedThrowsException_thenExecutionIsLogged(CapturedOutput capturedOutput) throws Throwable {
        when(joinPoint.proceed()).thenThrow(new RuntimeException("Error occurred"));
        MethodSignature signature = mock(MethodSignature.class);
        when(executionTimeLogger.executionTimeUnit()).thenReturn(ExecutionTimeLogger.ExecutionTimeUnit.MSEC);
        when(executionTimeLogger.maxThresholdTime()).thenReturn(0.0d);
        when(joinPoint.getSignature()).thenReturn(signature);
        when(signature.getName()).thenReturn("testMethod");
        when(signature.getDeclaringTypeName()).thenReturn("package.test.class");

        assertThrows(RuntimeException.class, () -> aspect.executionTimeLogger(joinPoint, executionTimeLogger));
        assertTrue(capturedOutput.toString().contains("class: testMethod was executed in"));
    }

    @Test
    @ExtendWith(OutputCaptureExtension.class)
    void whenExecutionTimeExceedsMaxThreshold_thenWarningIsLogged(CapturedOutput capturedOutput) throws Throwable {
        when(joinPoint.proceed()).thenAnswer(invocationOnMock -> {
            Thread.sleep(100);
            return new Object();
        });
        when(executionTimeLogger.executionTimeUnit()).thenReturn(ExecutionTimeLogger.ExecutionTimeUnit.NSEC);
        when(executionTimeLogger.maxThresholdTime()).thenReturn(0.1d);
        MethodSignature signature = mock(MethodSignature.class);
        when(joinPoint.getSignature()).thenReturn(signature);
        when(signature.getName()).thenReturn("testMethod");
        when(signature.getDeclaringTypeName()).thenReturn("package.test.class");

        aspect.executionTimeLogger(joinPoint, executionTimeLogger);

        verify(joinPoint, times(1)).proceed();
        assertTrue(capturedOutput.toString().contains("which was higher than expected max threshold time of"));
    }

    @Test
    @ExtendWith(OutputCaptureExtension.class)
    void whenExecutionTimeDoesNotExceedMaxThreshold_thenInfoIsLogged(CapturedOutput capturedOutput) throws Throwable {
        when(joinPoint.proceed()).thenReturn(new Object());
        when(executionTimeLogger.executionTimeUnit()).thenReturn(ExecutionTimeLogger.ExecutionTimeUnit.WEEK);
        when(executionTimeLogger.maxThresholdTime()).thenReturn(Double.MAX_VALUE);
        MethodSignature signature = mock(MethodSignature.class);
        when(joinPoint.getSignature()).thenReturn(signature);
        when(signature.getName()).thenReturn("testMethod");
        when(signature.getDeclaringTypeName()).thenReturn("package.test.class");

        aspect.executionTimeLogger(joinPoint, executionTimeLogger);

        verify(joinPoint, times(1)).proceed();
        assertTrue(capturedOutput.toString().contains("class: testMethod was executed in"));
    }

    @Test
    @ExtendWith(OutputCaptureExtension.class)
    void whenExecutionTimeUnitIsNSEC_thenExecutionTimeIsLoggedInNSec(CapturedOutput capturedOutput) throws Throwable {
        when(joinPoint.proceed()).thenReturn(new Object());
        when(executionTimeLogger.executionTimeUnit()).thenReturn(ExecutionTimeLogger.ExecutionTimeUnit.NSEC);
        when(executionTimeLogger.maxThresholdTime()).thenReturn(0.0d);
        MethodSignature signature = mock(MethodSignature.class);
        when(joinPoint.getSignature()).thenReturn(signature);
        when(signature.getName()).thenReturn("testMethod");
        when(signature.getDeclaringTypeName()).thenReturn("package.test.class");

        aspect.executionTimeLogger(joinPoint, executionTimeLogger);

        verify(joinPoint, times(1)).proceed();
        assertTrue(capturedOutput.toString().contains(ExecutionTimeLogger.ExecutionTimeUnit.NSEC.getLongName()));
    }

    @Test
    @ExtendWith(OutputCaptureExtension.class)
    void whenExecutionTimeUnitIsUSEC_thenExecutionTimeIsLoggedInUSec(CapturedOutput capturedOutput) throws Throwable {
        when(joinPoint.proceed()).thenReturn(new Object());
        when(executionTimeLogger.executionTimeUnit()).thenReturn(ExecutionTimeLogger.ExecutionTimeUnit.USEC);
        when(executionTimeLogger.maxThresholdTime()).thenReturn(0.0d);
        MethodSignature signature = mock(MethodSignature.class);
        when(joinPoint.getSignature()).thenReturn(signature);
        when(signature.getName()).thenReturn("testMethod");
        when(signature.getDeclaringTypeName()).thenReturn("package.test.class");

        aspect.executionTimeLogger(joinPoint, executionTimeLogger);

        verify(joinPoint, times(1)).proceed();
        assertTrue(capturedOutput.toString().contains(ExecutionTimeLogger.ExecutionTimeUnit.USEC.getLongName()));
    }

    @Test
    @ExtendWith(OutputCaptureExtension.class)
    void whenExecutionTimeUnitIsMSEC_thenExecutionTimeIsLoggedInMSec(CapturedOutput capturedOutput) throws Throwable {
        when(joinPoint.proceed()).thenReturn(new Object());
        when(executionTimeLogger.executionTimeUnit()).thenReturn(ExecutionTimeLogger.ExecutionTimeUnit.MSEC);
        when(executionTimeLogger.maxThresholdTime()).thenReturn(0.0d);
        MethodSignature signature = mock(MethodSignature.class);
        when(joinPoint.getSignature()).thenReturn(signature);
        when(signature.getName()).thenReturn("testMethod");
        when(signature.getDeclaringTypeName()).thenReturn("package.test.class");

        aspect.executionTimeLogger(joinPoint, executionTimeLogger);

        verify(joinPoint, times(1)).proceed();
        assertTrue(capturedOutput.toString().contains(ExecutionTimeLogger.ExecutionTimeUnit.MSEC.getLongName()));
    }

    @Test
    @ExtendWith(OutputCaptureExtension.class)
    void whenExecutionTimeUnitIsSEC_thenExecutionTimeIsLoggedInSec(CapturedOutput capturedOutput) throws Throwable {
        when(joinPoint.proceed()).thenReturn(new Object());
        when(executionTimeLogger.executionTimeUnit()).thenReturn(ExecutionTimeLogger.ExecutionTimeUnit.SEC);
        when(executionTimeLogger.maxThresholdTime()).thenReturn(0.0d);
        MethodSignature signature = mock(MethodSignature.class);
        when(joinPoint.getSignature()).thenReturn(signature);
        when(signature.getName()).thenReturn("testMethod");
        when(signature.getDeclaringTypeName()).thenReturn("package.test.class");

        aspect.executionTimeLogger(joinPoint, executionTimeLogger);

        verify(joinPoint, times(1)).proceed();
        assertTrue(capturedOutput.toString().contains(ExecutionTimeLogger.ExecutionTimeUnit.SEC.getLongName()));
    }

    @Test
    @ExtendWith(OutputCaptureExtension.class)
    void whenExecutionTimeUnitIsMIN_thenExecutionTimeIsLoggedInMin(CapturedOutput capturedOutput) throws Throwable {
        when(joinPoint.proceed()).thenReturn(new Object());
        when(executionTimeLogger.executionTimeUnit()).thenReturn(ExecutionTimeLogger.ExecutionTimeUnit.MIN);
        when(executionTimeLogger.maxThresholdTime()).thenReturn(0.0d);
        MethodSignature signature = mock(MethodSignature.class);
        when(joinPoint.getSignature()).thenReturn(signature);
        when(signature.getName()).thenReturn("testMethod");
        when(signature.getDeclaringTypeName()).thenReturn("package.test.class");

        aspect.executionTimeLogger(joinPoint, executionTimeLogger);

        verify(joinPoint, times(1)).proceed();
        assertTrue(capturedOutput.toString().contains(ExecutionTimeLogger.ExecutionTimeUnit.MIN.getLongName()));
    }

    @Test
    @ExtendWith(OutputCaptureExtension.class)
    void whenExecutionTimeUnitIsDAY_thenExecutionTimeIsLoggedInDay(CapturedOutput capturedOutput) throws Throwable {
        when(joinPoint.proceed()).thenReturn(new Object());
        when(executionTimeLogger.executionTimeUnit()).thenReturn(ExecutionTimeLogger.ExecutionTimeUnit.DAY);
        when(executionTimeLogger.maxThresholdTime()).thenReturn(0.0d);
        MethodSignature signature = mock(MethodSignature.class);
        when(joinPoint.getSignature()).thenReturn(signature);
        when(signature.getName()).thenReturn("testMethod");
        when(signature.getDeclaringTypeName()).thenReturn("package.test.class");

        aspect.executionTimeLogger(joinPoint, executionTimeLogger);

        verify(joinPoint, times(1)).proceed();
        assertTrue(capturedOutput.toString().contains(ExecutionTimeLogger.ExecutionTimeUnit.DAY.getLongName()));
    }

    @Test
    @ExtendWith(OutputCaptureExtension.class)
    void whenExecutionTimeUnitIsWEEK_thenExecutionTimeIsLoggedInWeek(CapturedOutput capturedOutput) throws Throwable {
        when(joinPoint.proceed()).thenReturn(new Object());
        when(executionTimeLogger.executionTimeUnit()).thenReturn(ExecutionTimeLogger.ExecutionTimeUnit.WEEK);
        when(executionTimeLogger.maxThresholdTime()).thenReturn(0.0d);
        MethodSignature signature = mock(MethodSignature.class);
        when(joinPoint.getSignature()).thenReturn(signature);
        when(signature.getName()).thenReturn("testMethod");
        when(signature.getDeclaringTypeName()).thenReturn("package.test.class");

        aspect.executionTimeLogger(joinPoint, executionTimeLogger);

        verify(joinPoint, times(1)).proceed();
        assertTrue(capturedOutput.toString().contains(ExecutionTimeLogger.ExecutionTimeUnit.WEEK.getLongName()));
    }

    @Test
    @ExtendWith(OutputCaptureExtension.class)
    void whenLogExecutionTimeThrowsException_thenExceptionIsLogged(CapturedOutput capturedOutput) throws Throwable {
        when(joinPoint.proceed()).thenReturn(new Object());
        when(executionTimeLogger.executionTimeUnit()).thenReturn(ExecutionTimeLogger.ExecutionTimeUnit.HOUR);
        when(executionTimeLogger.maxThresholdTime()).thenReturn(0.0d);
        when(joinPoint.getSignature()).thenThrow(new RuntimeException("Error occurred while getting signature"));

        aspect.executionTimeLogger(joinPoint, executionTimeLogger);

        verify(joinPoint, times(1)).proceed();
        assertTrue(capturedOutput.toString().contains("Error occurred while logging execution time"));
    }
}
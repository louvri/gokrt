package icontext_test

import (
	"context"
	"testing"
	"time"

	customContext "github.com/louvri/gokrt/context"
	"github.com/louvri/gokrt/sys_key"
)

func TestWithoutDeadline_RemovesDeadline(t *testing.T) {
	baseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, hasDeadline := baseCtx.Deadline(); !hasDeadline {
		t.Fatal("Base context should have a deadline")
	}

	ctx := customContext.New(baseCtx).(*customContext.Context)

	if deadline, hasDeadline := ctx.Deadline(); !hasDeadline {
		t.Fatal("Context should have a deadline before WithoutDeadline()")
	} else {
		t.Logf("Original deadline: %v", deadline)
	}

	newCtx := ctx.WithoutDeadline()

	if deadline, hasDeadline := newCtx.Deadline(); hasDeadline {
		t.Errorf("Context should not have a deadline after WithoutDeadline(), but got: %v", deadline)
	} else {
		t.Log("✓ Deadline successfully removed")
	}
}

func TestWithoutDeadline_PreservesAllProperties(t *testing.T) {
	baseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ctx := customContext.New(baseCtx).(*customContext.Context)

	testData := map[sys_key.SysKey]any{
		sys_key.FILE_KEY:        "test_file.txt",
		sys_key.FILE_OBJECT_KEY: map[string]string{"name": "file1"},
		sys_key.SOF:             int64(0),
		sys_key.EOF:             int64(1024),
		sys_key.DATA_REF:        []byte("test data"),
		sys_key.CACHE_KEY:       "cache_123",
		sys_key.GOKRT_CONTEXT:   "gokrt_value",
	}

	for key, value := range testData {
		ctx.Set(key, value)
	}

	ctx.Set("custom_key_1", "custom_value_1")
	ctx.Set("custom_key_2", 42)

	newCtx := ctx.WithoutDeadline().(*customContext.Context)

	t.Run("VerifySystemProperties", func(t *testing.T) {
		for key, expectedValue := range testData {
			actualValue := newCtx.Get(key)

			if actual, ok := actualValue.(map[string]string); ok {
				expected := expectedValue.(map[string]string)
				for key, val := range expected {
					if actual[key] != val {
						t.Errorf("Property %v: expected %v, got %v", key, expectedValue, actualValue)
					}
				}
			} else if actual, ok := actualValue.(map[string]any); ok {
				expected := expectedValue.(map[string]any)
				for key, val := range expected {
					if actual[key] != val {
						t.Errorf("Property %v: expected %v, got %v", key, expectedValue, actualValue)
					}
				}
			} else if actual, ok := actualValue.([]uint8); ok {
				expected := expectedValue.([]uint8)
				if string(actual) != string(expected) {
					t.Errorf("Property %v: expected %v, got %v", key, expectedValue, actualValue)
				}

			} else if actualValue != expectedValue {
				t.Errorf("Property %v: expected %v, got %v", key, expectedValue, actualValue)
			} else {
				t.Logf("✓ Property %v preserved: %v", key, actualValue)
			}
		}
	})

	t.Run("VerifyCustomProperties", func(t *testing.T) {
		if val := newCtx.Value("custom_key_1"); val != "custom_value_1" {
			t.Errorf("Custom property custom_key_1: expected 'custom_value_1', got %v", val)
		} else {
			t.Logf("✓ Custom property custom_key_1 preserved: %v", val)
		}

		if val := newCtx.Value("custom_key_2"); val != 42 {
			t.Errorf("Custom property custom_key_2: expected 42, got %v", val)
		} else {
			t.Logf("✓ Custom property custom_key_2 preserved: %v", val)
		}
	})

	t.Run("VerifyDeadlineRemoved", func(t *testing.T) {
		if _, hasDeadline := newCtx.Deadline(); hasDeadline {
			t.Error("Context should not have a deadline after WithoutDeadline()")
		} else {
			t.Log("✓ Deadline successfully removed")
		}
	})
}

func TestWithoutDeadline_OriginalContextUnaffected(t *testing.T) {
	baseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ctx := customContext.New(baseCtx).(*customContext.Context)
	ctx.Set(sys_key.FILE_KEY, "original_file.txt")

	originalDeadline, hadDeadline := ctx.Deadline()
	if !hadDeadline {
		t.Fatal("Original context should have a deadline")
	}

	newCtx := ctx.WithoutDeadline().(*customContext.Context)

	currentDeadline, stillHasDeadline := ctx.Deadline()
	if !stillHasDeadline {
		t.Error("⚠️  Original context lost its deadline (mutation detected!)")
	} else if !currentDeadline.Equal(originalDeadline) {
		t.Error("⚠️  Original context deadline changed (mutation detected!)")
	} else {
		t.Log("✓ Original context deadline preserved")
	}

	if _, hasDeadline := newCtx.Deadline(); hasDeadline {
		t.Error("New context should not have a deadline")
	} else {
		t.Log("✓ New context has no deadline")
	}

	if ctx == newCtx {
		t.Error("⚠️  ctx and newCtx are the same instance (should be different!)")
	} else {
		t.Log("✓ New context is a separate instance")
	}
}

func TestWithoutDeadline_ChainedContexts(t *testing.T) {
	baseCtx := context.WithValue(context.Background(), "level", "base")
	level1Ctx, cancel := context.WithTimeout(baseCtx, 5*time.Second)
	defer cancel()
	level2Ctx := context.WithValue(level1Ctx, "user", "john")

	ctx := customContext.New(level2Ctx).(*customContext.Context)
	ctx.Set(sys_key.FILE_KEY, "test.txt")

	newCtx := ctx.WithoutDeadline()

	if _, hasDeadline := newCtx.Deadline(); hasDeadline {
		t.Error("New context should not have a deadline")
	}

	if val := newCtx.Value("level"); val != "base" {
		t.Errorf("Value 'level' from base context: expected 'base', got %v", val)
	} else {
		t.Log("✓ Base context value preserved")
	}

	if val := newCtx.Value("user"); val != "john" {
		t.Errorf("Value 'user' from level2 context: expected 'john', got %v", val)
	} else {
		t.Log("✓ Chained context value preserved")
	}

	if val := newCtx.(*customContext.Context).Get(sys_key.FILE_KEY); val != "test.txt" {
		t.Errorf("Property FILE_KEY: expected 'test.txt', got %v", val)
	} else {
		t.Log("✓ Custom context property preserved")
	}
}

func TestWithoutDeadline_DoneChannelBehavior(t *testing.T) {
	baseCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ctx := customContext.New(baseCtx).(*customContext.Context)

	if ctx.Done() == nil {
		t.Fatal("Original context should have a Done channel")
	}

	newCtx := ctx.WithoutDeadline()

	if newCtx.Done() != nil {
		t.Error("New context Done() should return nil")
	} else {
		t.Log("✓ Done channel is nil (no cancellation)")
	}

	time.Sleep(150 * time.Millisecond)

	select {
	case <-ctx.Done():
		t.Log("✓ Original context timed out as expected")
	default:
		t.Error("Original context should have timed out")
	}

	select {
	case <-newCtx.Done():
		t.Error("New context should never be done")
	default:
		t.Log("✓ New context is not done (immune to timeout)")
	}
}

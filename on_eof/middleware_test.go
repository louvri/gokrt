package on_eof

import (
	"context"
	"errors"
	"testing"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/sys_key"
	"github.com/stretchr/testify/assert"
)

// TestMiddleware_WithEOFContext tests middleware behavior when EOF is present in context
func TestMiddleware_WithEOFContext(t *testing.T) {
	// Arrange
	var executionOrder []string

	middleware1 := createTestMiddleware("middleware1", &executionOrder)
	middleware2 := createTestMiddleware("middleware2", &executionOrder)

	baseEndpoint := func(ctx context.Context, req any) (any, error) {
		executionOrder = append(executionOrder, "base_endpoint")
		return "base_result", nil
	}

	ctx := context.WithValue(context.Background(), sys_key.EOF, true)

	// Act
	wrappedEndpoint := Middleware(middleware1, middleware2)(baseEndpoint)
	result, err := wrappedEndpoint(ctx, "test_request")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "", result) // Should return empty string when EOF is present
	// Middlewares should execute in reverse order, base endpoint should NOT execute
	assert.Equal(t, []string{"middleware1", "middleware2"}, executionOrder)
}

// TestMiddleware_WithoutEOFContext tests normal middleware execution
func TestMiddleware_WithoutEOFContext(t *testing.T) {
	// Arrange
	var executionOrder []string

	middleware1 := createTestMiddleware("middleware1", &executionOrder)
	middleware2 := createTestMiddleware("middleware2", &executionOrder)

	baseEndpoint := func(ctx context.Context, req any) (any, error) {
		executionOrder = append(executionOrder, "base_endpoint")
		return "normal_result", nil
	}

	ctx := context.Background() // No EOF in context

	// Act
	wrappedEndpoint := Middleware(middleware1, middleware2)(baseEndpoint)
	result, err := wrappedEndpoint(ctx, "test_request")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "normal_result", result)
	// Should execute base endpoint normally, middlewares not applied
	assert.Equal(t, []string{"base_endpoint"}, executionOrder)
}

// TestMiddleware_EmptyMiddlewareList tests behavior with no middlewares
func TestMiddleware_EmptyMiddlewareList(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected any
	}{
		{
			name:     "with EOF context and no middlewares",
			ctx:      context.WithValue(context.Background(), sys_key.EOF, true),
			expected: "",
		},
		{
			name:     "without EOF context and no middlewares",
			ctx:      context.Background(),
			expected: "base_result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			baseEndpoint := func(ctx context.Context, req any) (any, error) {
				return "base_result", nil
			}

			// Act
			wrappedEndpoint := Middleware()(baseEndpoint) // No middlewares
			result, err := wrappedEndpoint(tt.ctx, "test_request")

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestMiddleware_EOFWithDifferentValues tests different EOF values
func TestMiddleware_EOFWithDifferentValues(t *testing.T) {
	tests := []struct {
		name          string
		eofValue      any
		shouldTrigger bool
	}{
		{"EOF with true", true, true},
		{"EOF with false", false, true},
		{"EOF with string", "end", true},
		{"EOF with number", 1, true},
		{"EOF with nil", nil, false}, // nil should not trigger EOF behavior
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			var middlewareExecuted bool
			middleware1 := func(next endpoint.Endpoint) endpoint.Endpoint {
				return func(ctx context.Context, req any) (any, error) {
					middlewareExecuted = true
					return next(ctx, req)
				}
			}

			baseEndpoint := func(ctx context.Context, req any) (any, error) {
				return "base_result", nil
			}

			var ctx context.Context
			if tt.eofValue != nil {
				ctx = context.WithValue(context.Background(), sys_key.EOF, tt.eofValue)
			} else {
				ctx = context.Background()
			}

			// Act
			wrappedEndpoint := Middleware(middleware1)(baseEndpoint)
			result, err := wrappedEndpoint(ctx, "test")

			// Assert
			assert.NoError(t, err)
			if tt.shouldTrigger {
				assert.Equal(t, "", result)
				assert.True(t, middlewareExecuted)
			} else {
				assert.Equal(t, "base_result", result)
				assert.False(t, middlewareExecuted)
			}
		})
	}
}

// TestMiddleware_ErrorPropagation tests error handling
func TestMiddleware_ErrorPropagation(t *testing.T) {
	expectedError := errors.New("middleware error")

	tests := []struct {
		name string
		ctx  context.Context
	}{
		{"error with EOF context", context.WithValue(context.Background(), sys_key.EOF, true)},
		{"error without EOF context", context.Background()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			errorMiddleware := func(next endpoint.Endpoint) endpoint.Endpoint {
				return func(ctx context.Context, req any) (any, error) {
					return nil, expectedError
				}
			}

			baseEndpoint := func(ctx context.Context, req any) (any, error) {
				return nil, expectedError // This should not be reached in EOF case
			}

			// Act
			wrappedEndpoint := Middleware(errorMiddleware)(baseEndpoint)
			result, err := wrappedEndpoint(tt.ctx, "test")

			// Assert
			assert.Error(t, err)
			assert.Equal(t, expectedError, err)
			assert.Nil(t, result)
		})
	}
}

// TestMiddleware_MiddlewareOrder tests that middlewares execute in reverse order during EOF
func TestMiddleware_MiddlewareOrder(t *testing.T) {
	// Arrange
	var executionOrder []string

	middleware1 := func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			executionOrder = append(executionOrder, "middleware1_start")
			result, err := next(ctx, req)
			executionOrder = append(executionOrder, "middleware1_end")
			return result, err
		}
	}

	middleware2 := func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			executionOrder = append(executionOrder, "middleware2_start")
			result, err := next(ctx, req)
			executionOrder = append(executionOrder, "middleware2_end")
			return result, err
		}
	}

	middleware3 := func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			executionOrder = append(executionOrder, "middleware3_start")
			result, err := next(ctx, req)
			executionOrder = append(executionOrder, "middleware3_end")
			return result, err
		}
	}

	baseEndpoint := func(ctx context.Context, req any) (any, error) {
		executionOrder = append(executionOrder, "base_endpoint")
		return "result", nil
	}

	ctx := context.WithValue(context.Background(), sys_key.EOF, true)

	// Act
	wrappedEndpoint := Middleware(middleware1, middleware2, middleware3)(baseEndpoint)
	result, err := wrappedEndpoint(ctx, "test")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "", result)

	// Should execute in reverse order: middleware3 -> middleware2 -> middleware1 -> empty endpoint
	expectedOrder := []string{
		"middleware1_start",
		"middleware2_start",
		"middleware3_start",
		"middleware3_end",
		"middleware2_end",
		"middleware1_end",
	}
	assert.Equal(t, expectedOrder, executionOrder)
}

// TestMiddleware_SingleMiddleware tests behavior with single middleware
func TestMiddleware_SingleMiddleware(t *testing.T) {
	tests := []struct {
		name                    string
		ctx                     context.Context
		expectedResult          any
		shouldExecuteMiddleware bool
	}{
		{
			name:                    "single middleware with EOF",
			ctx:                     context.WithValue(context.Background(), sys_key.EOF, true),
			expectedResult:          "",
			shouldExecuteMiddleware: true,
		},
		{
			name:                    "single middleware without EOF",
			ctx:                     context.Background(),
			expectedResult:          "base_result",
			shouldExecuteMiddleware: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			var middlewareExecuted bool
			middleware := func(next endpoint.Endpoint) endpoint.Endpoint {
				return func(ctx context.Context, req any) (any, error) {
					middlewareExecuted = true
					return next(ctx, req)
				}
			}

			baseEndpoint := func(ctx context.Context, req any) (any, error) {
				return "base_result", nil
			}

			// Act
			wrappedEndpoint := Middleware(middleware)(baseEndpoint)
			result, err := wrappedEndpoint(tt.ctx, "test")

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedResult, result)
			assert.Equal(t, tt.shouldExecuteMiddleware, middlewareExecuted)
		})
	}
}

// TestMiddleware_RequestPropagation tests that request is properly passed through
func TestMiddleware_RequestPropagation(t *testing.T) {
	// Arrange
	testRequest := map[string]any{
		"user_id": 123,
		"action":  "test",
	}

	var receivedRequests []any

	middleware1 := func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			receivedRequests = append(receivedRequests, req)
			return next(ctx, req)
		}
	}

	middleware2 := func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			receivedRequests = append(receivedRequests, req)
			return next(ctx, req)
		}
	}

	baseEndpoint := func(ctx context.Context, req any) (any, error) {
		receivedRequests = append(receivedRequests, req)
		return "result", nil
	}

	ctx := context.WithValue(context.Background(), sys_key.EOF, true)

	// Act
	wrappedEndpoint := Middleware(middleware1, middleware2)(baseEndpoint)
	_, err := wrappedEndpoint(ctx, testRequest)

	// Assert
	assert.NoError(t, err)
	// All middlewares should receive the same request
	assert.Len(t, receivedRequests, 2) // Only middlewares, not base endpoint
	for _, req := range receivedRequests {
		assert.Equal(t, testRequest, req)
	}
}

// Helper function to create test middleware that tracks execution order
func createTestMiddleware(name string, executionOrder *[]string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			*executionOrder = append(*executionOrder, name)
			return next(ctx, req)
		}
	}
}

// Benchmark tests
func BenchmarkMiddleware_WithEOF(b *testing.B) {
	middleware1 := func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			return next(ctx, req)
		}
	}

	middleware2 := func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			return next(ctx, req)
		}
	}

	baseEndpoint := func(ctx context.Context, req any) (any, error) {
		return "result", nil
	}

	wrappedEndpoint := Middleware(middleware1, middleware2)(baseEndpoint)
	ctx := context.WithValue(context.Background(), sys_key.EOF, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = wrappedEndpoint(ctx, "benchmark")
	}
}

func BenchmarkMiddleware_WithoutEOF(b *testing.B) {
	middleware1 := func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			return next(ctx, req)
		}
	}

	middleware2 := func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			return next(ctx, req)
		}
	}

	baseEndpoint := func(ctx context.Context, req any) (any, error) {
		return "result", nil
	}

	wrappedEndpoint := Middleware(middleware1, middleware2)(baseEndpoint)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = wrappedEndpoint(ctx, "benchmark")
	}
}

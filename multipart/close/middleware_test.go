package close

import (
	"context"
	"testing"

	icontext "github.com/louvri/gokrt/context"
	"github.com/louvri/gokrt/sys_key"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockConnection is a mock implementation of the connection.Connection interface
type MockConnection struct {
	mock.Mock
	driver string
	closed bool
}

func (m *MockConnection) Driver() string {
	return m.driver
}

func (m *MockConnection) Close() error {
	m.closed = true
	return m.Called().Error(0)
}

// IsClosed checks if the connection is closed
func (m *MockConnection) IsClosed() bool {
	return m.closed
}

// TestMiddleware_WithEOF tests the middleware behavior when EOF is present
func TestMiddleware_WithEOF(t *testing.T) {
	// Arrange
	mockConn1 := &MockConnection{driver: "multipart"}
	mockConn2 := &MockConnection{driver: "multipart"}
	mockConn3 := &MockConnection{driver: "database"} // Should not be closed

	mockConn1.On("Close").Return(nil)
	mockConn2.On("Close").Return(nil)

	fileObjects := map[string]any{
		"file1": mockConn1,
		"file2": mockConn2,
		"file3": mockConn3,
	}

	ctx := icontext.New(context.Background())
	ictx, _ := ctx.Value(sys_key.GOKRT_CONTEXT).(*icontext.Context)
	ictx.Set(sys_key.EOF, "true")
	ictx.Set(sys_key.FILE_OBJECT_KEY, fileObjects)

	ctx = context.WithValue(context.Background(), sys_key.GOKRT_CONTEXT, ictx)

	baseEndpoint := func(ctx context.Context, req any) (any, error) {
		return "success", nil
	}

	// Act
	middleware := Middleware()
	wrappedEndpoint := middleware(baseEndpoint)
	result, err := wrappedEndpoint(ctx, "test_request")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "success", result)

	// Verify that multipart connections were closed
	mockConn1.AssertCalled(t, "Close")
	mockConn2.AssertCalled(t, "Close")
	mockConn3.AssertNotCalled(t, "Close") // Database connection should not be closed

	assert.True(t, mockConn1.IsClosed())
	assert.True(t, mockConn2.IsClosed())
	assert.False(t, mockConn3.IsClosed())
}

// TestMiddleware_WithoutEOF tests that connections are not closed when EOF is not present
func TestMiddleware_WithoutEOF(t *testing.T) {
	// Arrange
	mockConn := &MockConnection{driver: "multipart"}
	mockConn.On("Close").Return(nil)

	fileObjects := map[string]any{
		"file1": mockConn,
	}

	ctx := icontext.New(context.Background())
	ictx, _ := ctx.Value(sys_key.GOKRT_CONTEXT).(*icontext.Context)
	ictx.Set(sys_key.EOF, "") // No EOF set
	ictx.Set(sys_key.FILE_OBJECT_KEY, fileObjects)

	ctx = context.WithValue(context.Background(), sys_key.GOKRT_CONTEXT, ictx)

	baseEndpoint := func(ctx context.Context, req any) (any, error) {
		return "success", nil
	}

	// Act
	middleware := Middleware()
	wrappedEndpoint := middleware(baseEndpoint)
	result, err := wrappedEndpoint(ctx, "test_request")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "success", result)

	// Verify that the connection was not closed
	mockConn.AssertNotCalled(t, "Close")
	assert.False(t, mockConn.IsClosed())
}

// TestMiddleware_WithoutInternalContext tests creating a new internal context when not present
func TestMiddleware_WithoutInternalContext(t *testing.T) {
	// Arrange
	ctx := context.Background() // No internal context

	baseEndpoint := func(ctx context.Context, req any) (any, error) {
		// Verify internal context was created
		ictx, ok := ctx.Value(sys_key.GOKRT_CONTEXT).(*icontext.Context)
		assert.True(t, ok)
		assert.NotNil(t, ictx)
		return "success", nil
	}

	// Act
	middleware := Middleware()
	wrappedEndpoint := middleware(baseEndpoint)
	result, err := wrappedEndpoint(ctx, "test_request")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "success", result)
}

// TestMiddleware_WithInvalidInternalContext tests handling invalid internal context
func TestMiddleware_WithInvalidInternalContext(t *testing.T) {
	// Arrange
	ctx := context.WithValue(context.Background(), sys_key.GOKRT_CONTEXT, "invalid_context")

	baseEndpoint := func(ctx context.Context, req any) (any, error) {
		// Verify new internal context was created
		ictx, ok := ctx.Value(sys_key.GOKRT_CONTEXT).(*icontext.Context)
		assert.True(t, ok)
		assert.NotNil(t, ictx)
		return "success", nil
	}

	// Act
	middleware := Middleware()
	wrappedEndpoint := middleware(baseEndpoint)
	result, err := wrappedEndpoint(ctx, "test_request")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "success", result)
}

// TestMiddleware_WithoutFileObjects tests behavior when FILE_OBJECT_KEY is not present
func TestMiddleware_WithoutFileObjects(t *testing.T) {
	// Arrange
	ctx := icontext.New(context.Background())
	ictx, _ := ctx.Value(sys_key.GOKRT_CONTEXT).(*icontext.Context)
	ictx.Set(sys_key.EOF, "true")
	// No FILE_OBJECT_KEY set

	ctx = context.WithValue(context.Background(), sys_key.GOKRT_CONTEXT, ictx)

	baseEndpoint := func(ctx context.Context, req any) (any, error) {
		return "success", nil
	}

	// Act
	middleware := Middleware()
	wrappedEndpoint := middleware(baseEndpoint)
	result, err := wrappedEndpoint(ctx, "test_request")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	// Should not panic or error when no file objects present
}

// TestMiddleware_WithInvalidFileObjects tests behavior when FILE_OBJECT_KEY is not a map
func TestMiddleware_WithInvalidFileObjects(t *testing.T) {
	// Arrange
	ctx := icontext.New(context.Background())
	ictx, _ := ctx.Value(sys_key.GOKRT_CONTEXT).(*icontext.Context)
	ictx.Set(sys_key.EOF, "true")
	ictx.Set(sys_key.FILE_OBJECT_KEY, "invalid_file_objects") // Not a map

	ctx = context.WithValue(context.Background(), sys_key.GOKRT_CONTEXT, ictx)

	baseEndpoint := func(ctx context.Context, req any) (any, error) {
		return "success", nil
	}

	// Act
	middleware := Middleware()
	wrappedEndpoint := middleware(baseEndpoint)
	result, err := wrappedEndpoint(ctx, "test_request")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	// Should not panic when file objects is not a map
}

// TestMiddleware_MixedConnectionTypes tests handling of mixed connection types
func TestMiddleware_MixedConnectionTypes(t *testing.T) {
	// Arrange
	multipartConn := &MockConnection{driver: "multipart"}
	databaseConn := &MockConnection{driver: "database"}
	httpConn := &MockConnection{driver: "http"}

	multipartConn.On("Close").Return(nil)

	fileObjects := map[string]any{
		"multipart_file": multipartConn,
		"db_connection":  databaseConn,
		"http_client":    httpConn,
		"string_value":   "not_a_connection",
		"number_value":   123,
	}

	ctx := icontext.New(context.Background())
	ictx, _ := ctx.Value(sys_key.GOKRT_CONTEXT).(*icontext.Context)
	ictx.Set(sys_key.EOF, "end_of_file")
	ictx.Set(sys_key.FILE_OBJECT_KEY, fileObjects)

	ctx = context.WithValue(context.Background(), sys_key.GOKRT_CONTEXT, ictx)

	baseEndpoint := func(ctx context.Context, req any) (any, error) {
		return "success", nil
	}

	// Act
	middleware := Middleware()
	wrappedEndpoint := middleware(baseEndpoint)
	result, err := wrappedEndpoint(ctx, "test_request")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "success", result)

	// Only multipart connection should be closed
	multipartConn.AssertCalled(t, "Close")
	databaseConn.AssertNotCalled(t, "Close")
	httpConn.AssertNotCalled(t, "Close")

	assert.True(t, multipartConn.IsClosed())
	assert.False(t, databaseConn.IsClosed())
	assert.False(t, httpConn.IsClosed())
}

// TestMiddleware_EOFWithDifferentValues tests different EOF values
func TestMiddleware_EOFWithDifferentValues(t *testing.T) {
	tests := []struct {
		name        string
		eofValue    any
		shouldClose bool
	}{
		{"EOF with string 'true'", "true", true},
		{"EOF with string 'end'", "end", true},
		{"EOF with boolean true", true, true},
		{"EOF with boolean false", false, true},
		{"EOF with number", 1, true},
		{"EOF with nil", nil, false},
		{"EOF with empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockConn := &MockConnection{driver: "multipart"}
			if tt.shouldClose {
				mockConn.On("Close").Return(nil)
			}

			fileObjects := map[string]any{
				"file1": mockConn,
			}

			ctx := icontext.New(context.Background())
			ictx, _ := ctx.Value(sys_key.GOKRT_CONTEXT).(*icontext.Context)
			ictx.Set(sys_key.EOF, tt.eofValue)
			ictx.Set(sys_key.FILE_OBJECT_KEY, fileObjects)

			ctx = context.WithValue(context.Background(), sys_key.GOKRT_CONTEXT, ictx)

			baseEndpoint := func(ctx context.Context, req any) (any, error) {
				return "success", nil
			}

			// Act
			middleware := Middleware()
			wrappedEndpoint := middleware(baseEndpoint)
			result, err := wrappedEndpoint(ctx, "test_request")

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, "success", result)

			if tt.shouldClose {
				mockConn.AssertCalled(t, "Close")
				assert.True(t, mockConn.IsClosed())
			} else {
				mockConn.AssertNotCalled(t, "Close")
				assert.False(t, mockConn.IsClosed())
			}
		})
	}
}

// TestMiddleware_ErrorPropagation tests that errors from the next endpoint are properly propagated
func TestMiddleware_ErrorPropagation(t *testing.T) {
	// Arrange
	expectedError := assert.AnError

	ictx := icontext.New(context.Background())
	ctx := context.WithValue(context.Background(), sys_key.GOKRT_CONTEXT, ictx)

	baseEndpoint := func(ctx context.Context, req any) (any, error) {
		return nil, expectedError
	}

	// Act
	middleware := Middleware()
	wrappedEndpoint := middleware(baseEndpoint)
	result, err := wrappedEndpoint(ctx, "test_request")

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Nil(t, result)
}

// TestMiddleware_ConnectionCloseError tests handling of connection close errors
func TestMiddleware_ConnectionCloseError(t *testing.T) {
	// Arrange
	closeError := assert.AnError
	mockConn := &MockConnection{driver: "multipart"}
	mockConn.On("Close").Return(closeError)

	fileObjects := map[string]any{
		"file1": mockConn,
	}

	ctx := icontext.New(context.Background())
	ictx, _ := ctx.Value(sys_key.GOKRT_CONTEXT).(*icontext.Context)
	ictx.Set(sys_key.EOF, "true")
	ictx.Set(sys_key.FILE_OBJECT_KEY, fileObjects)

	ctx = context.WithValue(context.Background(), sys_key.GOKRT_CONTEXT, ictx)

	baseEndpoint := func(ctx context.Context, req any) (any, error) {
		return "success", nil
	}

	// Act
	middleware := Middleware()
	wrappedEndpoint := middleware(baseEndpoint)
	result, err := wrappedEndpoint(ctx, "test_request")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "success", result)

	// Verify that the connection close error was handled
	mockConn.AssertCalled(t, "Close")
	assert.True(t, mockConn.IsClosed())
}

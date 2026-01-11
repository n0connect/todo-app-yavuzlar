package auth

import "context"

type contextKey string

const userUUIDKey contextKey = "user_uuid"

// WithUserUUID adds user UUID to context
func WithUserUUID(ctx context.Context, userUUID string) context.Context {
	return context.WithValue(ctx, userUUIDKey, userUUID)
}

// GetUserUUID retrieves user UUID from context
func GetUserUUID(ctx context.Context) (string, bool) {
	userUUID, ok := ctx.Value(userUUIDKey).(string)
	return userUUID, ok
}

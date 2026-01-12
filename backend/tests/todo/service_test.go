package todo

import (
	"testing"

	"github.com/google/uuid"

	"todo-app-backend/internal/encryption"
	"todo-app-backend/tests/testutil"
)

func TestBuildAAD_Format(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	userUUID := "test-user-uuid-123456789012"
	todoID, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	field := encryption.FieldTitle
	purpose := encryption.PurposeStoredRecord

	aad := encryption.BuildAAD(userUUID, todoID, field, purpose)

	// AAD should be exactly 39 bytes
	if len(aad) != 39 {
		t.Errorf("BuildAAD() length = %d, want 39", len(aad))
	}

	// AAD should start with magic bytes
	expectedMagic := []byte{'P', 'X', 'A', 'D'}
	for i := 0; i < 4; i++ {
		if aad[i] != expectedMagic[i] {
			t.Errorf("BuildAAD() magic byte[%d] = %d, want %d", i, aad[i], expectedMagic[i])
		}
	}
}

func TestBuildAAD_DifferentUsers_DifferentAAD(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	todoID, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	field := encryption.FieldTitle
	purpose := encryption.PurposeStoredRecord

	userUUID1 := "user1-uuid-123456789012"
	userUUID2 := "user2-uuid-987654321098"

	aad1 := encryption.BuildAAD(userUUID1, todoID, field, purpose)
	aad2 := encryption.BuildAAD(userUUID2, todoID, field, purpose)

	// AADs should be different for different users
	if string(aad1) == string(aad2) {
		t.Error("BuildAAD() should produce different AADs for different users")
	}
}

func TestBuildAAD_DifferentFields_DifferentAAD(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	userUUID := "test-user-uuid-123456789012"
	todoID, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	purpose := encryption.PurposeStoredRecord

	aadTitle := encryption.BuildAAD(userUUID, todoID, encryption.FieldTitle, purpose)
	aadTags := encryption.BuildAAD(userUUID, todoID, encryption.FieldTags, purpose)

	// AADs should be different for different fields
	if string(aadTitle) == string(aadTags) {
		t.Error("BuildAAD() should produce different AADs for different fields")
	}
}

func TestBuildAAD_DifferentTodoIDs_DifferentAAD(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	userUUID := "test-user-uuid-123456789012"
	todoID1, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	todoID2, _ := uuid.Parse("660e8400-e29b-41d4-a716-446655440001")
	field := encryption.FieldTitle
	purpose := encryption.PurposeStoredRecord

	aad1 := encryption.BuildAAD(userUUID, todoID1, field, purpose)
	aad2 := encryption.BuildAAD(userUUID, todoID2, field, purpose)

	// AADs should be different for different todo IDs
	if string(aad1) == string(aad2) {
		t.Error("BuildAAD() should produce different AADs for different todo IDs")
	}
}

func TestValidateField(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	validFields := []encryption.Field{
		encryption.FieldTitle,
		encryption.FieldContent,
		encryption.FieldTags,
	}

	for _, field := range validFields {
		if err := encryption.ValidateField(field); err != nil {
			t.Errorf("ValidateField(%d) should be valid, got error: %v", field, err)
		}
	}

	// Test invalid field
	invalidField := encryption.Field(0xFF)
	if err := encryption.ValidateField(invalidField); err == nil {
		t.Error("ValidateField() with invalid field should have failed")
	}
}

func TestValidatePurpose(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	validPurpose := encryption.PurposeStoredRecord
	if err := encryption.ValidatePurpose(validPurpose); err != nil {
		t.Errorf("ValidatePurpose(%d) should be valid, got error: %v", validPurpose, err)
	}

	// Test invalid purpose
	invalidPurpose := encryption.Purpose(0xFF)
	if err := encryption.ValidatePurpose(invalidPurpose); err == nil {
		t.Error("ValidatePurpose() with invalid purpose should have failed")
	}
}

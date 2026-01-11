package encryption

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"todo-app-backend/internal/utils"

	"github.com/google/uuid"
)

var aadLogger = utils.NewLogger("AAD")

// Field represents the encrypted field type
type Field uint8

const (
	FieldTitle   Field = 0x01
	FieldContent Field = 0x02
	FieldTags    Field = 0x03
	// Future: FieldAttachment = 0x04
)

// Purpose represents the encryption purpose/context
type Purpose uint8

const (
	PurposeStoredRecord Purpose = 0x01
	// Future: PurposeResponsePayload = 0x02, PurposeMigration = 0x03
)

var (
	aadMagic   = [4]byte{'P', 'X', 'A', 'D'} // Magic bytes to prevent protocol confusion
	aadVersion = byte(0x01)                  // Format version for future compatibility
)

// BuildAAD returns a compact, unambiguous, versioned AAD blob.
// Format: MAGIC(4) || VER(1) || USER_TAG(16) || TODO_ID(16) || FIELD(1) || PURPOSE(1) = 39 bytes
// USER_TAG = SHA-256(userUUIDString)[:16] to keep a fixed 16-byte field.
// NOTE: caller must ensure userUUID and todoID are server-derived (JWT/DB), not client input.
func BuildAAD(userUUID string, todoID uuid.UUID, field Field, purpose Purpose) []byte {
	aadLogger.Debug("BuildAAD: building AAD for userUUID=%s todoID=%s field=%d purpose=%d", userUUID, todoID.String(), field, purpose)
	// Fixed-size layout: 39 bytes
	aad := make([]byte, 0, 39)
	aad = append(aad, aadMagic[:]...)
	aad = append(aad, aadVersion)
	userTag := sha256.Sum256([]byte(userUUID))
	aad = append(aad, userTag[:16]...) // 16 bytes (user_tag)
	aad = append(aad, todoID[:]...)    // 16 bytes (UUID binary format)
	aad = append(aad, byte(field))
	aad = append(aad, byte(purpose))
	aadLogger.Debug("BuildAAD: AAD built successfully: length=%d hex=%s", len(aad), hex.EncodeToString(aad))
	return aad
}

// ValidateField validates that a field code is valid
func ValidateField(field Field) error {
	switch field {
	case FieldTitle, FieldContent, FieldTags:
		return nil
	default:
		return fmt.Errorf("invalid field code: %d", field)
	}
}

// ValidatePurpose validates that a purpose code is valid
func ValidatePurpose(purpose Purpose) error {
	switch purpose {
	case PurposeStoredRecord:
		return nil
	default:
		return fmt.Errorf("invalid purpose code: %d", purpose)
	}
}

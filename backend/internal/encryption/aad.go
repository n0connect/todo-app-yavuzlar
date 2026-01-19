package encryption

import (
	"encoding/hex"
	"fmt"

	"todo-app-backend/internal/cryptoengine"
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
	aadMagic = [4]byte{'P', 'X', 'A', 'D'} // Magic bytes to prevent protocol confusion
)

const (
	AADVersion = 0x02
	AADLength  = 55
)

// BuildAAD returns a compact, unambiguous, versioned AAD blob (v2).
// Format: MAGIC(4) || VER(1) || USER_TAG(32) || TODO_ID(16) || FIELD(1) || PURPOSE(1) = 55 bytes
// USER_TAG = SHA-256(userUUIDString) to reduce collision risk.
// NOTE: caller must ensure userUUID and todoID are server-derived (JWT/DB), not client input.
func BuildAAD(userUUID string, todoID uuid.UUID, field Field, purpose Purpose) []byte {
	aadLogger.Debug("BuildAAD: building AAD for userUUID=%s todoID=%s field=%d purpose=%d", userUUID, todoID.String(), field, purpose)
	aad := make([]byte, 0, AADLength)
	aad = append(aad, aadMagic[:]...)
	aad = append(aad, byte(AADVersion))
	userTag, err := cryptoengine.SHA256([]byte(userUUID))
	if err != nil {
		aadLogger.LogError("SHA256 (user tag)", err)
		return nil
	}
	aad = append(aad, userTag...)
	aad = append(aad, todoID[:]...)
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

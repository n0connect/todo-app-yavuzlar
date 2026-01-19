package encryption

import (
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"todo-app-backend/internal/config"
	"todo-app-backend/internal/utils"
)

var masterKeyLogger = utils.NewLogger("MASTER_KEY")

type masterKeySet struct {
	activeID string
	active   []byte
	old      map[string][]byte
}

var (
	masterKeys     masterKeySet
	masterKeyOnce  sync.Once
	masterKeyError error
)

func InitMasterKeys() error {
	masterKeyOnce.Do(func() {
		activeID := strings.TrimSpace(config.GetEnv("MASTER_KEY_ACTIVE_ID", ""))
		activeKeyRaw := strings.TrimSpace(config.GetEnv("MASTER_KEY_ACTIVE", ""))

		if activeKeyRaw == "" {
			masterKeyError = fmt.Errorf("MASTER_KEY_ACTIVE is required")
			return
		}
		if activeID == "" {
			masterKeyError = fmt.Errorf("MASTER_KEY_ACTIVE_ID is required")
			return
		}

		activeKey, err := decodeKeyMaterial(activeKeyRaw)
		if err != nil {
			masterKeyError = fmt.Errorf("invalid active master key: %w", err)
			return
		}

		oldKeys := make(map[string][]byte)
		oldRaw := strings.TrimSpace(config.GetEnv("MASTER_KEY_OLD", ""))
		if oldRaw != "" {
			for _, entry := range strings.Split(oldRaw, ",") {
				entry = strings.TrimSpace(entry)
				if entry == "" {
					continue
				}
				parts := strings.SplitN(entry, ":", 2)
				if len(parts) != 2 {
					masterKeyLogger.Warn("MASTER_KEY_OLD entry ignored (missing kid): %s", entry)
					continue
				}
				kid := strings.TrimSpace(parts[0])
				keyRaw := strings.TrimSpace(parts[1])
				if kid == "" || keyRaw == "" {
					masterKeyLogger.Warn("MASTER_KEY_OLD entry ignored (empty kid/key)")
					continue
				}
				key, err := decodeKeyMaterial(keyRaw)
				if err != nil {
					masterKeyLogger.Warn("MASTER_KEY_OLD entry ignored (invalid key) kid=%s", kid)
					continue
				}
				oldKeys[kid] = key
			}
		}

		masterKeys = masterKeySet{
			activeID: activeID,
			active:   activeKey,
			old:      oldKeys,
		}
		masterKeyLogger.Info("Master keys initialized: active_id=%s old_keys=%d", activeID, len(oldKeys))
	})
	return masterKeyError
}

func ActiveMasterKey() []byte {
	if err := InitMasterKeys(); err != nil {
		panic(err)
	}
	return masterKeys.active
}

func ActiveMasterKeyID() string {
	if err := InitMasterKeys(); err != nil {
		panic(err)
	}
	return masterKeys.activeID
}

func MasterKeyByID(id string) ([]byte, bool) {
	if err := InitMasterKeys(); err != nil {
		return nil, false
	}
	if id == masterKeys.activeID {
		return masterKeys.active, true
	}
	key, ok := masterKeys.old[id]
	return key, ok
}

func HasOldMasterKeys() bool {
	if err := InitMasterKeys(); err != nil {
		return false
	}
	return len(masterKeys.old) > 0
}

func decodeKeyMaterial(keyRaw string) ([]byte, error) {
	if decoded, err := hex.DecodeString(keyRaw); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if len(keyRaw) == 32 {
		return []byte(keyRaw), nil
	}
	return nil, fmt.Errorf("invalid key length")
}

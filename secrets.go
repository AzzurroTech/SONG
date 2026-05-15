package song

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

type SecretsManager struct {
	secrets map[string]string
	mu      sync.RWMutex
	loaded  bool
}

var globalSecrets *SecretsManager

func InitSecrets() error {
	sm := &SecretsManager{
		secrets: make(map[string]string),
	}

	sm.loadFromEnv()

	secretsFile := os.Getenv("SONG_SECRETS_FILE")
	if secretsFile != "" {
		if err := sm.loadFromFile(secretsFile); err != nil {
			return fmt.Errorf("failed to load secrets file: %w", err)
		}
	}

	sm.loaded = true
	globalSecrets = sm

	log.Printf("Secrets initialized: %d secret(s) loaded", len(sm.secrets))
	return nil
}

func (sm *SecretsManager) loadFromEnv() {
	const prefix = "SONG_SECRET_"
	for _, envVar := range os.Environ() {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.HasPrefix(parts[0], prefix) {
			secretName := strings.TrimPrefix(parts[0], prefix)
			sm.secrets[secretName] = parts[1]
		}
	}
}

func (sm *SecretsManager) loadFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot open secrets file %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			log.Printf("Skipping invalid line %d in secrets file", lineNum)
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)
		sm.secrets[key] = value
	}

	return scanner.Err()
}

func GetSecret(name string) (string, bool) {
	if globalSecrets == nil {
		return "", false
	}
	globalSecrets.mu.RLock()
	defer globalSecrets.mu.RUnlock()
	val, ok := globalSecrets.secrets[strings.ToUpper(name)]
	return val, ok
}

func GetSecretOrDefault(name, defaultVal string) string {
	if val, ok := GetSecret(name); ok {
		return val
	}
	return defaultVal
}

func GetSecrets() map[string]string {
	if globalSecrets == nil {
		return map[string]string{}
	}
	globalSecrets.mu.RLock()
	defer globalSecrets.mu.RUnlock()

	copy := make(map[string]string, len(globalSecrets.secrets))
	for k, v := range globalSecrets.secrets {
		copy[k] = v
	}
	return copy
}

func RequireSecret(name string) string {
	val, ok := GetSecret(name)
	if !ok {
		panic(fmt.Sprintf("required secret %s is not set", name))
	}
	return val
}

func ReloadSecrets() error {
	return InitSecrets()
}

func SecretCount() int {
	if globalSecrets == nil {
		return 0
	}
	globalSecrets.mu.RLock()
	defer globalSecrets.mu.RUnlock()
	return len(globalSecrets.secrets)
}

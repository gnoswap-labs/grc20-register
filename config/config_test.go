package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_ValidateConfig(t *testing.T) {
	t.Parallel()

	t.Run("invalid chain ID", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultConfig()
		cfg.ChainID = "" // empty

		assert.ErrorIs(t, ValidateConfig(cfg), ErrInvalidChainID)
	})

	t.Run("invalid mnemonic", func(t *testing.T) {
		t.Parallel()

		cfg := DefaultConfig()
		cfg.Mnemonic = "maybe valid mnemonic" // invalid mnemonic

		assert.ErrorIs(t, ValidateConfig(cfg), ErrInvalidMnemonic)
	})

	t.Run("valid configuration", func(t *testing.T) {
		t.Parallel()

		assert.NoError(t, ValidateConfig(DefaultConfig()))
	})
}

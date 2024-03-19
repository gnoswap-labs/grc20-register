package config

import (
	"errors"
	"fmt"

	"github.com/gnolang/gno/tm2/pkg/crypto/bip39"
)

const (
	DefaultRpcUrl  = "http://localhost:26657" // r3v4_xxx: HARDCODED
	DefaultChainID = "dev"                    // r3v4_xxx: HARDCODED
	//nolint:lll // Mnemonic is naturally long
	DefaultMnemonic = "source bonus chronic canvas draft south burst lottery vacant surface solve popular case indicate oppose farm nothing bullet exhibit title speed wink action roast"
)

var (
	ErrInvalidRpcUrl   = errors.New("invalid rpc url")
	ErrInvalidChainID  = errors.New("invalid chain ID")
	ErrInvalidMnemonic = errors.New("invalid mnemonic")
)

// Config defines the base-level Register configuration
type Config struct {
	// The chain ID associated with the remote Gno chain
	ChainID string `toml:"chain_id"`

	// The mnemonic for the register
	Mnemonic string `toml:"mnemonic"`
}

// DefaultConfig returns the default register configuration
func DefaultConfig() *Config {
	return &Config{
		ChainID:  DefaultChainID,
		Mnemonic: DefaultMnemonic,
	}
}

// ValidateConfig validates the register configuration
func ValidateConfig(config *Config) error {
	// validate the chain ID
	if config.ChainID == "" {
		return ErrInvalidChainID
	}

	// validate the mnemonic is bip39-compliant
	if !bip39.IsMnemonicValid(config.Mnemonic) {
		return fmt.Errorf("%w, %s", ErrInvalidMnemonic, config.Mnemonic)
	}

	return nil
}

// Package render switches command output between human-readable and JSON.
package render

import (
	"encoding/json"
	"fmt"
	"os"
)

// JSON prints v as indented JSON to stdout.
func JSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encoding JSON output: %w", err)
	}
	return nil
}

// Auto runs jsonFn when asJSON is true, otherwise humanFn. Either may be nil.
func Auto(asJSON bool, jsonFn func() error, humanFn func() error) error {
	if asJSON {
		if jsonFn != nil {
			return jsonFn()
		}
		return nil
	}
	if humanFn != nil {
		return humanFn()
	}
	return nil
}

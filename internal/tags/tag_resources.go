package tags

import (
	"errors"
	"fmt"
	"strings"
)

const (
	// Separator used in resource identifiers.
	resourceIDSeparator = `,`
)

var errInvalidResourceIdentifier = errors.New("invalid resource identifier")

// GetResourceID parses a given resource identifier for tag identifier and tag key.
func GetResourceID(resourceID string) (string, string, error) {
	parts := strings.SplitN(resourceID, resourceIDSeparator, 2)

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("%w (%[2]s), expected ID%[3]sKEY", errInvalidResourceIdentifier, resourceID, resourceIDSeparator)
	}

	return parts[0], parts[1], nil
}

// SetResourceID creates a resource identifier given a tag identifier and a tag key.
func SetResourceID(identifier string, key string) string {
	parts := []string{identifier, key}
	resourceID := strings.Join(parts, resourceIDSeparator)

	return resourceID
}

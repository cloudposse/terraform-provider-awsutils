package provider

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var errInvalidAssumeRoleDuration = errors.New("duration must be between 15 minutes (15m) and 12 hours (12h), inclusive")

// validAssumeRoleDuration validates a string can be parsed as a valid time.Duration
// and is within a minimum of 15 minutes and maximum of 12 hours.
func validAssumeRoleDuration(v interface{}, k string) (ws []string, errs []error) {
	duration, err := time.ParseDuration(v.(string))
	if err != nil {
		errs = append(errs, fmt.Errorf("%q cannot be parsed as a duration: %w", k, err))
		return ws, errs
	}

	if duration.Minutes() < 15 || duration.Hours() > 12 {
		errs = append(errs, fmt.Errorf("%w: %q", errInvalidAssumeRoleDuration, k))
	}

	return ws, errs
}

var validAssumeRoleSessionName = validation.All(
	validation.StringLenBetween(2, 64),
	validation.StringMatch(regexp.MustCompile(`[\w+=,.@\-]*`), ""),
)

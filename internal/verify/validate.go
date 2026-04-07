package verify

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/structure"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var (
	accountIDRegexp = regexp.MustCompile(`^(aws|\d{12})$`)
	partitionRegexp = regexp.MustCompile(`^aws(-[a-z]+)*$`)
	regionRegexp    = regexp.MustCompile(`^[a-z]{2}(-[a-z]+)+-\d$`)
)

var errValidation = errors.New("validation error")

func ValidARN(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if value == "" {
		return ws, errors
	}

	parsedARN, err := arn.Parse(value)
	if err != nil {
		errors = append(errors, fmt.Errorf("%w: %q (%s) is an invalid ARN: %w", errValidation, k, value, err))
		return ws, errors
	}

	if parsedARN.Partition == "" {
		errors = append(errors, fmt.Errorf("%w: %q (%s) is an invalid ARN: missing partition value", errValidation, k, value))
	} else if !partitionRegexp.MatchString(parsedARN.Partition) {
		errors = append(errors, fmt.Errorf("%w: %q (%s) is an invalid ARN: invalid partition value (expecting to match regular expression: %s)", errValidation, k, value, partitionRegexp))
	}

	if parsedARN.Region != "" && !regionRegexp.MatchString(parsedARN.Region) {
		errors = append(errors, fmt.Errorf("%w: %q (%s) is an invalid ARN: invalid region value (expecting to match regular expression: %s)", errValidation, k, value, regionRegexp))
	}

	if parsedARN.AccountID != "" && !accountIDRegexp.MatchString(parsedARN.AccountID) {
		errors = append(errors, fmt.Errorf("%w: %q (%s) is an invalid ARN: invalid account ID value (expecting to match regular expression: %s)", errValidation, k, value, accountIDRegexp))
	}

	if parsedARN.Resource == "" {
		errors = append(errors, fmt.Errorf("%w: %q (%s) is an invalid ARN: missing resource value", errValidation, k, value))
	}

	return ws, errors
}

func ValidAccountID(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	// http://docs.aws.amazon.com/lambda/latest/dg/API_AddPermission.html
	pattern := `^\d{12}$`
	if !regexp.MustCompile(pattern).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%w: %q doesn't look like AWS Account ID (exactly 12 digits): %q",
			errValidation, k, value))
	}

	return ws, errors
}

// validateCIDRBlock validates that the specified CIDR block is valid:
// - The CIDR block parses to an IP address and network
// - The CIDR block is the CIDR block for the network.
func validateCIDRBlock(cidr string) error {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("%w: %q is not a valid CIDR block: %w", errValidation, cidr, err)
	}

	if !CIDRBlocksEqual(cidr, ipnet.String()) {
		return fmt.Errorf("%w: %q is not a valid CIDR block; did you mean %q?", errValidation, cidr, ipnet)
	}

	return nil
}

// ValidCIDRNetworkAddress ensures that the string value is a valid CIDR that
// represents a network address - it adds an error otherwise.
func ValidCIDRNetworkAddress(v interface{}, k string) (ws []string, errors []error) {
	if err := validateCIDRBlock(v.(string)); err != nil {
		errors = append(errors, err)
		return ws, errors
	}

	return ws, errors
}

func ValidIAMPolicyJSON(v interface{}, k string) (ws []string, errors []error) {
	// IAM Policy documents need to be valid JSON, and pass legacy parsing
	value := v.(string)
	if len(value) < 1 {
		errors = append(errors, fmt.Errorf("%w: %q contains an invalid JSON policy", errValidation, k))
		return ws, errors
	}
	if value[:1] != "{" {
		errors = append(errors, fmt.Errorf("%w: %q contains an invalid JSON policy", errValidation, k))
		return ws, errors
	}
	if _, err := structure.NormalizeJsonString(v); err != nil {
		errors = append(errors, fmt.Errorf("%w: %q contains an invalid JSON: %w", errValidation, k, err))
	}
	return ws, errors
}

// ValidateIPv4CIDRBlock validates that the specified CIDR block is valid:
// - The CIDR block parses to an IP address and network
// - The IP address is an IPv4 address
// - The CIDR block is the CIDR block for the network.
func ValidateIPv4CIDRBlock(cidr string) error {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("%w: %q is not a valid CIDR block: %w", errValidation, cidr, err)
	}

	ipv4 := ip.To4()
	if ipv4 == nil {
		return fmt.Errorf("%w: %q is not a valid IPv4 CIDR block", errValidation, cidr)
	}

	if !CIDRBlocksEqual(cidr, ipnet.String()) {
		return fmt.Errorf("%w: %q is not a valid IPv4 CIDR block; did you mean %q?", errValidation, cidr, ipnet)
	}

	return nil
}

// ValidateIPv6CIDRBlock validates that the specified CIDR block is valid:
// - The CIDR block parses to an IP address and network
// - The IP address is an IPv6 address
// - The CIDR block is the CIDR block for the network.
func ValidateIPv6CIDRBlock(cidr string) error {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("%w: %q is not a valid CIDR block: %w", errValidation, cidr, err)
	}

	ipv4 := ip.To4()
	if ipv4 != nil {
		return fmt.Errorf("%w: %q is not a valid IPv6 CIDR block", errValidation, cidr)
	}

	if !CIDRBlocksEqual(cidr, ipnet.String()) {
		return fmt.Errorf("%w: %q is not a valid IPv6 CIDR block; did you mean %q?", errValidation, cidr, ipnet)
	}

	return nil
}

// ValidIPv4CIDRNetworkAddress ensures that the string value is a valid IPv4 CIDR that
// represents a network address - it adds an error otherwise.
func ValidIPv4CIDRNetworkAddress(v interface{}, k string) (ws []string, errors []error) {
	if err := ValidateIPv4CIDRBlock(v.(string)); err != nil {
		errors = append(errors, err)
		return ws, errors
	}

	return ws, errors
}

// ValidIPv6CIDRNetworkAddress ensures that the string value is a valid IPv6 CIDR that
// represents a network address - it adds an error otherwise.
func ValidIPv6CIDRNetworkAddress(v interface{}, k string) (ws []string, errors []error) {
	if err := ValidateIPv6CIDRBlock(v.(string)); err != nil {
		errors = append(errors, err)
		return ws, errors
	}

	return ws, errors
}

func ValidLaunchTemplateID(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) < 1 {
		errors = append(errors, fmt.Errorf("%w: %q cannot be shorter than 1 character", errValidation, k))
	} else if len(value) > 255 {
		errors = append(errors, fmt.Errorf("%w: %q cannot be longer than 255 characters", errValidation, k))
	} else if !regexp.MustCompile(`^lt\-[a-z0-9]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%w: %q must begin with 'lt-' and be comprised of only alphanumeric characters: %v", errValidation, k, value))
	}
	return ws, errors
}

func ValidLaunchTemplateName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if len(value) < 3 {
		errors = append(errors, fmt.Errorf("%w: %q cannot be less than 3 characters", errValidation, k))
	} else if strings.HasSuffix(k, "prefix") && len(value) > 99 {
		errors = append(errors, fmt.Errorf("%w: %q cannot be longer than 99 characters, name is limited to 125", errValidation, k))
	} else if !strings.HasSuffix(k, "prefix") && len(value) > 125 {
		errors = append(errors, fmt.Errorf("%w: %q cannot be longer than 125 characters", errValidation, k))
	} else if !regexp.MustCompile(`^[0-9a-zA-Z()./_\-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf("%w: %q can only alphanumeric characters and ()./_- symbols", errValidation, k))
	}
	return ws, errors
}

func ValidOnceADayWindowFormat(v interface{}, k string) (ws []string, errors []error) {
	// valid time format is "hh24:mi"
	validTimeFormat := "([0-1][0-9]|2[0-3]):([0-5][0-9])"
	validTimeFormatConsolidated := "^(" + validTimeFormat + "-" + validTimeFormat + "|)$"

	value := v.(string)
	if !regexp.MustCompile(validTimeFormatConsolidated).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%w: %q must satisfy the format of \"hh24:mi-hh24:mi\"", errValidation, k))
	}
	return ws, errors
}

func ValidOnceAWeekWindowFormat(v interface{}, k string) (ws []string, errors []error) {
	// valid time format is "ddd:hh24:mi"
	validTimeFormat := "(sun|mon|tue|wed|thu|fri|sat):([0-1][0-9]|2[0-3]):([0-5][0-9])"
	validTimeFormatConsolidated := "^(" + validTimeFormat + "-" + validTimeFormat + "|)$"

	value := strings.ToLower(v.(string))
	if !regexp.MustCompile(validTimeFormatConsolidated).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%w: %q must satisfy the format of \"ddd:hh24:mi-ddd:hh24:mi\"", errValidation, k))
	}
	return ws, errors
}

func ValidStringIsJSONOrYAML(v interface{}, k string) (ws []string, errors []error) {
	if looksLikeJsonString(v) {
		if _, err := structure.NormalizeJsonString(v); err != nil {
			errors = append(errors, fmt.Errorf("%w: %q contains an invalid JSON: %w", errValidation, k, err))
		}
	} else if _, err := checkYAMLString(v); err != nil {
		errors = append(errors, fmt.Errorf("%w: %q contains an invalid YAML: %w", errValidation, k, err))
	}
	return ws, errors
}

// ValidTypeStringNullableBoolean provides custom error messaging for TypeString booleans
// Some arguments require three values: true, false, and "" (unspecified).
// This ValidateFunc returns a custom message since the message with
// validation.StringInSlice([]string{"", "false", "true"}, false) is confusing:
// to be one of [ false true], got 1.
func ValidTypeStringNullableBoolean(v interface{}, k string) (ws []string, es []error) {
	value, ok := v.(string)
	if !ok {
		es = append(es, fmt.Errorf("%w: expected type of %s to be string", errValidation, k))
		return ws, es
	}

	for _, str := range []string{"", "0", "1", "false", "true"} {
		if value == str {
			return ws, es
		}
	}

	es = append(es, fmt.Errorf("%w: expected %s to be one of [\"\", false, true], got %s", errValidation, k, value))
	return ws, es
}

// ValidTypeStringNullableFloat provides custom error messaging for TypeString floats
// Some arguments require a floating point value or an unspecified, empty field.
func ValidTypeStringNullableFloat(v interface{}, k string) (ws []string, es []error) {
	value, ok := v.(string)
	if !ok {
		es = append(es, fmt.Errorf("%w: expected type of %s to be string", errValidation, k))
		return ws, es
	}

	if value == "" {
		return ws, es
	}

	if _, err := strconv.ParseFloat(value, 64); err != nil {
		es = append(es, fmt.Errorf("%w: %s: cannot parse '%s' as float: %w", errValidation, k, value, err))
	}

	return ws, es
}

// ValidUTCTimestamp validates a string in UTC Format required by APIs including:
// https://docs.aws.amazon.com/iot/latest/apireference/API_CloudwatchMetricAction.html
// https://docs.aws.amazon.com/AmazonRDS/latest/APIReference/API_RestoreDBInstanceToPointInTime.html
func ValidUTCTimestamp(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	_, err := time.Parse(time.RFC3339, value)
	if err != nil {
		errors = append(errors, fmt.Errorf("%w: %q must be in RFC3339 time format %q. Example: %w", errValidation, k, time.RFC3339, err))
	}
	return ws, errors
}

var ValidStringDateOrPositiveInt = validation.Any(
	validation.IsRFC3339Time,
	validation.StringMatch(regexp.MustCompile(`^\d+$`), "must be a positive integer value"),
)

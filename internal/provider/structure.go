package provider

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

// Diff returns the elements in `slice1` that aren't in `slice2`.
func Diff(slice1 []string, slice2 []string) []string {
	var diff []string

	for _, s1 := range slice1 {
		if !contains(slice2, s1) {
			diff = append(diff, s1)
		}
	}

	return diff
}

// Takes a slice of pointers to string and returns a slice of strings
func ExpandStringSliceofPointers(input []*string) []string {
	var output []string

	for _, s := range input {
		output = append(output, *s)
	}
	return output
}

// Takes the result of flatmap.Expand for an array of strings and returns a []*string
func ExpandStringList(configured []interface{}) []*string {
	vs := make([]*string, 0, len(configured))
	for _, v := range configured {
		val, ok := v.(string)
		if ok && val != "" {
			vs = append(vs, aws.String(v.(string)))
		}
	}
	return vs
}

// Takes the result of schema.Set of strings and returns a []*string
func ExpandStringSet(configured *schema.Set) []*string {
	return ExpandStringList(configured.List())
}

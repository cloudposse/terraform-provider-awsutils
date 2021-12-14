package flex

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

// Takes the result of flatmap.Expand for an array of strings
// and returns a []*string
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

// Takes list of pointers to strings. Expand to an array
// of raw strings and returns a []interface{}
// to keep compatibility w/ schema.NewSetschema.NewSet
func FlattenStringList(list []*string) []interface{} {
	vs := make([]interface{}, 0, len(list))
	for _, v := range list {
		vs = append(vs, *v)
	}
	return vs
}

// Expands a map of string to interface to a map of string to *string
func ExpandStringMap(m map[string]interface{}) map[string]*string {
	stringMap := make(map[string]*string, len(m))
	for k, v := range m {
		stringMap[k] = aws.String(v.(string))
	}
	return stringMap
}

// Takes a slice of pointers to string and returns a slice of strings
func ExpandStringSliceofPointers(input []*string) []string {
	var output []string

	for _, s := range input {
		output = append(output, *s)
	}
	return output
}

// Takes the result of schema.Set of strings and returns a []*string
func ExpandStringSet(configured *schema.Set) []*string {
	return ExpandStringList(configured.List()) // nosemgrep: helper-schema-Set-extraneous-ExpandStringList-with-List
}

func FlattenStringSet(list []*string) *schema.Set {
	return schema.NewSet(schema.HashString, FlattenStringList(list)) // nosemgrep: helper-schema-Set-extraneous-NewSet-with-FlattenStringList
}

// Takes the result of schema.Set of strings and returns a []*int64
func ExpandInt64Set(configured *schema.Set) []*int64 {
	return ExpandInt64List(configured.List())
}

func FlattenInt64Set(list []*int64) *schema.Set {
	return schema.NewSet(schema.HashInt, FlattenInt64List(list))
}

// Takes the result of flatmap.Expand for an array of int64
// and returns a []*int64
func ExpandInt64List(configured []interface{}) []*int64 {
	vs := make([]*int64, 0, len(configured))
	for _, v := range configured {
		vs = append(vs, aws.Int64(int64(v.(int))))
	}
	return vs
}

// Takes list of pointers to int64s. Expand to an array
// of raw ints and returns a []interface{}
// to keep compatibility w/ schema.NewSet
func FlattenInt64List(list []*int64) []interface{} {
	vs := make([]interface{}, 0, len(list))
	for _, v := range list {
		vs = append(vs, int(aws.Int64Value(v)))
	}
	return vs
}

package provider

import (
	"context"
	"github.com/cloudposse/terraform-provider-awsutils/internal/service/sts"
	"github.com/hashicorp/aws-sdk-go-base/v2/servicemocks"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"io"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/cloudposse/terraform-provider-awsutils/names"
	"github.com/hashicorp/aws-sdk-go-base/v2/mockdata"
)

func suppressLogging() func() {
	// Save the current logger output
	originalOutput := log.Writer()

	// Redirect logs to /dev/null
	log.SetOutput(io.Discard)

	// Return a function to restore the original output
	return func() {
		log.SetOutput(originalOutput)
	}
}

func TestProviderConfig(t *testing.T) {
	// Create provider
	ctx := context.Background()
	p, err := New(ctx)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	oldEnv := servicemocks.InitSessionTestEnv()
	defer servicemocks.PopEnv(oldEnv)

	// Set up STS mock
	closeSts, _, stsEndpoint := mockdata.GetMockedAwsApiSession("STS", []*servicemocks.MockEndpoint{
		servicemocks.MockStsAssumeRoleValidEndpoint,
		servicemocks.MockStsGetCallerIdentityValidEndpoint,
	})
	defer closeSts()

	// Basic provider configuration
	config := map[string]any{
		"region":                      "us-west-2",
		"access_key":                  servicemocks.MockStaticAccessKey,
		"secret_key":                  servicemocks.MockStaticSecretKey,
		"skip_credentials_validation": true,
		"skip_requesting_account_id":  true,
		"endpoints": []any{
			map[string]any{
				"sts": stsEndpoint,
			},
		},
		"assume_role": []any{
			map[string]any{
				"role_arn":     servicemocks.MockStsAssumeRoleArn,
				"session_name": servicemocks.MockStsAssumeRoleSessionName,
			},
		},
	}

	restoreLog := suppressLogging() // Suppress logs
	defer restoreLog()              // Restore logs after test

	// Configure and test the provider
	rc := schema.TestResourceDataRaw(t, p.Schema, config)
	meta, diags := p.ConfigureContextFunc(ctx, rc)
	if diags.HasError() {
		t.Fatalf("Provider configuration failed: %v", diags)
	}
	if meta == nil {
		t.Fatal("Provider meta is nil")
	}

	// Test caller identity data source
	ds := sts.DataSourceCallerIdentity()
	if ds == nil {
		t.Fatal("DataSourceCallerIdentity returned nil")
	}

	rd := schema.TestResourceDataRaw(t, ds.Schema, map[string]interface{}{})
	d := ds.Read(rd, meta)
	if d != nil {
		t.Fatalf("Failed to read caller identity: %v", d)
	}

	// Verify caller identity values
	expectedValues := map[string]string{
		"account_id": servicemocks.MockStsGetCallerIdentityAccountID,
		"arn":        "arn:aws:iam::222222222222:user/Alice", // should be, but is not: servicemocks.MockStsGetCallerIdentityArn,
	}

	for k, expected := range expectedValues {
		if v := rd.Get(k); v != expected {
			t.Errorf("Expected %s to be %q, got %q", k, expected, v)
		}
	}

}

func TestExpandEndpoints(t *testing.T) {
	oldEnv := stashEnv()
	defer popEnv(oldEnv)

	endpoints := make(map[string]interface{})
	for _, serviceKey := range names.Aliases() {
		endpoints[serviceKey] = ""
	}
	endpoints["sts"] = "https://sts.fake.test"

	results := make(map[string]string)

	err := expandEndpoints([]interface{}{endpoints}, results)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 endpoint, got %d", len(results))
	}

	if v := results["sts"]; v != "https://sts.fake.test" {
		t.Errorf("Expected endpoint %q, got %v", "https://sts.fake.test", results)
	}
}

func TestEndpointMultipleKeys(t *testing.T) {
	oldEnv := stashEnv()
	defer popEnv(oldEnv)

	testcases := []struct {
		endpoints        map[string]string
		expectedService  string
		expectedEndpoint string
	}{
		{
			endpoints: map[string]string{
				"transcribe": "https://transcribe.fake.test",
			},
			expectedService:  names.Transcribe,
			expectedEndpoint: "https://transcribe.fake.test",
		},
		{
			endpoints: map[string]string{
				"transcribeservice": "https://transcribe.fake.test",
			},
			expectedService:  names.Transcribe,
			expectedEndpoint: "https://transcribe.fake.test",
		},
		{
			endpoints: map[string]string{
				"transcribe":        "https://transcribe.fake.test",
				"transcribeservice": "https://transcribeservice.fake.test",
			},
			expectedService:  names.Transcribe,
			expectedEndpoint: "https://transcribe.fake.test",
		},
	}

	for _, testcase := range testcases {
		// Reset environment for each test case without defer
		os.Clearenv()

		endpoints := make(map[string]interface{})
		for _, serviceKey := range names.Aliases() {
			endpoints[serviceKey] = ""
		}
		for k, v := range testcase.endpoints {
			endpoints[k] = v
		}

		results := make(map[string]string)

		err := expandEndpoints([]interface{}{endpoints}, results)
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}

		if a, e := len(results), 1; a != e {
			t.Errorf("Expected 1 endpoint, got %d", len(results))
		}

		if v := results[testcase.expectedService]; v != testcase.expectedEndpoint {
			t.Errorf("Expected endpoint[%s] to be %q, got %v", testcase.expectedService, testcase.expectedEndpoint, results)
		}
	}
}

func TestEndpointEnvVarPrecedence(t *testing.T) {
	oldEnv := stashEnv()
	defer popEnv(oldEnv)

	testcases := []struct {
		endpoints        map[string]string
		envvars          map[string]string
		expectedService  string
		expectedEndpoint string
	}{
		{
			endpoints: map[string]string{},
			envvars: map[string]string{
				"TF_AWS_STS_ENDPOINT": "https://sts.fake.test",
			},
			expectedService:  names.STS,
			expectedEndpoint: "https://sts.fake.test",
		},
		{
			endpoints: map[string]string{},
			envvars: map[string]string{
				"AWS_STS_ENDPOINT": "https://sts-deprecated.fake.test",
			},
			expectedService:  names.STS,
			expectedEndpoint: "https://sts-deprecated.fake.test",
		},
		{
			endpoints: map[string]string{},
			envvars: map[string]string{
				"TF_AWS_STS_ENDPOINT": "https://sts.fake.test",
				"AWS_STS_ENDPOINT":    "https://sts-deprecated.fake.test",
			},
			expectedService:  names.STS,
			expectedEndpoint: "https://sts.fake.test",
		},
		{
			endpoints: map[string]string{
				"sts": "https://sts-config.fake.test",
			},
			envvars: map[string]string{
				"TF_AWS_STS_ENDPOINT": "https://sts-env.fake.test",
			},
			expectedService:  names.STS,
			expectedEndpoint: "https://sts-config.fake.test",
		},
	}

	for _, testcase := range testcases {
		// Reset environment for each test case without defer
		os.Clearenv()

		for k, v := range testcase.envvars {
			os.Setenv(k, v)
		}

		endpoints := make(map[string]interface{})
		for _, serviceKey := range names.Aliases() {
			endpoints[serviceKey] = ""
		}
		for k, v := range testcase.endpoints {
			endpoints[k] = v
		}

		results := make(map[string]string)

		err := expandEndpoints([]interface{}{endpoints}, results)
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}

		if a, e := len(results), 1; a != e {
			t.Errorf("Expected 1 endpoint, got %d", len(results))
		}

		if v := results[testcase.expectedService]; v != testcase.expectedEndpoint {
			t.Errorf("Expected endpoint[%s] to be %q, got %v", testcase.expectedService, testcase.expectedEndpoint, results)
		}
	}
}

func stashEnv() []string {
	env := os.Environ()
	os.Clearenv()
	return env
}

func popEnv(env []string) {
	os.Clearenv()

	for _, e := range env {
		p := strings.SplitN(e, "=", 2)
		k, v := p[0], ""
		if len(p) > 1 {
			v = p[1]
		}
		os.Setenv(k, v)
	}
}

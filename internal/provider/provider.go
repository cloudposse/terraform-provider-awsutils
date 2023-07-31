package provider

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/cloudposse/terraform-provider-awsutils/internal/conns"
	"github.com/cloudposse/terraform-provider-awsutils/internal/experimental/nullable"
	"github.com/cloudposse/terraform-provider-awsutils/internal/service/ec2"
	"github.com/cloudposse/terraform-provider-awsutils/internal/service/guardduty"
	"github.com/cloudposse/terraform-provider-awsutils/internal/service/iam"
	"github.com/cloudposse/terraform-provider-awsutils/internal/service/macie2"
	"github.com/cloudposse/terraform-provider-awsutils/internal/service/securityhub"
	"github.com/cloudposse/terraform-provider-awsutils/internal/service/sts"
	tftags "github.com/cloudposse/terraform-provider-awsutils/internal/tags"
	"github.com/cloudposse/terraform-provider-awsutils/internal/verify"
	"github.com/cloudposse/terraform-provider-awsutils/names"
	awsbase "github.com/hashicorp/aws-sdk-go-base/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// New returns a new, initialized Terraform Plugin SDK v2-style provider instance.
// The provider instance is fully configured once the `ConfigureContextFunc` has been called.
func New(_ context.Context) (*schema.Provider, error) {
	// The actual provider
	provider := &schema.Provider{
		// This schema must match exactly the Terraform Protocol v6 (Terraform Plugin Framework) provider's schema.
		// Notably the attributes can have no Default values.
		Schema: map[string]*schema.Schema{
			"access_key": {
				Type:     schema.TypeString,
				Optional: true,
				Description: "The access key for API operations. You can retrieve this\n" +
					"from the 'Security & Credentials' section of the AWS console.",
			},
			"allowed_account_ids": {
				Type:          schema.TypeSet,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{"forbidden_account_ids"},
				Set:           schema.HashString,
			},
			"assume_role":                   assumeRoleSchema(),
			"assume_role_with_web_identity": assumeRoleWithWebIdentitySchema(),
			"custom_ca_bundle": {
				Type:     schema.TypeString,
				Optional: true,
				Description: "File containing custom root and intermediate certificates. " +
					"Can also be configured using the `AWS_CA_BUNDLE` environment variable. " +
					"(Setting `ca_bundle` in the shared config file is not supported.)",
			},
			"default_tags": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Configuration block with settings to default resource tags across all resources.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"tags": {
							Type:        schema.TypeMap,
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "Resource tags to default across all resources",
						},
					},
				},
			},
			"ec2_metadata_service_endpoint": {
				Type:     schema.TypeString,
				Optional: true,
				Description: "Address of the EC2 metadata service endpoint to use. " +
					"Can also be configured using the `AWS_EC2_METADATA_SERVICE_ENDPOINT` environment variable.",
			},
			"ec2_metadata_service_endpoint_mode": {
				Type:     schema.TypeString,
				Optional: true,
				Description: "Protocol to use with EC2 metadata service endpoint." +
					"Valid values are `IPv4` and `IPv6`. Can also be configured using the `AWS_EC2_METADATA_SERVICE_ENDPOINT_MODE` environment variable.",
			},
			"endpoints": endpointsSchema(),
			"forbidden_account_ids": {
				Type:          schema.TypeSet,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{"allowed_account_ids"},
				Set:           schema.HashString,
			},
			"http_proxy": {
				Type:     schema.TypeString,
				Optional: true,
				Description: "The address of an HTTP proxy to use when accessing the AWS API. " +
					"Can also be configured using the `HTTP_PROXY` or `HTTPS_PROXY` environment variables.",
			},
			"ignore_tags": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Configuration block with settings to ignore resource tags across all resources.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"keys": {
							Type:        schema.TypeSet,
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
							Description: "Resource tag keys to ignore across all resources.",
						},
						"key_prefixes": {
							Type:        schema.TypeSet,
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
							Description: "Resource tag key prefixes to ignore across all resources.",
						},
					},
				},
			},
			"insecure": {
				Type:     schema.TypeBool,
				Optional: true,
				Description: "Explicitly allow the provider to perform \"insecure\" SSL requests. If omitted, " +
					"default value is `false`",
			},
			"max_retries": {
				Type:     schema.TypeInt,
				Optional: true,
				Description: "The maximum number of times an AWS API request is\n" +
					"being executed. If the API request still fails, an error is\n" +
					"thrown.",
			},
			"profile": {
				Type:     schema.TypeString,
				Optional: true,
				Description: "The profile for API operations. If not set, the default profile\n" +
					"created with `aws configure` will be used.",
			},
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Description: "The region where AWS operations will take place. Examples\n" +
					"are us-east-1, us-west-2, etc.", // lintignore:AWSAT003,
			},
			"s3_force_path_style": {
				Type:       schema.TypeBool,
				Optional:   true,
				Deprecated: "Use s3_use_path_style instead.",
				Description: "Set this to true to enable the request to use path-style addressing,\n" +
					"i.e., https://s3.amazonaws.com/BUCKET/KEY. By default, the S3 client will\n" +
					"use virtual hosted bucket addressing when possible\n" +
					"(https://BUCKET.s3.amazonaws.com/KEY). Specific to the Amazon S3 service.",
			},
			"s3_use_path_style": {
				Type:     schema.TypeBool,
				Optional: true,
				Description: "Set this to true to enable the request to use path-style addressing,\n" +
					"i.e., https://s3.amazonaws.com/BUCKET/KEY. By default, the S3 client will\n" +
					"use virtual hosted bucket addressing when possible\n" +
					"(https://BUCKET.s3.amazonaws.com/KEY). Specific to the Amazon S3 service.",
			},
			"secret_key": {
				Type:     schema.TypeString,
				Optional: true,
				Description: "The secret key for API operations. You can retrieve this\n" +
					"from the 'Security & Credentials' section of the AWS console.",
			},
			"shared_config_files": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of paths to shared config files. If not set, defaults to [~/.aws/config].",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"shared_credentials_file": {
				Type:          schema.TypeString,
				Optional:      true,
				Deprecated:    "Use shared_credentials_files instead.",
				ConflictsWith: []string{"shared_credentials_files"},
				Description:   "The path to the shared credentials file. If not set, defaults to ~/.aws/credentials.",
			},
			"shared_credentials_files": {
				Type:          schema.TypeList,
				Optional:      true,
				ConflictsWith: []string{"shared_credentials_file"},
				Description:   "List of paths to shared credentials files. If not set, defaults to [~/.aws/credentials].",
				Elem:          &schema.Schema{Type: schema.TypeString},
			},
			"skip_credentials_validation": {
				Type:     schema.TypeBool,
				Optional: true,
				Description: "Skip the credentials validation via STS API. " +
					"Used for AWS API implementations that do not have STS available/implemented.",
			},
			"skip_get_ec2_platforms": {
				Type:     schema.TypeBool,
				Optional: true,
				Description: "Skip getting the supported EC2 platforms. " +
					"Used by users that don't have ec2:DescribeAccountAttributes permissions.",
			},
			"skip_metadata_api_check": {
				Type:         nullable.TypeNullableBool,
				Optional:     true,
				ValidateFunc: nullable.ValidateTypeStringNullableBool,
				Description: "Skip the AWS Metadata API check. " +
					"Used for AWS API implementations that do not have a metadata api endpoint.",
			},
			"skip_region_validation": {
				Type:     schema.TypeBool,
				Optional: true,
				Description: "Skip static validation of region name. " +
					"Used by users of alternative AWS-like APIs or users w/ access to regions that are not public (yet).",
			},
			"skip_requesting_account_id": {
				Type:     schema.TypeBool,
				Optional: true,
				Description: "Skip requesting the account ID. " +
					"Used for AWS API implementations that do not have IAM/STS API and/or metadata API.",
			},
			"sts_region": {
				Type:     schema.TypeString,
				Optional: true,
				Description: "The region where AWS STS operations will take place. Examples\n" +
					"are us-east-1 and us-west-2.", // lintignore:AWSAT003,
			},
			"token": {
				Type:     schema.TypeString,
				Optional: true,
				Description: "session token. A session token is only required if you are\n" +
					"using temporary security credentials.",
			},
			"use_dualstack_endpoint": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Resolve an endpoint with DualStack capability",
			},
			"use_fips_endpoint": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Resolve an endpoint with FIPS capability",
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"awsutils_ec2_client_vpn_export_client_config": ec2.DataSourceEC2ExportClientVpnClientConfiguration(),
			"awsutils_caller_identity":                     sts.DataSourceCallerIdentity(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"awsutils_default_vpc_deletion":               ec2.ResourceDefaultVpcDeletion(),
			"awsutils_expiring_iam_access_key":            iam.ResourceExpiringAccessKey(),
			"awsutils_guardduty_organization_settings":    guardduty.ResourceAwsUtilsGuardDutyOrganizationSettings(),
			"awsutils_macie2_organization_settings":       macie2.ResourceAwsUtilsMacie2OrganizationSettings(),
			"awsutils_security_hub_control_disablement":   securityhub.ResourceSecurityHubControlDisablement(),
			"awsutils_security_hub_organization_settings": securityhub.ResourceSecurityHubOrganizationSettings(),
		},
	}

	provider.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		terraformVersion := provider.TerraformVersion
		if terraformVersion == "" {
			// Terraform 0.12 introduced this field to the protocol
			// We can therefore assume that if it's missing it's 0.10 or 0.11
			terraformVersion = "0.11+compatible"
		}
		return providerConfigure(ctx, d, terraformVersion)
	}

	return provider, nil
}

func providerConfigure(ctx context.Context, d *schema.ResourceData, terraformVersion string) (interface{}, diag.Diagnostics) {
	config := conns.Config{
		AccessKey:                      d.Get("access_key").(string),
		DefaultTagsConfig:              expandProviderDefaultTags(d.Get("default_tags").([]interface{})),
		CustomCABundle:                 d.Get("custom_ca_bundle").(string),
		EC2MetadataServiceEndpoint:     d.Get("ec2_metadata_service_endpoint").(string),
		EC2MetadataServiceEndpointMode: d.Get("ec2_metadata_service_endpoint_mode").(string),
		Endpoints:                      make(map[string]string),
		HTTPProxy:                      d.Get("http_proxy").(string),
		IgnoreTagsConfig:               expandProviderIgnoreTags(d.Get("ignore_tags").([]interface{})),
		Insecure:                       d.Get("insecure").(bool),
		MaxRetries:                     25, // Set default here, not in schema (muxing with v6 provider).
		Profile:                        d.Get("profile").(string),
		Region:                         d.Get("region").(string),
		S3UsePathStyle:                 d.Get("s3_use_path_style").(bool) || d.Get("s3_force_path_style").(bool),
		SecretKey:                      d.Get("secret_key").(string),
		SkipCredsValidation:            d.Get("skip_credentials_validation").(bool),
		SkipGetEC2Platforms:            d.Get("skip_get_ec2_platforms").(bool),
		SkipRegionValidation:           d.Get("skip_region_validation").(bool),
		SkipRequestingAccountId:        d.Get("skip_requesting_account_id").(bool),
		STSRegion:                      d.Get("sts_region").(string),
		TerraformVersion:               terraformVersion,
		Token:                          d.Get("token").(string),
		UseDualStackEndpoint:           d.Get("use_dualstack_endpoint").(bool),
		UseFIPSEndpoint:                d.Get("use_fips_endpoint").(bool),
	}

	if v, ok := d.GetOk("max_retries"); ok {
		config.MaxRetries = v.(int)
	}

	if raw := d.Get("shared_config_files").([]interface{}); len(raw) != 0 {
		l := make([]string, len(raw))
		for i, v := range raw {
			l[i] = v.(string)
		}
		config.SharedConfigFiles = l
	}

	if v := d.Get("shared_credentials_file").(string); v != "" {
		config.SharedCredentialsFiles = []string{v}
	}

	if raw := d.Get("shared_credentials_files").([]interface{}); len(raw) != 0 {
		l := make([]string, len(raw))
		for i, v := range raw {
			l[i] = v.(string)
		}
		config.SharedCredentialsFiles = l
	}

	if l, ok := d.Get("assume_role").([]interface{}); ok && len(l) > 0 && l[0] != nil {
		config.AssumeRole = expandAssumeRole(l[0].(map[string]interface{}))
		log.Printf("[INFO] assume_role configuration set: (ARN: %q, SessionID: %q, ExternalID: %q)", config.AssumeRole.RoleARN, config.AssumeRole.SessionName, config.AssumeRole.ExternalID)
	}

	if l, ok := d.Get("assume_role_with_web_identity").([]interface{}); ok && len(l) > 0 && l[0] != nil {
		config.AssumeRoleWithWebIdentity = expandAssumeRoleWithWebIdentity(l[0].(map[string]interface{}))
		log.Printf("[INFO] assume_role_with_web_identity configuration set: (ARN: %q, SessionID: %q)", config.AssumeRoleWithWebIdentity.RoleARN, config.AssumeRoleWithWebIdentity.SessionName)
	}

	if err := expandEndpoints(d.Get("endpoints").(*schema.Set).List(), config.Endpoints); err != nil {
		return nil, diag.FromErr(err)
	}

	if v, ok := d.GetOk("allowed_account_ids"); ok {
		for _, accountIDRaw := range v.(*schema.Set).List() {
			config.AllowedAccountIds = append(config.AllowedAccountIds, accountIDRaw.(string))
		}
	}

	if v, ok := d.GetOk("forbidden_account_ids"); ok {
		for _, accountIDRaw := range v.(*schema.Set).List() {
			config.ForbiddenAccountIds = append(config.ForbiddenAccountIds, accountIDRaw.(string))
		}
	}

	if v, null, _ := nullable.Bool(d.Get("skip_metadata_api_check").(string)).Value(); !null {
		if v {
			config.EC2MetadataServiceEnableState = imds.ClientDisabled
		} else {
			config.EC2MetadataServiceEnableState = imds.ClientEnabled
		}
	}

	return config.Client(ctx)
}

func assumeRoleSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"duration": {
					Type:          schema.TypeString,
					Optional:      true,
					Description:   "The duration, between 15 minutes and 12 hours, of the role session. Valid time units are ns, us (or µs), ms, s, h, or m.",
					ValidateFunc:  validAssumeRoleDuration,
					ConflictsWith: []string{"assume_role.0.duration_seconds"},
				},
				"duration_seconds": {
					Type:          schema.TypeInt,
					Optional:      true,
					Deprecated:    "Use assume_role.duration instead",
					Description:   "The duration, in seconds, of the role session.",
					ValidateFunc:  validation.IntBetween(900, 43200),
					ConflictsWith: []string{"assume_role.0.duration"},
				},
				"external_id": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "A unique identifier that might be required when you assume a role in another account.",
					ValidateFunc: validation.All(
						validation.StringLenBetween(2, 1224),
						validation.StringMatch(regexp.MustCompile(`[\w+=,.@:\/\-]*`), ""),
					),
				},
				"policy": {
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "IAM Policy JSON describing further restricting permissions for the IAM Role being assumed.",
					ValidateFunc: validation.StringIsJSON,
				},
				"policy_arns": {
					Type:        schema.TypeSet,
					Optional:    true,
					Description: "Amazon Resource Names (ARNs) of IAM Policies describing further restricting permissions for the IAM Role being assumed.",
					Elem: &schema.Schema{
						Type:         schema.TypeString,
						ValidateFunc: verify.ValidARN,
					},
				},
				"role_arn": {
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "Amazon Resource Name (ARN) of an IAM Role to assume prior to making API calls.",
					ValidateFunc: verify.ValidARN,
				},
				"session_name": {
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "An identifier for the assumed role session.",
					ValidateFunc: validAssumeRoleSessionName,
				},
				"tags": {
					Type:        schema.TypeMap,
					Optional:    true,
					Description: "Assume role session tags.",
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
				"transitive_tag_keys": {
					Type:        schema.TypeSet,
					Optional:    true,
					Description: "Assume role session tag keys to pass to any subsequent sessions.",
					Elem:        &schema.Schema{Type: schema.TypeString},
				},
			},
		},
	}
}

func assumeRoleWithWebIdentitySchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"duration": {
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "The duration, between 15 minutes and 12 hours, of the role session. Valid time units are ns, us (or µs), ms, s, h, or m.",
					ValidateFunc: validAssumeRoleDuration,
				},
				"policy": {
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "IAM Policy JSON describing further restricting permissions for the IAM Role being assumed.",
					ValidateFunc: validation.StringIsJSON,
				},
				"policy_arns": {
					Type:        schema.TypeSet,
					Optional:    true,
					Description: "Amazon Resource Names (ARNs) of IAM Policies describing further restricting permissions for the IAM Role being assumed.",
					Elem: &schema.Schema{
						Type:         schema.TypeString,
						ValidateFunc: verify.ValidARN,
					},
				},
				"role_arn": {
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "Amazon Resource Name (ARN) of an IAM Role to assume prior to making API calls.",
					ValidateFunc: verify.ValidARN,
				},
				"session_name": {
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "An identifier for the assumed role session.",
					ValidateFunc: validAssumeRoleSessionName,
				},
				"web_identity_token": {
					Type:         schema.TypeString,
					Optional:     true,
					ValidateFunc: validation.StringLenBetween(4, 20000),
					ExactlyOneOf: []string{"assume_role_with_web_identity.0.web_identity_token", "assume_role_with_web_identity.0.web_identity_token_file"},
				},
				"web_identity_token_file": {
					Type:         schema.TypeString,
					Optional:     true,
					ExactlyOneOf: []string{"assume_role_with_web_identity.0.web_identity_token", "assume_role_with_web_identity.0.web_identity_token_file"},
				},
			},
		},
	}
}

func endpointsSchema() *schema.Schema {
	endpointsAttributes := make(map[string]*schema.Schema)

	for _, serviceKey := range names.Aliases() {
		endpointsAttributes[serviceKey] = &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Default:     "",
			Description: "Use this to override the default service endpoint URL",
		}
	}

	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: endpointsAttributes,
		},
	}
}

func expandAssumeRole(m map[string]interface{}) *awsbase.AssumeRole {
	assumeRole := awsbase.AssumeRole{}

	if v, ok := m["duration"].(string); ok && v != "" {
		duration, _ := time.ParseDuration(v)
		assumeRole.Duration = duration
	}

	if v, ok := m["duration_seconds"].(int); ok && v != 0 {
		assumeRole.Duration = time.Duration(v) * time.Second
	}

	if v, ok := m["external_id"].(string); ok && v != "" {
		assumeRole.ExternalID = v
	}

	if v, ok := m["policy"].(string); ok && v != "" {
		assumeRole.Policy = v
	}

	if policyARNSet, ok := m["policy_arns"].(*schema.Set); ok && policyARNSet.Len() > 0 {
		for _, policyARNRaw := range policyARNSet.List() {
			policyARN, ok := policyARNRaw.(string)

			if !ok {
				continue
			}

			assumeRole.PolicyARNs = append(assumeRole.PolicyARNs, policyARN)
		}
	}

	if v, ok := m["role_arn"].(string); ok && v != "" {
		assumeRole.RoleARN = v
	}

	if v, ok := m["session_name"].(string); ok && v != "" {
		assumeRole.SessionName = v
	}

	if tagMapRaw, ok := m["tags"].(map[string]interface{}); ok && len(tagMapRaw) > 0 {
		assumeRole.Tags = make(map[string]string)

		for k, vRaw := range tagMapRaw {
			v, ok := vRaw.(string)

			if !ok {
				continue
			}

			assumeRole.Tags[k] = v
		}
	}

	if transitiveTagKeySet, ok := m["transitive_tag_keys"].(*schema.Set); ok && transitiveTagKeySet.Len() > 0 {
		for _, transitiveTagKeyRaw := range transitiveTagKeySet.List() {
			transitiveTagKey, ok := transitiveTagKeyRaw.(string)

			if !ok {
				continue
			}

			assumeRole.TransitiveTagKeys = append(assumeRole.TransitiveTagKeys, transitiveTagKey)
		}
	}

	return &assumeRole
}

func expandAssumeRoleWithWebIdentity(m map[string]interface{}) *awsbase.AssumeRoleWithWebIdentity {
	assumeRole := awsbase.AssumeRoleWithWebIdentity{}

	if v, ok := m["duration"].(string); ok && v != "" {
		duration, _ := time.ParseDuration(v)
		assumeRole.Duration = duration
	}

	if v, ok := m["duration_seconds"].(int); ok && v != 0 {
		assumeRole.Duration = time.Duration(v) * time.Second
	}

	if v, ok := m["policy"].(string); ok && v != "" {
		assumeRole.Policy = v
	}

	if policyARNSet, ok := m["policy_arns"].(*schema.Set); ok && policyARNSet.Len() > 0 {
		for _, policyARNRaw := range policyARNSet.List() {
			policyARN, ok := policyARNRaw.(string)

			if !ok {
				continue
			}

			assumeRole.PolicyARNs = append(assumeRole.PolicyARNs, policyARN)
		}
	}

	if v, ok := m["role_arn"].(string); ok && v != "" {
		assumeRole.RoleARN = v
	}

	if v, ok := m["session_name"].(string); ok && v != "" {
		assumeRole.SessionName = v
	}

	if v, ok := m["web_identity_token"].(string); ok && v != "" {
		assumeRole.WebIdentityToken = v
	}

	if v, ok := m["web_identity_token_file"].(string); ok && v != "" {
		assumeRole.WebIdentityTokenFile = v
	}

	return &assumeRole
}

func expandProviderDefaultTags(l []interface{}) *tftags.DefaultConfig {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	defaultConfig := &tftags.DefaultConfig{}
	m := l[0].(map[string]interface{})

	if v, ok := m["tags"].(map[string]interface{}); ok {
		defaultConfig.Tags = tftags.New(v)
	}
	return defaultConfig
}

func expandProviderIgnoreTags(l []interface{}) *tftags.IgnoreConfig {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	ignoreConfig := &tftags.IgnoreConfig{}
	m := l[0].(map[string]interface{})

	if v, ok := m["keys"].(*schema.Set); ok {
		ignoreConfig.Keys = tftags.New(v.List())
	}

	if v, ok := m["key_prefixes"].(*schema.Set); ok {
		ignoreConfig.KeyPrefixes = tftags.New(v.List())
	}

	return ignoreConfig
}

func expandEndpoints(endpointsSetList []interface{}, out map[string]string) error {
	for _, endpointsSetI := range endpointsSetList {
		endpoints := endpointsSetI.(map[string]interface{})

		for _, hclKey := range names.Aliases() {
			var serviceKey string
			var err error
			if serviceKey, err = names.ProviderPackageForAlias(hclKey); err != nil {
				return fmt.Errorf("failed to assign endpoint (%s): %w", hclKey, err)
			}

			if out[serviceKey] == "" && endpoints[hclKey].(string) != "" {
				out[serviceKey] = endpoints[hclKey].(string)
			}
		}
	}

	for _, service := range names.ProviderPackages() {
		if out[service] != "" {
			continue
		}

		envvar := names.EnvVar(service)
		if envvar != "" {
			if v := os.Getenv(envvar); v != "" {
				out[service] = v
				continue
			}
		}
		if envvarDeprecated := names.DeprecatedEnvVar(service); envvarDeprecated != "" {
			if v := os.Getenv(envvarDeprecated); v != "" {
				log.Printf("[WARN] The environment variable %q is deprecated. Use %q instead.", envvarDeprecated, envvar)
				out[service] = v
			}
		}
	}

	return nil
}

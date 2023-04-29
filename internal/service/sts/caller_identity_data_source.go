package sts

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/cloudposse/terraform-provider-awsutils/internal/conns"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceCallerIdentity() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceCallerIdentityRead,

		Schema: map[string]*schema.Schema{
			"account_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"eks_role_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"user_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

// Take the ARN output of sts get caller identity and return the role
// example: arn:aws:sts::123456789012:assumed-role/role-name/role-session-name
// returns: arn:aws:iam::123456789012:role/role-name
func getIamRoleArnFromCallerRoleArn(sessionRole string) string {
	parts := strings.Split(sessionRole, "/")
	if len(parts) != 3 {
		return sessionRole
	}

	arnParts := strings.Split(parts[0], ":")
	if len(arnParts) != 6 {
		return sessionRole
	}

	accountId := arnParts[4]
	partition := arnParts[1]
	roleName := parts[1]

	return fmt.Sprintf("arn:%s:iam::%s:role/%s", partition, accountId, roleName)
}

func dataSourceCallerIdentityRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*conns.AWSClient).STSConn

	log.Printf("[DEBUG] Reading Caller Identity")
	res, err := client.GetCallerIdentity(&sts.GetCallerIdentityInput{})

	if err != nil {
		return fmt.Errorf("getting Caller Identity: %w", err)
	}

	log.Printf("[DEBUG] Received Caller Identity: %s", res)

	d.SetId(aws.StringValue(res.Account))
	d.Set("account_id", res.Account)
	d.Set("arn", res.Arn)
	d.Set("user_id", res.UserId)

	// If the caller identity is an assumed role, get the IAM role ARN and set it as the ARN
	if strings.HasPrefix(aws.StringValue(res.Arn), "arn:aws:sts") {
		d.Set("eks_role_arn", getIamRoleArnFromCallerRoleArn(aws.StringValue(res.Arn)))
	}

	return nil
}

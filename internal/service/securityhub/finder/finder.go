package finder

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/securityhub"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func AdminAccount(conn *securityhub.SecurityHub, adminAccountID string) (*securityhub.AdminAccount, error) {
	input := &securityhub.ListOrganizationAdminAccountsInput{}
	var result *securityhub.AdminAccount

	err := conn.ListOrganizationAdminAccountsPages(input, func(page *securityhub.ListOrganizationAdminAccountsOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, adminAccount := range page.AdminAccounts {
			if adminAccount == nil {
				continue
			}

			if aws.StringValue(adminAccount.AccountId) == adminAccountID {
				result = adminAccount
				return false
			}
		}

		return !lastPage
	})

	return result, err
}

func SecurityHubControl(conn *securityhub.SecurityHub, controlArn string) (*securityhub.StandardsControl, error) {
	standardsInput := &securityhub.GetEnabledStandardsInput{}
	standards, err := conn.GetEnabledStandards(standardsInput)
	if err != nil {
		return nil, err
	}

	for _, s := range standards.StandardsSubscriptions {
		input := &securityhub.DescribeStandardsControlsInput{
			StandardsSubscriptionArn: s.StandardsSubscriptionArn,
		}

		controls, err := conn.DescribeStandardsControls(input)

		for _, c := range controls.Controls {
			if *c.StandardsControlArn == controlArn {
				return c, nil
			}
		}

		if err != nil {
			return nil, err
		}
	}

	return nil, &resource.NotFoundError{
		Message: fmt.Sprintf("%s is not a valid control arn", controlArn),
	}
}

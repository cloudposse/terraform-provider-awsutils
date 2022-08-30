package securityhub

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/securityhub"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func FindAdminAccount(conn *securityhub.SecurityHub, adminAccountID string) (*securityhub.AdminAccount, error) {
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

func FindSecurityHubControl(conn *securityhub.SecurityHub, controlArn string) (*securityhub.StandardsControl, error) {
	standardsInput := &securityhub.GetEnabledStandardsInput{}
	standards, err := conn.GetEnabledStandards(standardsInput)
	if err != nil {
		return nil, err
	}

	for _, s := range standards.StandardsSubscriptions {
		input := &securityhub.DescribeStandardsControlsInput{
			StandardsSubscriptionArn: s.StandardsSubscriptionArn,
		}

		var foundControl *securityhub.StandardsControl
		err := conn.DescribeStandardsControlsPages(input, func(page *securityhub.DescribeStandardsControlsOutput, lastPage bool) bool {
			for _, c := range page.Controls {
				if *c.StandardsControlArn == controlArn {
					foundControl = c
					return false
				}
			}
			return !lastPage
		})

		if err != nil {
			return nil, err
		}

		if foundControl != nil {
			return foundControl, nil
		}
	}

	return nil, &resource.NotFoundError{
		Message: fmt.Sprintf("Could not find a control with arn %s", controlArn),
	}
}

func IsSecurityHubOrganizationSettingsAutoEnabled(conn *securityhub.SecurityHub) (bool, error) {
	input := &securityhub.DescribeOrganizationConfigurationInput{}
	settings, err := conn.DescribeOrganizationConfiguration(input)
	if err != nil {
		return false, err
	}

	return *settings.AutoEnable, err
}

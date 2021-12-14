package guardduty

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/guardduty"
)

func FindAdminAccount(conn *guardduty.GuardDuty, adminAccountID string) (*guardduty.AdminAccount, error) {
	input := &guardduty.ListOrganizationAdminAccountsInput{}
	var result *guardduty.AdminAccount

	err := conn.ListOrganizationAdminAccountsPages(input, func(page *guardduty.ListOrganizationAdminAccountsOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, adminAccount := range page.AdminAccounts {
			if adminAccount == nil {
				continue
			}

			if aws.StringValue(adminAccount.AdminAccountId) == adminAccountID {
				result = adminAccount
				return false
			}
		}

		return !lastPage
	})

	return result, err
}

func IsGuardDutyOrganizationSettingsAutoEnabled(conn *guardduty.GuardDuty, detectorID string) (bool, error) {
	input := &guardduty.DescribeOrganizationConfigurationInput{DetectorId: &detectorID}
	settings, err := conn.DescribeOrganizationConfiguration(input)
	if err != nil {
		return false, err
	}

	return *settings.AutoEnable, err
}

package macie2

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/macie2"
)

func FindAdminAccount(conn *macie2.Macie2, adminAccountID string) (*macie2.AdminAccount, error) {
	input := &macie2.ListOrganizationAdminAccountsInput{}
	var result *macie2.AdminAccount

	err := conn.ListOrganizationAdminAccountsPages(input, func(page *macie2.ListOrganizationAdminAccountsOutput, lastPage bool) bool {
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

func IsMacie2OrganizationSettingsAutoEnabled(conn *macie2.Macie2) (bool, error) {
	input := &macie2.DescribeOrganizationConfigurationInput{}
	settings, err := conn.DescribeOrganizationConfiguration(input)
	if err != nil {
		return false, err
	}

	return *settings.AutoEnable, err
}

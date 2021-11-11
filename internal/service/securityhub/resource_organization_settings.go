package securityhub

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/securityhub"
	"github.com/cloudposse/terraform-provider-awsutils/internal/conns"
	"github.com/cloudposse/terraform-provider-awsutils/internal/flex"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceSecurityHubOrganizationSettings() *schema.Resource {
	return &schema.Resource{
		Description: `Enables a list of accounts as Security Hub member accounts in an existing AWS Organization.

Designating an account as the Security Hub Administrator account in an AWS Organization can optionally enable all 
newly created accounts and accounts that join the organization after the setting is enabled, however it does not 
enable existing accounts. Use this resource to enable a list of existing accounts`,
		Create:        resourceAwsSecurityHubOrganizationSettingsCreate,
		Read:          resourceAwsSecurityHubOrganizationSettingsRead,
		Update:        resourceAwsSecurityHubOrganizationSettingsUpdate,
		Delete:        resourceAwsSecurityHubOrganizationSettingsDelete,
		SchemaVersion: 1,
		Schema: map[string]*schema.Schema{
			"member_accounts": {
				Description: "A list of AWS Organization member accounts to associate with the Security Hub Administrator account.",
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
				Required:    true,
			},
			"auto_enable_new_accounts": {
				Description: "A flag to indicate if the automatic enablement setting, should be enabled. If enabled, Security Hub begins to enable new accounts as they are added to the organization",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
		},
	}
}

func resourceAwsSecurityHubOrganizationSettingsCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).SecurityHubConn
	memberAccounts := flex.ExpandStringSliceofPointers(flex.ExpandStringSet(d.Get("member_accounts").(*schema.Set)))
	autoEnable := d.Get("auto_enable_new_accounts").(bool)

	if err := addSecurityHubOrganizationMembers(conn, memberAccounts); err != nil {
		return err
	}

	if err := updateSecurityHubOrganizationSettings(conn, autoEnable); err != nil {
		return err
	}

	d.SetId(uuid.New().String())

	return resourceAwsSecurityHubOrganizationSettingsRead(d, meta)
}

func resourceAwsSecurityHubOrganizationSettingsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).SecurityHubConn

	enabled, err := IsSecurityHubOrganizationSettingsAutoEnabled(conn)
	if err != nil {
		return fmt.Errorf("error reading security hub organization settings: %s", err)
	}

	d.Set("auto_enable_new_accounts", enabled)

	return nil
}

func resourceAwsSecurityHubOrganizationSettingsUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).SecurityHubConn
	if d.HasChanges("auto_enable_new_accounts") {
		_, new := d.GetChange("auto_enable_new_accounts")
		autoEnable := new.(bool)

		if err := updateSecurityHubOrganizationSettings(conn, autoEnable); err != nil {
			return fmt.Errorf("error updating security hub organization settings: %s", err)
		}
	}

	if d.HasChange("member_accounts") {
		old, new := d.GetChange("member_accounts")

		oldExpanded := flex.ExpandStringSliceofPointers(flex.ExpandStringSet(old.(*schema.Set)))
		newExpanded := flex.ExpandStringSliceofPointers(flex.ExpandStringSet(new.(*schema.Set)))

		membersToAdd := flex.Diff(newExpanded, oldExpanded)
		if len(membersToAdd) > 0 {
			if err := addSecurityHubOrganizationMembers(conn, membersToAdd); err != nil {
				return fmt.Errorf("error setting security hub organization members: %s", err)
			}
		}

		membersToRemove := flex.Diff(oldExpanded, newExpanded)
		if len(membersToRemove) > 0 {
			if err := removeSecurityHubOrganizationMembers(conn, membersToRemove); err != nil {
				return fmt.Errorf("error removing security hub organization members: %s", err)
			}
		}
	}
	return nil
}

func resourceAwsSecurityHubOrganizationSettingsDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func updateSecurityHubOrganizationSettings(conn *securityhub.SecurityHub, autoEnable bool) error {
	updateOrgSettingsInput := &securityhub.UpdateOrganizationConfigurationInput{
		AutoEnable: aws.Bool(autoEnable),
	}
	if _, err := conn.UpdateOrganizationConfiguration(updateOrgSettingsInput); err != nil {
		return fmt.Errorf("error updating security hub administrator account settings: %s", err)
	}
	return nil
}

func makeAccountDetails(accounts []string) []*securityhub.AccountDetails {
	accountDetails := make([]*securityhub.AccountDetails, 0)
	for i := range accounts {
		accountDetails = append(accountDetails, &securityhub.AccountDetails{AccountId: aws.String(accounts[i])})
	}
	return accountDetails
}

func makeAccountIDs(accounts []string) []*string {
	accountIDs := make([]*string, 0)
	for i := range accounts {
		accountIDs = append(accountIDs, aws.String(accounts[i]))
	}
	return accountIDs
}

func addSecurityHubOrganizationMembers(conn *securityhub.SecurityHub, memberAccounts []string) error {
	if len(memberAccounts) > 0 {
		accountDetails := makeAccountDetails(memberAccounts)

		createMembersInput := &securityhub.CreateMembersInput{
			AccountDetails: accountDetails,
		}

		if result, err := conn.CreateMembers(createMembersInput); err != nil || len(result.UnprocessedAccounts) > 0 {
			if err != nil {
				return fmt.Errorf("error designating security hub administrator account members: %s", err)
			}
			return fmt.Errorf("error designating security hub administrator account members: %s", result.UnprocessedAccounts)
		}
	}
	return nil
}

func removeSecurityHubOrganizationMembers(conn *securityhub.SecurityHub, memberAccounts []string) error {
	accountIDs := makeAccountIDs(memberAccounts)
	if len(memberAccounts) > 0 {
		disassociateMembersInput := &securityhub.DisassociateMembersInput{
			AccountIds: accountIDs,
		}

		deleteMembersInput := &securityhub.DeleteMembersInput{
			AccountIds: accountIDs,
		}

		if _, err := conn.DisassociateMembers(disassociateMembersInput); err != nil {
			if err != nil {
				return fmt.Errorf("error disassociating security hub administrator account members: %s", err)
			}
		}

		if result, err := conn.DeleteMembers(deleteMembersInput); err != nil || len(result.UnprocessedAccounts) > 0 {
			if err != nil {
				return fmt.Errorf("error removing security hub administrator account members: %s", err)
			}
			return fmt.Errorf("error removing security hub administrator account members: %s", result.UnprocessedAccounts)
		}
	}
	return nil
}

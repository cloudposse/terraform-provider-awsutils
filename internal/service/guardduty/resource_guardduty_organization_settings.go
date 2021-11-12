package guardduty

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/guardduty"
	"github.com/cloudposse/terraform-provider-awsutils/internal/conns"
	"github.com/cloudposse/terraform-provider-awsutils/internal/flex"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceAwsUtilsGuardDutyOrganizationSettings() *schema.Resource {
	return &schema.Resource{
		Description: `Enables a list of accounts as GuardDuty member accounts in an existing AWS Organization.

Designating an account as the GuardDuty Administrator account in an AWS Organization can optionally enable all 
newly created accounts and accounts that join the organization after the setting is enabled, however it does not 
enable existing accounts. Use this resource to enable a list of existing accounts`,
		Create:        resourceAwsGuardDutyOrganizationSettingsCreate,
		Read:          resourceAwsGuardDutyOrganizationSettingsRead,
		Update:        resourceAwsGuardDutyOrganizationSettingsUpdate,
		Delete:        resourceAwsGuardDutyOrganizationSettingsDelete,
		SchemaVersion: 1,
		Schema: map[string]*schema.Schema{
			"id": {
				Description: "The ID of this resource.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"member_accounts": {
				Description: "A list of AWS Organization member accounts to associate with the GuardDuty Administrator account.",
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
				Required:    true,
			},
			"detector_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},
		},
	}
}

func getMemberAccounts(d *schema.ResourceData) []string {
	memberAccounts := flex.ExpandStringSliceofPointers(flex.ExpandStringSet(d.Get("member_accounts").(*schema.Set)))
	return memberAccounts
}

func resourceAwsGuardDutyOrganizationSettingsCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).GuardDutyConn
	memberAccounts := getMemberAccounts(d)
	detectorID := d.Get("detector_id").(string)

	if err := addGuardDutyOrganizationMembers(conn, detectorID, memberAccounts); err != nil {
		return err
	}

	d.SetId(uuid.New().String())

	return resourceAwsGuardDutyOrganizationSettingsRead(d, meta)
}

func resourceAwsGuardDutyOrganizationSettingsRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsGuardDutyOrganizationSettingsUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).GuardDutyConn
	detectorID := d.Get("detector_id").(string)

	if d.HasChange("member_accounts") {
		old, new := d.GetChange("member_accounts")

		oldExpanded := flex.ExpandStringSliceofPointers(flex.ExpandStringSet(old.(*schema.Set)))
		newExpanded := flex.ExpandStringSliceofPointers(flex.ExpandStringSet(new.(*schema.Set)))

		membersToAdd := flex.Diff(newExpanded, oldExpanded)
		if len(membersToAdd) > 0 {
			if err := addGuardDutyOrganizationMembers(conn, detectorID, membersToAdd); err != nil {
				return fmt.Errorf("error setting guardduty organization members: %s", err)
			}
		}

		membersToRemove := flex.Diff(oldExpanded, newExpanded)
		if len(membersToRemove) > 0 {
			if err := removeGuardDutyOrganizationMembers(conn, detectorID, membersToRemove); err != nil {
				return fmt.Errorf("error removing guardduty organization members: %s", err)
			}
		}
	}
	return nil
}

func resourceAwsGuardDutyOrganizationSettingsDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).GuardDutyConn
	detectorID := d.Get("detector_id").(string)
	membersToRemove := getMemberAccounts(d)

	if err := removeGuardDutyOrganizationMembers(conn, detectorID, membersToRemove); err != nil {
		return fmt.Errorf("error removing guardduty organization members: %s", err)
	}
	return nil
}

func makeGuardDutyAccountDetails(accounts []string) []*guardduty.AccountDetail {
	accountDetails := make([]*guardduty.AccountDetail, 0)
	for i := range accounts {
		accountDetails = append(accountDetails, &guardduty.AccountDetail{
			AccountId: aws.String(accounts[i]),
			Email:     aws.String("notused2join@awsorganization.com"),
		})
	}
	return accountDetails
}

func makeGuardDutyAccountIDs(accounts []string) []*string {
	accountIDs := make([]*string, 0)
	for i := range accounts {
		accountIDs = append(accountIDs, aws.String(accounts[i]))
	}
	return accountIDs
}

func addGuardDutyOrganizationMembers(conn *guardduty.GuardDuty, detectorID string, memberAccounts []string) error {
	if len(memberAccounts) > 0 {
		accountDetails := makeGuardDutyAccountDetails(memberAccounts)

		createMembersInput := &guardduty.CreateMembersInput{
			AccountDetails: accountDetails,
			DetectorId:     &detectorID,
		}

		if result, err := conn.CreateMembers(createMembersInput); err != nil || len(result.UnprocessedAccounts) > 0 {
			if err != nil {
				return fmt.Errorf("error designating guardduty administrator account members: %s", err)
			}
			return fmt.Errorf("error designating guardduty administrator account members: %s", result.UnprocessedAccounts)
		}
	}
	return nil
}

func removeGuardDutyOrganizationMembers(conn *guardduty.GuardDuty, detectorID string, memberAccounts []string) error {
	accountIDs := makeGuardDutyAccountIDs(memberAccounts)
	if len(memberAccounts) > 0 {
		disassociateMembersInput := &guardduty.DisassociateMembersInput{
			AccountIds: accountIDs,
			DetectorId: &detectorID,
		}

		deleteMembersInput := &guardduty.DeleteMembersInput{
			AccountIds: accountIDs,
			DetectorId: &detectorID,
		}

		if _, err := conn.DisassociateMembers(disassociateMembersInput); err != nil {
			if err != nil {
				return fmt.Errorf("error disassociating guardduty administrator account members: %s", err)
			}
		}

		if result, err := conn.DeleteMembers(deleteMembersInput); err != nil || len(result.UnprocessedAccounts) > 0 {
			if err != nil {
				return fmt.Errorf("error removing guardduty administrator account members: %s", err)
			}
			return fmt.Errorf("error removing guardduty administrator account members: %s", result.UnprocessedAccounts)
		}
	}
	return nil
}

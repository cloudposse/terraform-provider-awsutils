package organizations

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/cloudposse/terraform-provider-awsutils/internal/conns"
	"github.com/cloudposse/terraform-provider-awsutils/internal/verify"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var errInvalidDelegatedAdminID = errors.New("expected ID in the form of account_id/service_principal")

func ResourceDelegatedAdministrator() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceDelegatedAdministratorCreate,
		ReadWithoutTimeout:   resourceDelegatedAdministratorRead,
		DeleteWithoutTimeout: resourceDelegatedAdministratorDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"account_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: verify.ValidAccountID,
			},
			"service_principal": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringLenBetween(1, 128),
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"delegation_enabled_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"email": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"joined_method": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"joined_timestamp": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceDelegatedAdministratorCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).OrganizationsConn

	accountID := d.Get("account_id").(string)
	servicePrincipal := d.Get("service_principal").(string)
	input := &organizations.RegisterDelegatedAdministratorInput{
		AccountId:        aws.String(accountID),
		ServicePrincipal: aws.String(servicePrincipal),
	}

	_, err := conn.RegisterDelegatedAdministratorWithContext(ctx, input)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error creating Organizations DelegatedAdministrator (%s): %w", accountID, err))
	}

	d.SetId(fmt.Sprintf("%s/%s", accountID, servicePrincipal))

	return resourceDelegatedAdministratorRead(ctx, d, meta)
}

func resourceDelegatedAdministratorRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).OrganizationsConn

	accountID, servicePrincipal, err := DecodeOrganizationDelegatedAdministratorID(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("error decoding ID AWS Organization (%s) DelegatedAdministrators: %w", d.Id(), err))
	}
	input := &organizations.ListDelegatedAdministratorsInput{
		ServicePrincipal: aws.String(servicePrincipal),
	}
	var delegatedAccount *organizations.DelegatedAdministrator
	err = conn.ListDelegatedAdministratorsPagesWithContext(ctx, input, func(page *organizations.ListDelegatedAdministratorsOutput, lastPage bool) bool {
		for _, delegated := range page.DelegatedAdministrators {
			if aws.StringValue(delegated.Id) == accountID {
				delegatedAccount = delegated
			}
		}

		return !lastPage
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("error listing AWS Organization (%s) DelegatedAdministrators: %w", d.Id(), err))
	}

	if delegatedAccount == nil {
		log.Printf("[WARN] AWS Organization DelegatedAdministrators not found (%s), removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err := d.Set("arn", delegatedAccount.Arn); err != nil {
		return diag.FromErr(fmt.Errorf("error setting arn: %w", err))
	}
	if err := d.Set("delegation_enabled_date", aws.TimeValue(delegatedAccount.DelegationEnabledDate).Format(time.RFC3339)); err != nil {
		return diag.FromErr(fmt.Errorf("error setting delegation_enabled_date: %w", err))
	}
	if err := d.Set("email", delegatedAccount.Email); err != nil {
		return diag.FromErr(fmt.Errorf("error setting email: %w", err))
	}
	if err := d.Set("joined_method", delegatedAccount.JoinedMethod); err != nil {
		return diag.FromErr(fmt.Errorf("error setting joined_method: %w", err))
	}
	if err := d.Set("joined_timestamp", aws.TimeValue(delegatedAccount.JoinedTimestamp).Format(time.RFC3339)); err != nil {
		return diag.FromErr(fmt.Errorf("error setting joined_timestamp: %w", err))
	}
	if err := d.Set("name", delegatedAccount.Name); err != nil {
		return diag.FromErr(fmt.Errorf("error setting name: %w", err))
	}
	if err := d.Set("status", delegatedAccount.Status); err != nil {
		return diag.FromErr(fmt.Errorf("error setting status: %w", err))
	}
	if err := d.Set("account_id", accountID); err != nil {
		return diag.FromErr(fmt.Errorf("error setting account_id: %w", err))
	}
	if err := d.Set("service_principal", servicePrincipal); err != nil {
		return diag.FromErr(fmt.Errorf("error setting service_principal: %w", err))
	}

	return nil
}

func resourceDelegatedAdministratorDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).OrganizationsConn

	accountID, servicePrincipal, err := DecodeOrganizationDelegatedAdministratorID(d.Id())
	if err != nil {
		return diag.FromErr(fmt.Errorf("error decoding ID AWS Organization (%s) DelegatedAdministrators: %w", d.Id(), err))
	}
	input := &organizations.DeregisterDelegatedAdministratorInput{
		AccountId:        aws.String(accountID),
		ServicePrincipal: aws.String(servicePrincipal),
	}

	_, err = conn.DeregisterDelegatedAdministratorWithContext(ctx, input)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error deleting Organizations DelegatedAdministrator (%s): %w", d.Id(), err))
	}
	return nil
}

func DecodeOrganizationDelegatedAdministratorID(id string) (string, string, error) {
	idParts := strings.Split(id, "/")
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		return "", "", fmt.Errorf("%w, given: %q", errInvalidDelegatedAdminID, id)
	}
	return idParts[0], idParts[1], nil
}

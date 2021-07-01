package provider

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/securityhub"
	"github.com/cloudposse/terraform-provider-awsutils/internal/service/securityhub/finder"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceAwsUtilsSecurityHubControlDisablement() *schema.Resource {
	return &schema.Resource{
		Description: `Disables a Security Hub control in the configured region.

It can be useful to turn off security checks for controls that are not relevant to your environment. For example, you 
might use a single Amazon S3 bucket to log your CloudTrail logs. If so, you can turn off controls related to CloudTrail 
logging in all accounts and Regions except for the account and Region where the centralized S3 bucket is located. 
Disabling irrelevant controls reduces the number of irrelevant findings. It also removes the failed check from the 
readiness score for the associated standard.`,
		Create:        resourceAwsSecurityHubControlDisablementCreate,
		Read:          resourceAwsSecurityHubControlDisablementRead,
		Update:        resourceAwsSecurityHubControlDisablementUpdate,
		Delete:        resourceAwsSecurityHubControlDisablementDelete,
		SchemaVersion: 1,
		Schema: map[string]*schema.Schema{
			"control_arn": {
				Description: "The ARN of the Security Hub Standards Control to disable.",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
			},
			"reason": {
				Description: "The reason the control is being disabed.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
			},
		},
	}
}

func resourceAwsSecurityHubControlDisablementCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).securityhubconn
	controlArn := d.Get("control_arn").(string)
	reason := d.Get("reason").(string)

	input := &securityhub.UpdateStandardsControlInput{
		StandardsControlArn: &controlArn,
		ControlStatus:       aws.String("DISABLED"),
	}

	if reason != "" {
		input.DisabledReason = &reason
	}

	if _, err := conn.UpdateStandardsControl(input); err != nil {
		return fmt.Errorf("error disabling security hub control %s: %s", controlArn, err)
	}

	d.SetId(controlArn)

	return resourceAwsSecurityHubControlDisablementRead(d, meta)
}

func resourceAwsSecurityHubControlDisablementRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).securityhubconn
	controlArn := d.Get("control_arn").(string)

	control, err := finder.SecurityHubControl(conn, controlArn)
	if err != nil {
		return fmt.Errorf("error reading security hub control %s: %s", controlArn, err)
	}
	log.Printf("[DEBUG] Received Security Hub Control: %s", control)

	if !d.IsNewResource() && *control.ControlStatus != "DISABLED" {
		log.Printf("[WARN] Security Hub Control (%s) no longer disabled, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err := d.Set("reason", control.DisabledReason); err != nil {
		return err
	}

	return nil
}

func resourceAwsSecurityHubControlDisablementUpdate(d *schema.ResourceData, meta interface{}) error {
	if d.HasChanges("reason") {
		conn := meta.(*AWSClient).securityhubconn
		controlArn := d.Get("control_arn").(string)
		_, new := d.GetChange("reason")
		reason := new.(string)

		input := &securityhub.UpdateStandardsControlInput{
			StandardsControlArn: &controlArn,
			ControlStatus:       aws.String("DISABLED"),
			DisabledReason:      aws.String(reason),
		}

		if _, err := conn.UpdateStandardsControl(input); err != nil {
			return fmt.Errorf("error disabling security hub control %s: %s", controlArn, err)
		}
	}

	return nil
}

func resourceAwsSecurityHubControlDisablementDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).securityhubconn
	controlArn := d.Get("control_arn").(string)

	input := &securityhub.UpdateStandardsControlInput{
		StandardsControlArn: &controlArn,
		ControlStatus:       aws.String("ENABLED"),
		DisabledReason:      nil,
	}

	if _, err := conn.UpdateStandardsControl(input); err != nil {
		return fmt.Errorf("error updating security hub control %s: %s", controlArn, err)
	}

	return nil
}

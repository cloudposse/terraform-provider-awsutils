package meta

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/cloudposse/terraform-provider-awsutils/internal/verify"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceARN() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceARNRead,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: verify.ValidARN,
			},
			"partition": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"service": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"region": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"account": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"resource": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceARNRead(d *schema.ResourceData, meta interface{}) error {
	v := d.Get("arn").(string)
	arn, err := arn.Parse(v)
	if err != nil {
		return fmt.Errorf("error parsing '%s': %w", v, err)
	}

	d.SetId(arn.String())
	if err := d.Set("partition", arn.Partition); err != nil {
		return fmt.Errorf("error setting partition: %w", err)
	}
	if err := d.Set("service", arn.Service); err != nil {
		return fmt.Errorf("error setting service: %w", err)
	}
	if err := d.Set("region", arn.Region); err != nil {
		return fmt.Errorf("error setting region: %w", err)
	}
	if err := d.Set("account", arn.AccountID); err != nil {
		return fmt.Errorf("error setting account: %w", err)
	}
	if err := d.Set("resource", arn.Resource); err != nil {
		return fmt.Errorf("error setting resource: %w", err)
	}

	return nil
}

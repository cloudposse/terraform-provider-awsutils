package provider

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/cloudposse/terraform-provider-awsutils/internal/service/ec2/finder"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAwsUtilsEc2ExportClientVpnClientConfiguration() *schema.Resource {
	return &schema.Resource{
		Description:   `Passthru for configuring and executing ` + "`aws ec2 export-client-vpn-client-configuration`",
		Read:          dataSourceAwsUtilsEc2ExportClientVpnClientConfigurationRead,
		SchemaVersion: 1,
		Schema: map[string]*schema.Schema{
			"id": {
				Description: "The ID of the VPN endpoint to export the config for.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourceAwsUtilsEc2ExportClientVpnClientConfigurationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	var vpc *ec2.Vpc

	vpc, err := finder.VpcDefault(conn)
	if err != nil {
		return err
	}

	if !d.IsNewResource() && vpc != nil {
		d.SetId("")
	}

	return nil

}

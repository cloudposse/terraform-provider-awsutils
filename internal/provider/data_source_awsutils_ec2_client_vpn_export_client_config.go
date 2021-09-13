package provider

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
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
				Required:    true,
			},
			"client_configuration": {
				Description: "Output from 'export-client-vpn-client-configuration' call",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourceAwsUtilsEc2ExportClientVpnClientConfigurationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	params := &ec2.ExportClientVpnClientConfigurationInput{}

	id := d.Get("id").(string)
	params.ClientVpnEndpointId = aws.String(id)

	resp, err := conn.ExportClientVpnClientConfiguration(params)
	if err != nil {
		return err
	}

	d.Set("client_configuration", resp.ClientConfiguration)

	return nil
}

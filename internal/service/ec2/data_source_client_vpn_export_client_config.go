package ec2

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/cloudposse/terraform-provider-awsutils/internal/conns"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceEC2ExportClientVpnClientConfiguration() *schema.Resource {
	return &schema.Resource{
		Description:   `Passthru for configuring and executing ` + "`aws ec2 export-client-vpn-client-configuration`",
		Read:          dataSourceEc2ExportClientVpnClientConfigurationRead,
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

func dataSourceEc2ExportClientVpnClientConfigurationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).EC2Conn
	params := &ec2.ExportClientVpnClientConfigurationInput{}
	var id string

	if v, ok := d.GetOk("id"); ok {
		id = v.(string)
	}
	params.ClientVpnEndpointId = aws.String(id)

	resp, err := conn.ExportClientVpnClientConfiguration(params)
	if err != nil {
		return err
	}

	d.SetId(id)
	if err := d.Set("client_configuration", *resp.ClientConfiguration); err != nil {
		return fmt.Errorf("error setting names: %w", err)
	}

	return nil
}

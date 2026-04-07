package meta

import (
	"fmt"
	"log"

	"github.com/cloudposse/terraform-provider-awsutils/internal/conns"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourcePartition() *schema.Resource {
	return &schema.Resource{
		Read: dataSourcePartitionRead,

		Schema: map[string]*schema.Schema{
			"partition": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"dns_suffix": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"reverse_dns_prefix": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourcePartitionRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*conns.AWSClient)

	log.Printf("[DEBUG] Reading Partition.")
	d.SetId(meta.(*conns.AWSClient).Partition)

	log.Printf("[DEBUG] Setting AWS Partition to %s.", client.Partition)
	if err := d.Set("partition", meta.(*conns.AWSClient).Partition); err != nil {
		return fmt.Errorf("error setting partition: %w", err)
	}

	log.Printf("[DEBUG] Setting AWS URL Suffix to %s.", client.DNSSuffix)
	if err := d.Set("dns_suffix", meta.(*conns.AWSClient).DNSSuffix); err != nil {
		return fmt.Errorf("error setting dns_suffix: %w", err)
	}

	if err := d.Set("reverse_dns_prefix", meta.(*conns.AWSClient).ReverseDNSPrefix); err != nil {
		return fmt.Errorf("error setting reverse_dns_prefix: %w", err)
	}
	log.Printf("[DEBUG] Setting service prefix to %s.", meta.(*conns.AWSClient).ReverseDNSPrefix)

	return nil
}

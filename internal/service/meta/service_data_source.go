package meta

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/cloudposse/terraform-provider-awsutils/internal/conns"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	errReverseServiceDNSNameTooShort = errors.New("reverse service DNS names must have at least 4 parts")
	errServiceDNSNameTooShort        = errors.New("service DNS names must have at least 4 parts")
	errNoServiceID                   = errors.New("service ID not provided directly or through a DNS name")
)

func DataSourceService() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceServiceRead,

		Schema: map[string]*schema.Schema{
			"dns_name": {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				ExactlyOneOf: []string{"dns_name", "reverse_dns_name", "service_id"},
			},
			"partition": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"region": {
				Type:          schema.TypeString,
				Computed:      true,
				Optional:      true,
				ConflictsWith: []string{"dns_name", "reverse_dns_name"},
			},
			"reverse_dns_name": {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				ExactlyOneOf: []string{"dns_name", "reverse_dns_name", "service_id"},
			},
			"reverse_dns_prefix": {
				Type:          schema.TypeString,
				Computed:      true,
				Optional:      true,
				ConflictsWith: []string{"dns_name", "reverse_dns_name"},
			},
			"service_id": {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				ExactlyOneOf: []string{"dns_name", "reverse_dns_name", "service_id"},
			},
			"supported": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func dataSourceServiceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*conns.AWSClient)

	if v, ok := d.GetOk("reverse_dns_name"); ok {
		serviceParts := strings.Split(v.(string), ".")
		if len(serviceParts) < 4 {
			return fmt.Errorf("%w (%s has %d)", errReverseServiceDNSNameTooShort, v.(string), len(serviceParts))
		}

		if err := d.Set("service_id", serviceParts[len(serviceParts)-1]); err != nil {
			return fmt.Errorf("error setting service_id: %w", err)
		}
		if err := d.Set("region", serviceParts[len(serviceParts)-2]); err != nil {
			return fmt.Errorf("error setting region: %w", err)
		}
		if err := d.Set("reverse_dns_prefix", strings.Join(serviceParts[0:len(serviceParts)-2], ".")); err != nil {
			return fmt.Errorf("error setting reverse_dns_prefix: %w", err)
		}
	}

	if v, ok := d.GetOk("dns_name"); ok {
		serviceParts := InvertStringSlice(strings.Split(v.(string), "."))
		if len(serviceParts) < 4 {
			return fmt.Errorf("%w (%s has %d)", errServiceDNSNameTooShort, v.(string), len(serviceParts))
		}

		if err := d.Set("service_id", serviceParts[len(serviceParts)-1]); err != nil {
			return fmt.Errorf("error setting service_id: %w", err)
		}
		if err := d.Set("region", serviceParts[len(serviceParts)-2]); err != nil {
			return fmt.Errorf("error setting region: %w", err)
		}
		if err := d.Set("reverse_dns_prefix", strings.Join(serviceParts[0:len(serviceParts)-2], ".")); err != nil {
			return fmt.Errorf("error setting reverse_dns_prefix: %w", err)
		}
	}

	if _, ok := d.GetOk("region"); !ok {
		if err := d.Set("region", client.Region); err != nil {
			return fmt.Errorf("error setting region: %w", err)
		}
	}

	if _, ok := d.GetOk("service_id"); !ok {
		return errNoServiceID
	}

	if _, ok := d.GetOk("reverse_dns_prefix"); !ok {
		dnsParts := strings.Split(meta.(*conns.AWSClient).DNSSuffix, ".")
		if err := d.Set("reverse_dns_prefix", strings.Join(InvertStringSlice(dnsParts), ".")); err != nil {
			return fmt.Errorf("error setting reverse_dns_prefix: %w", err)
		}
	}

	reverseDNS := fmt.Sprintf("%s.%s.%s", d.Get("reverse_dns_prefix").(string), d.Get("region").(string), d.Get("service_id").(string))
	if err := d.Set("reverse_dns_name", reverseDNS); err != nil {
		return fmt.Errorf("error setting reverse_dns_name: %w", err)
	}
	if err := d.Set("dns_name", strings.ToLower(strings.Join(InvertStringSlice(strings.Split(reverseDNS, ".")), "."))); err != nil {
		return fmt.Errorf("error setting dns_name: %w", err)
	}

	if err := d.Set("supported", true); err != nil {
		return fmt.Errorf("error setting supported: %w", err)
	}
	if partition, ok := endpoints.PartitionForRegion(endpoints.DefaultPartitions(), d.Get("region").(string)); ok {
		if err := d.Set("partition", partition.ID()); err != nil {
			return fmt.Errorf("error setting partition: %w", err)
		}
		if _, ok := partition.Services()[d.Get("service_id").(string)]; !ok {
			if err := d.Set("supported", false); err != nil {
				return fmt.Errorf("error setting supported: %w", err)
			}
		}
	}

	d.SetId(reverseDNS)

	return nil
}

// invertStringSlice returns inverted string slice without sorting slice like sort.Reverse().
func InvertStringSlice(slice []string) []string {
	inverse := make([]string, 0)
	for i := 0; i < len(slice); i++ {
		inverse = append(inverse, slice[len(slice)-i-1])
	}
	return inverse
}

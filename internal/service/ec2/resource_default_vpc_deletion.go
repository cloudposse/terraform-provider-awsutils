package ec2

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/cloudposse/terraform-provider-awsutils/internal/conns"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const noDefaultVPC string = "no-default-vpc-found"

func ResourceDefaultVpcDeletion() *schema.Resource {
	return &schema.Resource{
		Description: `Deletes the default VPC along with the child resources of the VPC including Subnets, Route Tables, NACLs and Internet
Gateways in the specified region.

Best-practices call for not using the default VPC, but rather, creating a new set of VPCs as necessary. AWS Security
Hub will flag the default VPCs as non-compliant if they aren't configured with best-practices. Rather than jumping
through hoops, it's easier to delete to default VPCs. This task cannot be accomplished with the official AWS
Terraform Provider, so this resource is necessary.

Please note that applying this resource is destructive and nonreversible. This resource is unusual as it will
**DELETE** infrastructure when ` + "`terraform apply`" + ` is run rather than creating it. This is a permanent
deletion and nothing will be restored when ` + "`terraform destroy`" + ` is run. `,
		Create:        resourceDefaultVpcDeletionCreate,
		Read:          resourceDefaultVpcDeletionRead,
		Delete:        resourceDefaultVpcDeletionDelete,
		SchemaVersion: 1,
		Schema: map[string]*schema.Schema{
			"region": {
				Description: "The region to delete the default VPC from. If not specified, uses the provider's region.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
			},
			"id": {
				Description: "The ID of the VPC that was deleted.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

// getEC2ConnForRegion returns an EC2 connection for the specified region,
// or the default connection if no region is specified.
func getEC2ConnForRegion(d *schema.ResourceData, meta any) (*ec2.EC2, string, error) {
	client := meta.(*conns.AWSClient)

	// Check if region is explicitly set on the resource
	if v, ok := d.GetOk("region"); ok && v.(string) != "" {
		region := v.(string)
		log.Printf("[DEBUG] Creating EC2 client for account %s in region %s", client.AccountID, region)
		// Create a new EC2 client for the specified region
		sess, err := conns.NewSessionForRegion(client.Session.Config, region, client.TerraformVersion)
		if err != nil {
			return nil, "", fmt.Errorf("error creating session for region %s: %w", region, err)
		}
		return ec2.New(sess), region, nil
	}

	// Use the provider's default region
	return client.EC2Conn, client.Region, nil
}

func resourceDefaultVpcDeletionCreate(d *schema.ResourceData, meta any) error {
	conn, region, err := getEC2ConnForRegion(d, meta)
	if err != nil {
		return err
	}

	// Store the region in state
	if err := d.Set("region", region); err != nil {
		return fmt.Errorf("error setting region: %w", err)
	}

	vpc, err := FindDefaultVpc(conn)
	if err != nil {
		return err
	}

	if vpc == nil {
		d.SetId(noDefaultVPC)
		return resourceDefaultVpcDeletionRead(d, meta)
	}

	vpcid := aws.StringValue(vpc.VpcId)
	log.Printf("[INFO] Deleting default VPC %s in account %s region %s", vpcid, meta.(*conns.AWSClient).AccountID, region)

	if err = deleteInternetGateway(conn, vpcid); err != nil {
		return err
	}

	if err = deleteSubnets(conn, vpcid); err != nil {
		return err
	}

	if err = deleteVpc(conn, vpcid); err != nil {
		return err
	}

	d.SetId(vpcid)

	return resourceDefaultVpcDeletionRead(d, meta)
}

func resourceDefaultVpcDeletionRead(d *schema.ResourceData, meta any) error {
	conn, _, err := getEC2ConnForRegion(d, meta)
	if err != nil {
		return err
	}

	vpc, err := FindDefaultVpc(conn)
	if err != nil {
		return err
	}

	if !d.IsNewResource() && vpc != nil {
		d.SetId("")
	}

	return nil
}

func resourceDefaultVpcDeletionDelete(d *schema.ResourceData, meta any) error {
	log.Printf("[INFO] Removing default VPC deletion state")
	return nil
}

func deleteInternetGateway(conn *ec2.EC2, vpcID string) error {
	// Detach and Delete Internet Gateway
	ig, err := FindInternetGatewayForVPC(conn, vpcID)
	if err != nil {
		return fmt.Errorf("error while looking for EC2 Internet Gateway for VPC (%s): %w", vpcID, err)
	}

	if ig != nil {
		igid := *ig.InternetGatewayId
		detachInternetGatewayInput := &ec2.DetachInternetGatewayInput{
			InternetGatewayId: aws.String(igid),
			VpcId:             &vpcID,
		}

		if _, err = conn.DetachInternetGateway(detachInternetGatewayInput); err != nil {
			return fmt.Errorf("error while detaching EC2 Internet Gateway (%s): %w", igid, err)
		}

		deleteInternetGatewayInput := &ec2.DeleteInternetGatewayInput{
			InternetGatewayId: aws.String(igid),
		}

		if _, err = conn.DeleteInternetGateway(deleteInternetGatewayInput); err != nil {
			return fmt.Errorf("error while deleting EC2 Internet Gateway (%s): %w", igid, err)
		}
	}
	return nil
}

func deleteSubnets(conn *ec2.EC2, vpcID string) error {
	subnets, err := FindSubnetsForVPC(conn, vpcID)
	if err != nil {
		return fmt.Errorf("error while looking for EC2 Subnets for VPC (%s): %w", vpcID, err)
	}

	for _, s := range subnets {
		deleteSubnetInput := &ec2.DeleteSubnetInput{
			SubnetId: s.SubnetId,
		}

		if _, err = conn.DeleteSubnet(deleteSubnetInput); err != nil {
			return fmt.Errorf("error while deleting EC2 Subnet (%s): %w", *s.SubnetId, err)
		}
	}
	return nil
}

func deleteVpc(conn *ec2.EC2, vpcID string) error {
	deleteVpcInput := &ec2.DeleteVpcInput{
		VpcId: aws.String(vpcID),
	}

	if _, err := conn.DeleteVpc(deleteVpcInput); err != nil {
		return fmt.Errorf("error while deleting EC2 VPC (%s): %w", vpcID, err)
	}

	return nil
}

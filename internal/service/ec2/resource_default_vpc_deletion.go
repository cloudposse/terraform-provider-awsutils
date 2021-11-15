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
Gateways in the configured region.
		
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
			"id": {
				Description: "The ID of the VPC that was deleted.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func resourceDefaultVpcDeletionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).EC2Conn
	var vpc *ec2.Vpc

	vpc, err := FindDefaultVpc(conn)
	if err != nil {
		return err
	}

	if vpc == nil {
		d.SetId(noDefaultVPC)
		return resourceDefaultVpcDeletionRead(d, meta)
	}

	vpcid := aws.StringValue(vpc.VpcId)

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

func resourceDefaultVpcDeletionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).EC2Conn
	var vpc *ec2.Vpc

	vpc, err := FindDefaultVpc(conn)
	if err != nil {
		return err
	}

	if !d.IsNewResource() && vpc != nil {
		d.SetId("")
	}

	return nil
}

func resourceDefaultVpcDeletionDelete(d *schema.ResourceData, meta interface{}) error {
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

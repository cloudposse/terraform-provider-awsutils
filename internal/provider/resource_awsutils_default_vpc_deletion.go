package provider

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/cloudposse/terraform-provider-awsutils/internal/service/ec2/finder"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const noDefaultVPC string = "no-default-vpc-found"

func resourceAwsUtilsDefaultVpcDeletion() *schema.Resource {
	//lintignore:R011
	return &schema.Resource{
		Create:        resourceAwsDefaultVpcDeletionCreate,
		Read:          resourceAwsDefaultVpcDeletionRead,
		Delete:        resourceAwsDefaultVpcDeletionDelete,
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

func resourceAwsDefaultVpcDeletionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	var vpc *ec2.Vpc

	vpc, err := finder.VpcDefault(conn)
	if err != nil {
		return err
	}

	if vpc == nil {
		d.SetId(noDefaultVPC)
		return resourceAwsDefaultVpcDeletionRead(d, meta)
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

	return resourceAwsDefaultVpcDeletionRead(d, meta)
}

func resourceAwsDefaultVpcDeletionRead(d *schema.ResourceData, meta interface{}) error {
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

func resourceAwsDefaultVpcDeletionDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Removing default VPC deletion state")
	return nil
}

func deleteInternetGateway(conn *ec2.EC2, vpcID string) error {
	// Detach and Delete Internet Gateway
	ig, err := finder.InternetGatewayForVPC(conn, vpcID)
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
	subnets, err := finder.SubnetsForVPC(conn, vpcID)
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

package finder

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// InternetGatewayForVPC looks up the Internet Gateway for the given VPC. When not found, returns nil and potentially an API error.
func InternetGatewayForVPC(conn *ec2.EC2, vpcID string) (*ec2.InternetGateway, error) {
	filters := []*ec2.Filter{
		{
			Name:   aws.String("attachment.vpc-id"),
			Values: []*string{aws.String(vpcID)},
		},
	}

	input := &ec2.DescribeInternetGatewaysInput{
		Filters: filters,
	}

	output, err := conn.DescribeInternetGateways(input)

	if err != nil {
		return nil, err
	}

	if output == nil {
		return nil, nil
	}

	for _, ig := range output.InternetGateways {
		if ig == nil {
			continue
		}

		return ig, nil
	}

	return nil, nil
}

// SubnetsForVPC looks up a the Subnets for a VPC. When not found, returns nil and potentially an API error.
func SubnetsForVPC(conn *ec2.EC2, vpcID string) ([]*ec2.Subnet, error) {
	filters := []*ec2.Filter{
		{
			Name:   aws.String("vpc-id"),
			Values: []*string{aws.String(vpcID)},
		},
	}

	input := &ec2.DescribeSubnetsInput{
		Filters: filters,
	}

	output, err := conn.DescribeSubnets(input)

	if err != nil {
		return nil, err
	}

	if output == nil || len(output.Subnets) == 0 || output.Subnets[0] == nil {
		return nil, nil
	}

	return output.Subnets, nil
}

// VpcDefault looks up the Default Vpc. When not found, returns nil and potentially an API error.
func VpcDefault(conn *ec2.EC2) (*ec2.Vpc, error) {
	filters := []*ec2.Filter{
		{
			Name:   aws.String("isDefault"),
			Values: []*string{aws.String("true")},
		},
	}

	input := &ec2.DescribeVpcsInput{
		Filters: filters,
	}

	output, err := conn.DescribeVpcs(input)

	if err != nil {
		return nil, err
	}

	if output == nil {
		return nil, nil
	}

	for _, vpc := range output.Vpcs {
		if vpc == nil {
			continue
		}

		return vpc, nil
	}

	return nil, nil
}

package ec2

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/cloudposse/terraform-provider-awsutils/internal/conns"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestGetEC2ConnForRegion_UsesProviderRegion(t *testing.T) {
	providerRegion := "us-west-2"

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(providerRegion),
		Credentials: credentials.NewStaticCredentials("test", "test", ""),
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	providerEC2Conn := ec2.New(sess)

	client := &conns.AWSClient{
		EC2Conn:          providerEC2Conn,
		Region:           providerRegion,
		Session:          sess,
		AccountID:        "123456789012",
		TerraformVersion: "1.0.0",
	}

	resource := ResourceDefaultVpcDeletion()
	d := schema.TestResourceDataRaw(t, resource.Schema, map[string]any{})

	conn, region, err := getEC2ConnForRegion(d, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if region != providerRegion {
		t.Errorf("expected region %q, got %q", providerRegion, region)
	}

	if conn != providerEC2Conn {
		t.Error("expected to receive provider's EC2Conn when region not specified")
	}
}

func TestGetEC2ConnForRegion_UsesResourceRegion(t *testing.T) {
	providerRegion := "us-west-2"
	resourceRegion := "eu-west-1"

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(providerRegion),
		Credentials: credentials.NewStaticCredentials("test", "test", ""),
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	providerEC2Conn := ec2.New(sess)

	client := &conns.AWSClient{
		EC2Conn:          providerEC2Conn,
		Region:           providerRegion,
		Session:          sess,
		AccountID:        "123456789012",
		TerraformVersion: "1.0.0",
	}

	resource := ResourceDefaultVpcDeletion()
	d := schema.TestResourceDataRaw(t, resource.Schema, map[string]any{
		"region": resourceRegion,
	})

	conn, region, err := getEC2ConnForRegion(d, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if region != resourceRegion {
		t.Errorf("expected region %q, got %q", resourceRegion, region)
	}

	if conn == providerEC2Conn {
		t.Error("expected a new EC2 client when region is specified, got provider's EC2Conn")
	}

	if *conn.Config.Region != resourceRegion {
		t.Errorf("expected EC2 client region %q, got %q", resourceRegion, *conn.Config.Region)
	}
}

func TestGetEC2ConnForRegion_EmptyRegionUsesProvider(t *testing.T) {
	providerRegion := "us-east-1"

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(providerRegion),
		Credentials: credentials.NewStaticCredentials("test", "test", ""),
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	providerEC2Conn := ec2.New(sess)

	client := &conns.AWSClient{
		EC2Conn:          providerEC2Conn,
		Region:           providerRegion,
		Session:          sess,
		AccountID:        "123456789012",
		TerraformVersion: "1.0.0",
	}

	resource := ResourceDefaultVpcDeletion()
	d := schema.TestResourceDataRaw(t, resource.Schema, map[string]any{
		"region": "",
	})

	conn, region, err := getEC2ConnForRegion(d, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if region != providerRegion {
		t.Errorf("expected region %q, got %q", providerRegion, region)
	}

	if conn != providerEC2Conn {
		t.Error("expected provider's EC2Conn when region is empty string")
	}
}

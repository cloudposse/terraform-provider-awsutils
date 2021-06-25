package waiter

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudhsmv2"
	"github.com/cloudposse/terraform-provider-awsutils/internal/service/cloudhsmv2/finder"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func ClusterState(conn *cloudhsmv2.CloudHSMV2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		cluster, err := finder.Cluster(conn, id)

		if err != nil {
			return nil, "", err
		}

		if cluster == nil {
			return nil, "", nil
		}

		return cluster, aws.StringValue(cluster.State), err
	}
}

func HsmState(conn *cloudhsmv2.CloudHSMV2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		hsm, err := finder.Hsm(conn, id, "")

		if err != nil {
			return nil, "", err
		}

		if hsm == nil {
			return nil, "", nil
		}

		return hsm, aws.StringValue(hsm.State), err
	}
}

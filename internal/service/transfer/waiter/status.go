package waiter

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/transfer"
	"github.com/cloudposse/terraform-provider-awsutils/internal/service/transfer/finder"
	"github.com/cloudposse/terraform-provider-awsutils/internal/tfresource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func ServerState(conn *transfer.Transfer, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := finder.ServerByID(conn, id)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return output, aws.StringValue(output.State), nil
	}
}

func UserState(conn *transfer.Transfer, serverId, userName string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := finder.UserByID(conn, serverId, userName)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return output, "Available", nil
	}
}

package waiter

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/cloudposse/terraform-provider-awsutils/internal/service/sfn/finder"
	"github.com/cloudposse/terraform-provider-awsutils/internal/tfresource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func StateMachineStatus(conn *sfn.SFN, stateMachineArn string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := finder.StateMachineByARN(conn, stateMachineArn)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return output, aws.StringValue(output.Status), nil
	}
}

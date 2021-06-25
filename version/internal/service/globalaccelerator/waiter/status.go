package waiter

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/globalaccelerator"
	"github.com/cloudposse/terraform-provider-awsutils/internal/service/globalaccelerator/finder"
	"github.com/cloudposse/terraform-provider-awsutils/internal/tfresource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// AcceleratorStatus fetches the Accelerator and its Status
func AcceleratorStatus(conn *globalaccelerator.GlobalAccelerator, arn string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		accelerator, err := finder.AcceleratorByARN(conn, arn)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return accelerator, aws.StringValue(accelerator.Status), nil
	}
}

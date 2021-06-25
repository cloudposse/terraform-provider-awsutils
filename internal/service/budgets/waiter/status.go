package waiter

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/budgets"
	"github.com/cloudposse/terraform-provider-awsutils/internal/service/budgets/finder"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func ActionStatus(conn *budgets.Budgets, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		out, err := finder.ActionById(conn, id)
		if err != nil {
			if tfawserr.ErrCodeEquals(err, budgets.ErrCodeNotFoundException) {
				return nil, "", nil
			}
			return nil, "", err
		}

		action := out.Action
		return action, aws.StringValue(action.Status), err
	}
}

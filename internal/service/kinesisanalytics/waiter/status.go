package waiter

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesisanalytics"
	"github.com/cloudposse/terraform-provider-awsutils/internal/service/kinesisanalytics/finder"
	"github.com/cloudposse/terraform-provider-awsutils/internal/tfresource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// ApplicationStatus fetches the ApplicationDetail and its Status
func ApplicationStatus(conn *kinesisanalytics.KinesisAnalytics, name string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		applicationDetail, err := finder.ApplicationDetailByName(conn, name)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return applicationDetail, aws.StringValue(applicationDetail.ApplicationStatus), nil
	}
}

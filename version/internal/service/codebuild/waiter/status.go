package waiter

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/cloudposse/terraform-provider-awsutils/internal/service/codebuild/finder"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	ReportGroupStatusUnknown  = "Unknown"
	ReportGroupStatusNotFound = "NotFound"
)

// ReportGroupStatus fetches the Report Group and its Status
func ReportGroupStatus(conn *codebuild.CodeBuild, arn string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := finder.ReportGroupByArn(conn, arn)
		if err != nil {
			return nil, ReportGroupStatusUnknown, err
		}

		if output == nil {
			return nil, ReportGroupStatusNotFound, nil
		}

		return output, aws.StringValue(output.Status), nil
	}
}

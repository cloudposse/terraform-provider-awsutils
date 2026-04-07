package iam

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/cloudposse/terraform-provider-awsutils/internal/conns"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var (
	errEmptyGetAccessKeyLastUsedResponse = errors.New("error fetching IAM Access Key username via GetAccessKeyLastUsed: empty response")
	errMissingSecretAccessKey            = errors.New("CreateAccessKey response did not contain a Secret Access Key as expected")
)

func ResourceExpiringAccessKey() *schema.Resource {
	return &schema.Resource{
		Description: `Provides an IAM access key that expires after max_age seconds. This is a set of credentials that allow API requests to be made as an IAM user.`,
		Create:      resourceAccessKeyCreate,
		Read:        resourceAccessKeyRead,
		Update:      resourceAccessKeyUpdate,
		Delete:      resourceAccessKeyDelete,

		CustomizeDiff: resourceAccessKeyDiff,

		Importer: &schema.ResourceImporter{
			// ListAccessKeys requires UserName field in certain scenarios:
			//   ValidationError: Must specify userName when calling with non-User credentials
			// To prevent import from requiring this extra information, use GetAccessKeyLastUsed.
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				conn := meta.(*conns.AWSClient).IAMConn

				input := &iam.GetAccessKeyLastUsedInput{
					AccessKeyId: aws.String(d.Id()),
				}

				output, err := conn.GetAccessKeyLastUsed(input)
				if err != nil {
					return nil, fmt.Errorf("error fetching IAM Access Key (%s) username via GetAccessKeyLastUsed: %w", d.Id(), err)
				}

				if output == nil || output.UserName == nil {
					return nil, fmt.Errorf("%w: %s", errEmptyGetAccessKeyLastUsedResponse, d.Id())
				}

				if err := d.Set("user", output.UserName); err != nil {
					return nil, fmt.Errorf("error setting user: %w", err)
				}

				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"create_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"expiration_date": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
			"encrypted_secret": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"encrypted_ses_smtp_password_v4": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"key_fingerprint": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"max_age": {
				Type:     schema.TypeInt,
				ForceNew: true,
				Optional: true,
				Default:  math.MaxInt32,
			},
			"pgp_key": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
			"secret": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"ses_smtp_password_v4": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"status": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      iam.StatusTypeActive,
				ValidateFunc: validation.StringInSlice(iam.StatusType_Values(), false),
			},
			"user": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAccessKeyDiff(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	if d.Id() == "" {
		return nil
	}

	if v, ok := d.GetOk("expiration_date"); ok {
		now := time.Now().UTC()
		rotationTimestamp, err := time.Parse(time.RFC3339, v.(string))
		if err != nil {
			return fmt.Errorf("error parsing expiration_date (%s): %w", v.(string), err)
		}

		if now.After(rotationTimestamp) {
			if err := d.SetNewComputed("expiration_date"); err != nil {
				return fmt.Errorf("error setting expiration_date as new computed: %w", err)
			}
		}
	}
	return nil
}

func resourceAccessKeyCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).IAMConn

	request := &iam.CreateAccessKeyInput{
		UserName: aws.String(d.Get("user").(string)),
	}

	createResp, err := conn.CreateAccessKey(request)
	if err != nil {
		return fmt.Errorf(
			"error creating access key for user %s: %w",
			*request.UserName,
			err,
		)
	}

	if createResp.AccessKey == nil || createResp.AccessKey.SecretAccessKey == nil {
		return errMissingSecretAccessKey
	}

	d.SetId(aws.StringValue(createResp.AccessKey.AccessKeyId))

	// cleanupOnError attempts a best-effort delete of the just-created access
	// key when a post-create step fails. The original error is always returned;
	// any deletion error is logged but swallowed so the caller still sees the
	// actionable failure.
	cleanupOnError := func(origErr error) error {
		_, delErr := conn.DeleteAccessKey(&iam.DeleteAccessKeyInput{
			AccessKeyId: createResp.AccessKey.AccessKeyId,
			UserName:    request.UserName,
		})
		if delErr != nil {
			log.Printf("[WARN] failed to clean up IAM access key %s after post-create error: %s",
				aws.StringValue(createResp.AccessKey.AccessKeyId), delErr)
		}
		return origErr
	}

	sesSMTPPasswordV4, err := SessmTPPasswordFromSecretKeySigV4(createResp.AccessKey.SecretAccessKey, meta.(*conns.AWSClient).Region)
	if err != nil {
		return cleanupOnError(fmt.Errorf("error getting SES SigV4 SMTP Password from Secret Access Key: %w", err))
	}

	if v, ok := d.GetOk("pgp_key"); ok {
		pgpKey := v.(string)
		encryptionKey, err := RetrieveGPGKey(pgpKey)
		if err != nil {
			return cleanupOnError(err)
		}
		fingerprint, encrypted, err := EncryptValue(encryptionKey, *createResp.AccessKey.SecretAccessKey, "IAM Access Key Secret")
		if err != nil {
			return cleanupOnError(err)
		}

		if err := d.Set("key_fingerprint", fingerprint); err != nil {
			return cleanupOnError(fmt.Errorf("error setting key_fingerprint: %w", err))
		}
		if err := d.Set("encrypted_secret", encrypted); err != nil {
			return cleanupOnError(fmt.Errorf("error setting encrypted_secret: %w", err))
		}

		_, encrypted, err = EncryptValue(encryptionKey, sesSMTPPasswordV4, "SES SMTP password")
		if err != nil {
			return cleanupOnError(err)
		}

		if err := d.Set("encrypted_ses_smtp_password_v4", encrypted); err != nil {
			return cleanupOnError(fmt.Errorf("error setting encrypted_ses_smtp_password_v4: %w", err))
		}
	} else {
		if err := d.Set("secret", createResp.AccessKey.SecretAccessKey); err != nil {
			return cleanupOnError(err)
		}

		if err := d.Set("ses_smtp_password_v4", sesSMTPPasswordV4); err != nil {
			return cleanupOnError(err)
		}
	}

	if v, ok := d.GetOk("status"); ok && v.(string) == iam.StatusTypeInactive {
		input := &iam.UpdateAccessKeyInput{
			AccessKeyId: aws.String(d.Id()),
			Status:      aws.String(iam.StatusTypeInactive),
			UserName:    aws.String(d.Get("user").(string)),
		}

		_, err := conn.UpdateAccessKey(input)
		if err != nil {
			return fmt.Errorf("error deactivating IAM Access Key (%s): %w", d.Id(), err)
		}

		createResp.AccessKey.Status = aws.String(iam.StatusTypeInactive)
	}

	return resourceAccessKeyReadResult(d, &iam.AccessKeyMetadata{
		AccessKeyId: createResp.AccessKey.AccessKeyId,
		CreateDate:  createResp.AccessKey.CreateDate,
		Status:      createResp.AccessKey.Status,
		UserName:    createResp.AccessKey.UserName,
	})
}

func resourceAccessKeyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).IAMConn

	request := &iam.ListAccessKeysInput{
		UserName: aws.String(d.Get("user").(string)),
	}

	getResp, err := conn.ListAccessKeys(request)
	if err != nil {
		if iamerr, ok := err.(awserr.Error); ok && iamerr.Code() == "NoSuchEntity" { // XXX TEST ME
			// the user does not exist, so the key can't exist.
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading IAM access key: %w", err)
	}

	for _, key := range getResp.AccessKeyMetadata {
		if key.AccessKeyId != nil && *key.AccessKeyId == d.Id() {
			return resourceAccessKeyReadResult(d, key)
		}
	}

	// Guess the key isn't around anymore.
	d.SetId("")
	return nil
}

func resourceAccessKeyReadResult(d *schema.ResourceData, key *iam.AccessKeyMetadata) error {
	d.SetId(aws.StringValue(key.AccessKeyId))

	if key.CreateDate != nil {
		createDate := key.CreateDate
		expirationDate := createDate.Add(time.Duration(d.Get("max_age").(int)) * time.Second)

		if err := d.Set("create_date", aws.TimeValue(createDate).Format(time.RFC3339)); err != nil {
			return fmt.Errorf("error setting create_date: %w", err)
		}
		if err := d.Set("expiration_date", expirationDate.Format(time.RFC3339)); err != nil {
			return fmt.Errorf("error setting expiration_date: %w", err)
		}
	} else {
		if err := d.Set("create_date", nil); err != nil {
			return fmt.Errorf("error setting create_date: %w", err)
		}
		if err := d.Set("expiration_date", nil); err != nil {
			return fmt.Errorf("error setting expiration_date: %w", err)
		}
	}

	if err := d.Set("status", key.Status); err != nil {
		return fmt.Errorf("error setting status: %w", err)
	}
	if err := d.Set("user", key.UserName); err != nil {
		return fmt.Errorf("error setting user: %w", err)
	}

	return nil
}

func resourceAccessKeyUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).IAMConn

	if d.HasChange("status") {
		if err := resourceAccessKeyStatusUpdate(conn, d); err != nil {
			return err
		}
	}

	return resourceAccessKeyRead(d, meta)
}

func resourceAccessKeyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).IAMConn

	request := &iam.DeleteAccessKeyInput{
		AccessKeyId: aws.String(d.Id()),
		UserName:    aws.String(d.Get("user").(string)),
	}

	if _, err := conn.DeleteAccessKey(request); err != nil {
		return fmt.Errorf("error deleting access key %s: %w", d.Id(), err)
	}
	return nil
}

func resourceAccessKeyStatusUpdate(conn *iam.IAM, d *schema.ResourceData) error {
	request := &iam.UpdateAccessKeyInput{
		AccessKeyId: aws.String(d.Id()),
		Status:      aws.String(d.Get("status").(string)),
		UserName:    aws.String(d.Get("user").(string)),
	}

	if _, err := conn.UpdateAccessKey(request); err != nil {
		return fmt.Errorf("error updating access key %s: %w", d.Id(), err)
	}
	return nil
}

func hmacSignature(key []byte, value []byte) ([]byte, error) {
	h := hmac.New(sha256.New, key)
	if _, err := h.Write(value); err != nil {
		return []byte(""), err
	}
	return h.Sum(nil), nil
}

func SessmTPPasswordFromSecretKeySigV4(key *string, region string) (string, error) {
	if key == nil {
		return "", nil
	}
	const version = byte(0x04)
	date := []byte("11111111")
	service := []byte("ses")
	terminal := []byte("aws4_request")
	message := []byte("SendRawEmail")

	rawSig, err := hmacSignature([]byte("AWS4"+*key), date)
	if err != nil {
		return "", err
	}

	if rawSig, err = hmacSignature(rawSig, []byte(region)); err != nil {
		return "", err
	}
	if rawSig, err = hmacSignature(rawSig, service); err != nil {
		return "", err
	}
	if rawSig, err = hmacSignature(rawSig, terminal); err != nil {
		return "", err
	}
	if rawSig, err = hmacSignature(rawSig, message); err != nil {
		return "", err
	}

	versionedSig := make([]byte, 0, len(rawSig)+1)
	versionedSig = append(versionedSig, version)
	versionedSig = append(versionedSig, rawSig...)
	return base64.StdEncoding.EncodeToString(versionedSig), nil
}

package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type ssmAPI interface {
	GetParameter(ctx context.Context, in *ssm.GetParameterInput, opts ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

// keySource says where the Firebase service account key lives: an SSM
// SecureString in AWS, a local file otherwise. The parameter wins.
type keySource struct {
	Param string
	SSM   ssmAPI
	File  string
}

func serviceAccountKey(ctx context.Context, src keySource) ([]byte, error) {
	switch {
	case src.Param != "":
		out, err := src.SSM.GetParameter(ctx, &ssm.GetParameterInput{Name: aws.String(src.Param), WithDecryption: aws.Bool(true)})
		if err != nil {
			return nil, fmt.Errorf("reading parameter %s: %w", src.Param, err)
		}
		return []byte(aws.ToString(out.Parameter.Value)), nil
	case src.File != "":
		key, err := os.ReadFile(src.File)
		if err != nil {
			return nil, fmt.Errorf("reading key file: %w", err)
		}
		return key, nil
	}
	return nil, errors.New("set FIREBASE_SERVICE_ACCOUNT_PARAM (AWS) or FIREBASE_SERVICE_ACCOUNT_FILE (local)")
}

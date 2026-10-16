package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type fakeSSM struct {
	in  *ssm.GetParameterInput
	err error
}

func (f *fakeSSM) GetParameter(_ context.Context, in *ssm.GetParameterInput, _ ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	f.in = in
	if f.err != nil {
		return nil, f.err
	}
	return &ssm.GetParameterOutput{Parameter: &types.Parameter{Value: aws.String(`{"from":"ssm"}`)}}, nil
}

func TestKeyComesFromSSMWhenAParameterIsNamed(t *testing.T) {
	client := &fakeSSM{}
	key, err := serviceAccountKey(context.Background(), keySource{Param: "/status-trens-sp/firebase-sa", SSM: client})
	if err != nil {
		t.Fatal(err)
	}
	if string(key) != `{"from":"ssm"}` {
		t.Errorf("key = %s, want the parameter value", key)
	}
	if aws.ToString(client.in.Name) != "/status-trens-sp/firebase-sa" || !aws.ToBool(client.in.WithDecryption) {
		t.Errorf("GetParameter(%v): want the named parameter, decrypted", client.in)
	}
}

func TestKeyComesFromTheFileWithoutAParameter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sa.json")
	if err := os.WriteFile(path, []byte(`{"from":"file"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	key, err := serviceAccountKey(context.Background(), keySource{File: path})
	if err != nil {
		t.Fatal(err)
	}
	if string(key) != `{"from":"file"}` {
		t.Errorf("key = %s, want the file content", key)
	}
}

func TestKeyReportsFailures(t *testing.T) {
	boom := errors.New("access denied")
	if _, err := serviceAccountKey(context.Background(), keySource{Param: "/x", SSM: &fakeSSM{err: boom}}); !errors.Is(err, boom) {
		t.Errorf("error = %v, want it to wrap %v", err, boom)
	}
	if _, err := serviceAccountKey(context.Background(), keySource{}); err == nil {
		t.Error("expected an error when neither a parameter nor a file is given")
	}
}

/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scope

import (
	"os"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go/aws/session"
)

var (
	sessionCache sync.Map
)

const (
	webIdentityTokenFilePathEnvKey = "AWS_JWT_TOKEN_FILE"
)

func sessionForRegion(key string, region string, arn string) (*session.Session, error) {
	s, ok := sessionCache.Load(key)
	if ok {
		return s.(*session.Session), nil
	}
	filepath := os.Getenv(webIdentityTokenFilePathEnvKey)
	if len(filepath) == 0 {
		return nil, errors.New(" env is not set")
	}
	sn, err := session.NewSession(&aws.Config{
		Region:                        aws.String(region),
		CredentialsChainVerboseErrors: aws.Bool(true),
		LogLevel:                      aws.LogLevel(aws.LogDebug),
	})
	if err != nil {
		return nil, err
	}

	sess, err := session.NewSession(&aws.Config{
		Credentials: stscreds.NewWebIdentityCredentials(sn, arn, key, filepath),
	})
	if err != nil {
		return nil, err
	}

	sessionCache.Store(key, sess)
	return sess, nil
}

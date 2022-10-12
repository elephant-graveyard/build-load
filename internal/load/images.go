/*
Copyright © 2020 The Homeport Team

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

package load

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	corev1 "k8s.io/api/core/v1"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/gonvenience/wrap"
)

type ibmCloudIdentityToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	Expiration   int64  `json:"expiration"`
	Scope        string `json:"scope"`
}

func deleteContainerImage(kubeAccess KubeAccess, namespace string, secretRef *corev1.LocalObjectReference, imageURL string) error {
	host, org, repo, tag, err := parseImageURL(imageURL)
	if err != nil {
		return err
	}

	switch {
	case strings.Contains(host, "docker.io"):
		var token string
		if secretRef != nil {
			username, password, err := lookUpDockerCredentialsFromSecret(kubeAccess, namespace, secretRef)
			if err != nil {
				return err
			}

			token, err = dockerV2Login("hub.docker.com", username, password)
			if err != nil {
				return err
			}
		}

		return dockerV2Delete("hub.docker.com", token, org, repo, tag)

	case strings.Contains(host, "icr.io"):
		if secretRef == nil {
			return fmt.Errorf("unable to delete image %s, because no secret reference with access credentials is configured", imageURL)
		}

		username, password, err := lookUpDockerCredentialsFromSecret(kubeAccess, namespace, secretRef)
		if err != nil {
			return err
		}

		if username != "iamapikey" {
			return fmt.Errorf("failed to delete image %s, because %s/%s does not contain an IBM API key", imageURL, namespace, secretRef)
		}

		identityToken, err := getIBMCloudIdentityToken(password)
		if err != nil {
			return err
		}

		var bss string
		_, _ = jwt.Parse(identityToken.AccessToken, func(t *jwt.Token) (interface{}, error) {
			switch obj := t.Claims.(type) {
			case jwt.MapClaims:
				if account, ok := obj["account"]; ok {
					switch accountMap := account.(type) {
					case map[string]interface{}:
						switch tmp := accountMap["bss"].(type) {
						case string:
							bss = tmp
						}
					}
				}
			}

			return nil, nil
		})

		return icrDelete(*identityToken, bss, imageURL)
	}

	return nil
}

func parseImageURL(imageURL string) (string, string, string, string, error) {
	urlParts := strings.SplitN(imageURL, "/", 3)
	repoParts := strings.SplitN(urlParts[2], ":", 2)

	switch len(repoParts) {
	case 1:
		return urlParts[0], urlParts[1], urlParts[2], "latest", nil

	default:
		return urlParts[0], urlParts[1], repoParts[0], repoParts[1], nil
	}
}

func dockerV2Login(host string, username string, password string) (string, error) {
	type LoginData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	data, err := json.Marshal(LoginData{Username: username, Password: password})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://%s/v2/users/login/", host), bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		type LoginToken struct {
			Token string `json:"token"`
		}

		var loginToken LoginToken
		if err := json.Unmarshal(respData, &loginToken); err != nil {
			return "", err
		}

		return fmt.Sprintf("JWT %s", loginToken.Token), nil

	default:
		return "", fmt.Errorf(string(respData))
	}
}

func dockerV2Delete(host string, token string, org string, repo string, tag string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("https://%s/v2/repositories/%s/%s/", host, org, repo), nil)
	if err != nil {
		return err
	}

	if token != "" {
		req.Header.Set("Authorization", token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusAccepted:
		return nil

	default:
		return fmt.Errorf("failed with HTTP status code %d: %s", resp.StatusCode, string(respData))
	}
}

func getIBMCloudIdentityToken(apikey string) (*ibmCloudIdentityToken, error) {
	data := fmt.Sprintf("grant_type=%s&apikey=%s",
		url.QueryEscape("urn:ibm:params:oauth:grant-type:apikey"),
		apikey,
	)

	req, err := http.NewRequest("POST", "https://iam.cloud.ibm.com/identity/token", strings.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		var identityToken ibmCloudIdentityToken
		if err := json.Unmarshal(body, &identityToken); err != nil {
			return nil, err
		}

		return &identityToken, nil

	default:
		var responseMsg map[string]interface{}
		if err := json.Unmarshal(body, &responseMsg); err != nil {
			return nil, err
		}

		const context = "failed to obtain identity token from IAM"

		errorCode, errorCodeFound := responseMsg["errorCode"]
		errorMessage, errorMessageFound := responseMsg["errorMessage"]
		if errorCodeFound && errorMessageFound {
			return nil, wrap.Error(
				fmt.Errorf("%v: %v", errorCode, errorMessage),
				context,
			)
		}

		return nil, wrap.Error(fmt.Errorf(string(body)), context)
	}
}

func icrDelete(identityToken ibmCloudIdentityToken, accountID string, imageName string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("https://us.icr.io/api/v1/images/%s", url.QueryEscape(imageName)), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Account", accountID)
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", identityToken.TokenType, identityToken.AccessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	msg, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return nil

	default:
		return fmt.Errorf("failed to delete image %s: %s", imageName, string(msg))
	}
}

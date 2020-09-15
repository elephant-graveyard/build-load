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
	"io/ioutil"
	"net/http"
	"strings"

	buildv1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

func deleteContainerImage(kubeAccess KubeAccess, secretRef string, buildRun buildv1.BuildRun) error {
	username, password, err := lookUpDockerCredentialsFromSecret(kubeAccess, buildRun.Namespace, secretRef)
	if err != nil {
		return err
	}

	host, org, repo, tag, err := parseImageURL(buildRun.Status.BuildSpec.Output.ImageURL)
	if err != nil {
		return err
	}

	switch host {
	case "docker.io":
		token, err := dockerV2Login("hub.docker.com", username, password)
		if err != nil {
			return err
		}

		return dockerV2Delete("hub.docker.com", token, org, repo, tag)
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

	respData, err := ioutil.ReadAll(resp.Body)
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

		return loginToken.Token, nil

	default:
		return "", fmt.Errorf(string(respData))
	}
}

func dockerV2Delete(host string, token string, org string, repo string, tag string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("https://%s/v2/repositories/%s/%s/", host, org, repo), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "JWT "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	respData, err := ioutil.ReadAll(resp.Body)
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

package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestMain(m *testing.M) {
	server := NewBiedaTwitter()
	go func() {
		err := server.Start()
		if err != nil {
			panic(err)
		}
	}()
	m.Run()
	server.Stop()
}

func TestCreateTweet(t *testing.T) {
	// Given
	client := &http.Client{}
	tweet, err := json.Marshal(map[string]string{"text": "My #awesome tweet! #yolo"})
	assert.NoError(t, err)
	req, err := http.NewRequest("POST", "http://localhost:8080/tweet", bytes.NewBuffer(tweet))
	req.Header.Add("Authorization", "Basic "+basicAuth("henry", "secretpass"))

	// When
	resp, err := client.Do(req)
	assert.NoError(t, err)

	// Then
	defer resp.Body.Close()
	responseBytes, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	var responseJson map[string]interface{}
	err = json.Unmarshal(responseBytes, &responseJson)
	assert.NoError(t, err)
	assert.Contains(t, responseJson["tags"], "awesome")
	assert.Contains(t, responseJson["tags"], "yolo")
}

func TestGetTweetByTag(t *testing.T) {
	// Given
	client := &http.Client{}
	tweet, err := json.Marshal(map[string]string{"text": "My #awesome tweet! #yolo"})
	assert.NoError(t, err)
	req, err := http.NewRequest("POST", "http://localhost:8080/tweet", bytes.NewBuffer(tweet))
	req.Header.Add("Authorization", "Basic "+basicAuth("henry", "secretpass"))
	resp, err := client.Do(req)
	assert.NoError(t, err)

	// When
	resp, err = http.Get("http://localhost:8080/tweet/yolo")

	// Then
	defer resp.Body.Close()
	responseBytes, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	var responseJson map[string]interface{}
	err = json.Unmarshal(responseBytes, &responseJson)
	assert.NoError(t, err)
	assert.True(t, len(responseJson["tweets"].([]interface{})) > 1)
	assert.Regexp(t, "#yolo", responseJson["tweets"].([]interface{})[0].(map[string]interface{})["text"])
}

func TestGetNoTweetsForNonExistingTag(t *testing.T) {
	// When
	resp, err := http.Get("http://localhost:8080/tweet/noSuchTag")

	// Then
	defer resp.Body.Close()
	responseBytes, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	var responseJson map[string]interface{}
	err = json.Unmarshal(responseBytes, &responseJson)
	assert.NoError(t, err)
	assert.Len(t, responseJson["tweets"].([]interface{}), 0)
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

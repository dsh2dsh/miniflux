package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnyObject_MarshalJSON(t *testing.T) {
	testData := struct {
		SiteData *anyObject `json:"siteData,omitzero"`
	}{}

	b, err := json.Marshal(&testData)
	require.NoError(t, err)
	assert.Equal(t, "{}", string(b))

	testData.SiteData = newAnyObject(&struct {
		VideoId string `json:"videoId,omitzero"`
	}{
		VideoId: "foobar",
	})

	b, err = json.Marshal(&testData)
	require.NoError(t, err)
	assert.JSONEq(t, `{"siteData":{"videoId":"foobar"}}`, string(b))

	testData.SiteData = nil
	err = json.Unmarshal(b, &testData)
	require.NoError(t, err)
	require.NotEmpty(t, testData.SiteData.raw)
	assert.Nil(t, testData.SiteData.value)
	assert.JSONEq(t, `{"videoId":"foobar"}`, string(testData.SiteData.raw))
}

func TestAnyObject_UnmarshalJSON(t *testing.T) {
	var testData struct {
		SiteData *anyObject `json:"siteData,omitzero"`
	}
	err := json.Unmarshal([]byte(`{}`), &testData)
	require.NoError(t, err)
	assert.Empty(t, &testData)

	err = json.Unmarshal([]byte(`{"siteData":{"videoId":"foobar"}}`), &testData)
	require.NoError(t, err)
	assert.NotNil(t, testData.SiteData)
	assert.Nil(t, testData.SiteData.value)
	assert.JSONEq(t, `{"videoId":"foobar"}`, string(testData.SiteData.raw))

	b, err := json.Marshal(&testData)
	require.NoError(t, err)
	assert.JSONEq(t, `{"siteData":{"videoId":"foobar"}}`, string(b))

	var siteData struct {
		VideoId string `json:"videoId,omitempty"`
	}
	require.NoError(t, testData.SiteData.Decode(&siteData))
	assert.Equal(t, "foobar", siteData.VideoId)
}

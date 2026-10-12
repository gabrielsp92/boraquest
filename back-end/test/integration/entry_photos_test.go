package integration_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/response"
)

// jpegBytes starts with the JPEG signature, so it sniffs as image/jpeg.
var jpegBytes = append([]byte("\xff\xd8\xff\xe0"), bytes.Repeat([]byte{0}, 100)...)

// uploadPhoto posts content as the "photo" file of entry id.
func uploadPhoto(t *testing.T, baseURL, token, id string, content []byte) *http.Response {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("photo", "proof.jpg")
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/entries/"+id+"/photo", &body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { res.Body.Close() })
	return res
}

func createEntry(t *testing.T, baseURL, token string, req controllers.EntryRequest) controllers.EntryResponse {
	t.Helper()
	res := call(t, http.MethodPost, baseURL+"/api/v1/entries", token, req)
	require.Equal(t, http.StatusCreated, res.StatusCode)
	return decodeBody[controllers.EntryResponse](t, res)
}

func TestEntryPhotoAttachOnce(t *testing.T) {
	resetRules(t)
	resetEntries(t)
	srv := newServer(t)
	liaToken := login(t, srv.URL, "lia@boraquest.dev")
	r := createRule(t, srv.URL, liaToken, dailySumRuleReq)
	created := createEntry(t, srv.URL, liaToken, controllers.EntryRequest{RuleID: r.ID, MemberID: "lia"})
	assert.Empty(t, created.PhotoURL, "a new entry has no photo")

	res := uploadPhoto(t, srv.URL, liaToken, created.ID, jpegBytes)
	require.Equal(t, http.StatusOK, res.StatusCode)
	withPhoto := decodeBody[controllers.EntryResponse](t, res)
	wantURL := "https://photos.test/boraquest/familia/entries/" + created.ID + ".jpg"
	assert.Equal(t, wantURL, withPhoto.PhotoURL)
	assert.Equal(t, created.ID, withPhoto.ID)

	// Persisted and visible on both list periods.
	for _, period := range []string{"today", "week"} {
		res = call(t, http.MethodGet, srv.URL+"/api/v1/entries?period="+period, liaToken, nil)
		require.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, []controllers.EntryResponse{withPhoto}, decodeBody[controllers.EntryListResponse](t, res).Entries, period)
	}

	// A second photo is rejected and the first one stays.
	res = uploadPhoto(t, srv.URL, liaToken, created.ID, jpegBytes)
	require.Equal(t, http.StatusConflict, res.StatusCode)
	assert.Equal(t, "entry: this entry already has a photo", decodeBody[response.ErrorBody](t, res).Error)
	res = call(t, http.MethodGet, srv.URL+"/api/v1/entries?period=today", liaToken, nil)
	assert.Equal(t, wantURL, decodeBody[controllers.EntryListResponse](t, res).Entries[0].PhotoURL)
}

func TestEntryPhotoRejections(t *testing.T) {
	resetRules(t)
	resetEntries(t)
	srv := newServer(t)
	liaToken := login(t, srv.URL, "lia@boraquest.dev")
	betoToken := login(t, srv.URL, "beto@boraquest.dev")
	sumRule := createRule(t, srv.URL, liaToken, dailySumRuleReq)
	slipRule := createRule(t, srv.URL, liaToken, decreaseRuleReq)
	own := createEntry(t, srv.URL, liaToken, controllers.EntryRequest{RuleID: sumRule.ID, MemberID: "lia"})
	betos := createEntry(t, srv.URL, betoToken, controllers.EntryRequest{RuleID: sumRule.ID, MemberID: "beto"})
	slip := createEntry(t, srv.URL, liaToken, controllers.EntryRequest{RuleID: slipRule.ID, MemberID: "lia"})
	past := createEntry(t, srv.URL, liaToken, controllers.EntryRequest{RuleID: slipRule.ID, MemberID: "lia"})
	_, err := pool.Exec(context.Background(),
		`UPDATE entries SET score_type = 'sum', occurred_on = occurred_on - INTERVAL '1 day' WHERE id = $1`, past.ID)
	require.NoError(t, err)

	cases := map[string]struct {
		token   string
		id      string
		content []byte
		status  int
		message string
	}{
		"slip":            {liaToken, slip.ID, jpegBytes, http.StatusForbidden, "entry: only a completed quest can have a photo"},
		"someone else's":  {liaToken, betos.ID, jpegBytes, http.StatusForbidden, "entry: you can only add a photo to your own entry"},
		"past day":        {liaToken, past.ID, jpegBytes, http.StatusForbidden, "entry: you can only add a photo on the same day"},
		"unknown entry":   {liaToken, "ghost", jpegBytes, http.StatusNotFound, "entry not found"},
		"not an image":    {liaToken, own.ID, []byte("GIF89a not allowed"), http.StatusBadRequest, "entry: image must be jpg, png or webp, up to 8MB"},
		"empty file":      {liaToken, own.ID, nil, http.StatusBadRequest, "entry: image must be jpg, png or webp, up to 8MB"},
		"outside a guild": {login(t, srv.URL, outsiderID+"@boraquest.dev"), own.ID, jpegBytes, http.StatusForbidden, "you are not a member of any guild"},
		"no token":        {"", own.ID, jpegBytes, http.StatusUnauthorized, ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			res := uploadPhoto(t, srv.URL, tc.token, tc.id, tc.content)

			require.Equal(t, tc.status, res.StatusCode)
			if tc.message != "" {
				assert.Equal(t, tc.message, decodeBody[response.ErrorBody](t, res).Error)
			}
		})
	}

	// None of the rejected attempts attached anything.
	res := call(t, http.MethodGet, srv.URL+"/api/v1/entries?period=week", liaToken, nil)
	for _, e := range decodeBody[controllers.EntryListResponse](t, res).Entries {
		assert.Empty(t, e.PhotoURL, e.ID)
	}
}

func TestEntryPhotoRequiresFileField(t *testing.T) {
	srv := newServer(t)
	liaToken := login(t, srv.URL, "lia@boraquest.dev")

	res := call(t, http.MethodPost, srv.URL+"/api/v1/entries/any/photo", liaToken, map[string]string{"photo": "nope"})

	require.Equal(t, http.StatusBadRequest, res.StatusCode)
	assert.Equal(t, "a photo file (jpg, png or webp, up to 8MB) is required", decodeBody[response.ErrorBody](t, res).Error)
}

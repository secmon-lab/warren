package message_test

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/harlog"
	"github.com/secmon-lab/warren/pkg/domain/model/message"
)

//go:embed testdata/sns.har
var snsHar []byte

//go:embed testdata/sns.pem
var snsPem []byte

func TestSNSVerify(t *testing.T) {
	logs, err := harlog.ParseHARData(snsHar)
	gt.NoError(t, err)
	gt.A(t, logs).Length(1)

	log := logs[0]
	var msg message.SNS
	bodyData, err := io.ReadAll(log.Request.Body)
	gt.NoError(t, err)
	err = json.Unmarshal(bodyData, &msg)
	gt.NoError(t, err)

	cases := []struct {
		name    string
		msg     message.SNS
		client  message.HTTPClient
		wantErr bool
	}{
		{
			name: "valid signature",
			msg:  msg,
			client: &HTTPClientMock{
				GetFunc: func(url string) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(snsPem)),
					}, nil
				},
			},
			wantErr: false,
		},
		{
			name: "invalid signature",
			msg: message.SNS{
				Signature: "invalid",
			},
			client: &HTTPClientMock{
				GetFunc: func(url string) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(snsPem)),
					}, nil
				},
			},
			wantErr: true,
		},
		{
			name: "failed to get certificate",
			msg:  msg,
			client: &HTTPClientMock{
				GetFunc: func(url string) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.Verify(t.Context(), tc.client)
			if tc.wantErr {
				gt.Error(t, err)
			} else {
				gt.NoError(t, err)
			}
		})
	}
}
